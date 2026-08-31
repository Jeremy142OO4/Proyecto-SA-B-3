package messaging

import (
	"context"
	"log"
	"time"

	"bank-usac/service-customer/repositories"

	amqp "github.com/rabbitmq/amqp091-go"
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
	exchanges := []string{"bank.commands", "bank.events", "bank.dlx"}
	for _, ex := range exchanges {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return nil, err
		}
	}

	return &RabbitMQClient{conn: conn, channel: ch, repo: repo}, nil
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
		exchange := "bank.events"
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
