// Package bankrpc carries internal requests over RabbitMQ. Business handlers must be idempotent.
package bankrpc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/account-service/internal/telemetry"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type Response struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func Reply(status int, body any) Response { b, _ := json.Marshal(body); return Response{status, b} }

type Handler func(context.Context, json.RawMessage) Response

// Each call owns its channel and exclusive reply queue; replies cannot cross between replicas.
func Call(ctx context.Context, url, queue string, body any) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second, Dial: amqp.DefaultDial(3 * time.Second)})
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return Response{}, err
	}
	defer ch.Close()
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return Response{}, err
	}
	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		return Response{}, err
	}
	if err = ch.Confirm(false); err != nil {
		return Response{}, err
	}
	returned := ch.NotifyReturn(make(chan amqp.Return, 1))
	b, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	id := uuid.NewString()
	pubctx, headers, finish := telemetry.Publish(ctx, nil, b, queue)
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(pubctx, "", queue, true, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, CorrelationId: id, ReplyTo: q.Name, Expiration: "10000", Headers: headers, Body: b})
	if err == nil {
		var ok bool
		ok, err = confirmation.WaitContext(ctx)
		if err == nil && !ok {
			err = errors.New("broker rejected request")
		}
	}
	finish(err)
	if err != nil {
		return Response{}, err
	}
	for {
		select {
		case <-returned:
			return Response{}, errors.New("service queue unavailable")
		case d, ok := <-deliveries:
			if !ok {
				return Response{}, errors.New("RPC connection closed")
			}
			if d.CorrelationId == id {
				var r Response
				err = json.Unmarshal(d.Body, &r)
				return r, err
			}
		case <-ctx.Done():
			return Response{}, ctx.Err()
		}
	}
}

// Run reconnects after broker restarts. A failed reply requeues the request;
// database idempotency, not delivery count, prevents a repeated financial effect.
func Run(ctx context.Context, url, queue string, handler Handler) {
	for ctx.Err() == nil {
		_ = serve(ctx, url, queue, handler)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
func serve(ctx context.Context, url, queue string, handler Handler) error {
	conn, err := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second, Dial: amqp.DefaultDial(3 * time.Second)})
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err = ch.Qos(8, 0, false); err != nil {
		return err
	}
	if err = ch.Confirm(false); err != nil {
		return err
	}
	deliveries, err := ch.ConsumeWithContext(ctx, queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("consumer closed")
			}
			requestctx, finish := telemetry.Consume(ctx, d)
			requestctx, cancel := context.WithTimeout(requestctx, 8*time.Second)
			result := handler(requestctx, d.Body)
			cancel()
			b, _ := json.Marshal(result)
			replyctx, headers, end := telemetry.Publish(ctx, d.Headers, b, queue+".reply")
			confirm, e := ch.PublishWithDeferredConfirmWithContext(replyctx, "", d.ReplyTo, false, false, amqp.Publishing{CorrelationId: d.CorrelationId, Headers: headers, Body: b, ContentType: "application/json"})
			if e == nil {
				var confirmed bool
				confirmed, e = confirm.WaitContext(ctx)
				if e == nil && !confirmed {
					e = errors.New("reply not confirmed")
				}
			}
			end(e)
			finish(e)
			if e != nil {
				_ = d.Nack(false, true)
				return e
			}
			if e = d.Ack(false); e != nil {
				return e
			}
		}
	}
}
