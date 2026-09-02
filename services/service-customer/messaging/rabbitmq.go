package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"bank-usac/service-customer/repositories"
	"bank-usac/service-customer/events"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	intercambioComandos = "banco.comandos"
	intercambioEventos  = "banco.eventos"
	intercambioFallidos = "banco.fallidos"
	colaComandosCliente = "clientes.comandos"
	colaFallidosCliente = "clientes.comandos.fallidos"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	repo    repositories.CustomerRepository
}

func NewRabbitMQClient(url string, repo repositories.CustomerRepository) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declarar Exchanges estándar
	exchanges := []string{intercambioComandos, intercambioEventos, intercambioFallidos}
	for _, ex := range exchanges {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return nil, err
		}
	}
	if _, err := ch.QueueDeclare(colaFallidosCliente, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.QueueBind(colaFallidosCliente, "cliente.#", intercambioFallidos, false, nil); err != nil {
		return nil, err
	}
	argumentos := amqp.Table{
		"x-dead-letter-exchange": intercambioFallidos,
		"x-dead-letter-routing-key": "cliente.validacion.fallida",
	}
	if _, err := ch.QueueDeclare(colaComandosCliente, true, false, false, false, argumentos); err != nil {
		return nil, err
	}
	if err := ch.QueueBind(colaComandosCliente, events.ComandoValidarCliente, intercambioComandos, false, nil); err != nil {
		return nil, err
	}

	return &RabbitMQClient{conn: conn, channel: ch, repo: repo}, nil
}

func (r *RabbitMQClient) StartValidationConsumer(ctx context.Context) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	if err = ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		return err
	}
	entregas, err := ch.ConsumeWithContext(ctx, colaComandosCliente, "customer-service.validacion-cliente", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return err
	}
	go func() {
		defer ch.Close()
		for entrega := range entregas {
			var sobre events.EventEnvelope
			if json.Unmarshal(entrega.Body, &sobre) != nil || sobre.MessageID == uuid.Nil || sobre.CorrelationID == uuid.Nil {
				_ = entrega.Nack(false, false)
				continue
			}
			var solicitud events.SolicitudValidacionCliente
			if json.Unmarshal(sobre.Payload, &solicitud) != nil || solicitud.IDSolicitud == uuid.Nil || solicitud.IDCliente == uuid.Nil {
				_ = entrega.Nack(false, false)
				continue
			}
			if err := r.repo.RegistrarValidacionCliente(ctx, sobre.MessageID, sobre.CorrelationID, solicitud.IDSolicitud, solicitud.IDCliente); err != nil {
				log.Printf("[ValidacionCliente] Error: %v", err)
				_ = entrega.Nack(false, false)
				continue
			}
			_ = entrega.Ack(false)
		}
	}()
	return nil
}

func (r *RabbitMQClient) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// StartOutboxWorker envía periódicamente los eventos pendientes a RabbitMQ
func (r *RabbitMQClient) StartOutboxWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.processPendingOutbox(ctx)
			}
		}
	}()
}

func (r *RabbitMQClient) processPendingOutbox(ctx context.Context) {
	msgs, err := r.repo.GetPendingOutbox(ctx, 20)
	if err != nil || len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		exchange := intercambioEventos
		routingKey := msg.EventType

		err := r.channel.PublishWithContext(
			ctx,
			exchange,
			routingKey,
			false,
			false,
			amqp.Publishing{
				DeliveryMode:  amqp.Persistent,
				ContentType:   "application/json",
				CorrelationId: msg.CorrelationID.String(),
				Timestamp:     msg.CreatedAt,
				Body:          msg.Payload,
			},
		)

		if err != nil {
			log.Printf("[Outbox] Error publicando evento %s: %v", msg.EventType, err)
			_ = r.repo.IncrementOutboxAttempt(ctx, msg.ID, err.Error())
		} else {
			_ = r.repo.MarkOutboxPublished(ctx, msg.ID)
		}
	}
}
