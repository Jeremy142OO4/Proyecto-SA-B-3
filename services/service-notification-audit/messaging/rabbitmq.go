package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/services"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	auditSvc services.AuditService
}

func NewRabbitMQConsumer(url string, auditSvc services.AuditService) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// 1. Declarar Exchanges
	if err := ch.ExchangeDeclare("banco.eventos", "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare("banco.comandos", "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare("banco.respuestas", "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare("banco.fallidos", "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}

	// 2. Declarar Cola propia y durable para Notification & Audit
	queueName := "notification-audit.events.q"
	args := amqp.Table{
		"x-dead-letter-exchange":    "banco.fallidos",
		"x-dead-letter-routing-key": "notification-audit.dlq",
	}
	q, err := ch.QueueDeclare(queueName, true, false, false, false, args)
	if err != nil {
		return nil, err
	}

	// 3. Bindings para capturar todos los eventos del sistema
	routingKeys := []string{
		"cliente.#",
		"cuenta.#",
		"transferencia.#",
		"pago.#",
		"notificacion.#",
	}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(q.Name, rk, "banco.eventos", false, nil); err != nil {
			return nil, err
		}
	}
	if _, err := ch.QueueDeclare("notification-audit.commands.q", true, false, false, false, args); err != nil {
		return nil, err
	}
	for _, rk := range []string{events.ComandoRegistros, events.ComandoTraza, events.ComandoNotificaciones} {
		if err := ch.QueueBind("notification-audit.commands.q", rk, "banco.comandos", false, nil); err != nil {
			return nil, err
		}
	}

	return &RabbitMQConsumer{conn: conn, channel: ch, auditSvc: auditSvc}, nil
}

func (r *RabbitMQConsumer) StartConsuming(ctx context.Context) error {
	msgs, err := r.channel.Consume(
		"notification-audit.events.q",
		"notification-audit-worker",
		false, // Manual Ack obligatorio por SRE
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	comandos, err := r.channel.Consume("notification-audit.commands.q", "notification-audit-commands", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		log.Println("[RabbitMQ] Consumidor de Notification & Audit iniciado. Escuchando eventos...")
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				r.handleDelivery(ctx, msg)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-comandos:
				if !ok {
					return
				}
				r.handleCommand(ctx, msg)
			}
		}
	}()

	return nil
}

type respuestaRPC struct {
	Estado int             `json:"estado"`
	Cuerpo json.RawMessage `json:"cuerpo"`
}

func (r *RabbitMQConsumer) handleCommand(ctx context.Context, d amqp.Delivery) {
	var sobre events.EventEnvelope
	if err := json.Unmarshal(d.Body, &sobre); err != nil || sobre.MessageID == uuid.Nil || sobre.CorrelationID == uuid.Nil {
		_ = d.Nack(false, false)
		return
	}
	estado, cuerpo := r.executeCommand(ctx, sobre)
	contenido, _ := json.Marshal(respuestaRPC{Estado: estado, Cuerpo: cuerpo})
	respuesta := events.EventEnvelope{MessageID: uuid.New(), CorrelationID: sobre.CorrelationID, CausationID: &sobre.MessageID, Type: strings.TrimSuffix(strings.TrimSuffix(sobre.Type, ".solicitados"), ".solicitada") + ".respondida", Version: 1, OccurredAt: time.Now().UTC(), Producer: "notification-audit-service", Payload: contenido}
	bytes, _ := json.Marshal(respuesta)
	if err := r.channel.PublishWithContext(ctx, "banco.respuestas", respuesta.Type, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", CorrelationId: sobre.CorrelationID.String(), Body: bytes}); err != nil {
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func (r *RabbitMQConsumer) executeCommand(ctx context.Context, sobre events.EventEnvelope) (int, json.RawMessage) {
	responder := func(estado int, valor any) (int, json.RawMessage) { b, _ := json.Marshal(valor); return estado, b }
	fallar := func(err error) (int, json.RawMessage) { return responder(500, map[string]string{"error": err.Error()}) }
	switch sobre.Type {
	case events.ComandoRegistros:
		var req struct {
			Limite int `json:"limite"`
		}
		_ = json.Unmarshal(sobre.Payload, &req)
		registros, err := r.auditSvc.GetRecentAudits(ctx, req.Limite)
		if err != nil {
			return fallar(err)
		}
		return responder(200, registros)
	case events.ComandoTraza:
		var req struct {
			IDCorrelacion uuid.UUID `json:"idCorrelacion"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil || req.IDCorrelacion == uuid.Nil {
			return responder(400, map[string]string{"error": "idCorrelacion invalido"})
		}
		registros, err := r.auditSvc.GetAuditByCorrelation(ctx, req.IDCorrelacion)
		if err != nil {
			return fallar(err)
		}
		return responder(200, registros)
	case events.ComandoNotificaciones:
		var req struct {
			Limite int `json:"limite"`
		}
		_ = json.Unmarshal(sobre.Payload, &req)
		notificaciones, err := r.auditSvc.GetRecentNotifications(ctx, req.Limite)
		if err != nil {
			return fallar(err)
		}
		return responder(200, notificaciones)
	default:
		return fallar(errors.New("comando de auditoria no soportado"))
	}
}

func (r *RabbitMQConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) {
	var envelope events.EventEnvelope
	if err := json.Unmarshal(d.Body, &envelope); err != nil {
		log.Printf("[RabbitMQ] Error deserializando sobre de mensaje: %v. Enviando a DLQ.", err)
		_ = d.Nack(false, false) // Fallo permanente -> DLQ
		return
	}

	if err := r.auditSvc.ProcessEvent(ctx, &envelope); err != nil {
		log.Printf("[RabbitMQ] Error procesando evento %s (MessageID: %s): %v", envelope.Type, envelope.MessageID, err)
		// En caso de error no permanente, se podría reintentar o enviar a DLQ según política
		_ = d.Nack(false, false)
		return
	}

	// Confirmación manual de recepción exitosa
	_ = d.Ack(false)
}

func (r *RabbitMQConsumer) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
