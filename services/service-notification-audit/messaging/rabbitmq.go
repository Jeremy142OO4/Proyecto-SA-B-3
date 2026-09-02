package messaging

import (
	"context"
	"encoding/json"
	"log"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/services"

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
		"cliente.*",
		"cuenta.*",
		"transferencia.*",
		"pago.*",
		"notificacion.*",
	}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(q.Name, rk, "banco.eventos", false, nil); err != nil {
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

	return nil
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
