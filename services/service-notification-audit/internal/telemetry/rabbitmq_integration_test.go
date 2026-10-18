package telemetry

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
	"os"
	"testing"
	"time"
)

func TestRabbitMQTraceIntegration(t *testing.T) {
	url := os.Getenv("TEST_RABBITMQ_URL")
	if url == "" {
		t.Skip("isolated TEST_RABBITMQ_URL required")
	}
	tp, _ := tracing(t)
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := tp.Tracer("test").Start(context.Background(), "real broker")
	defer root.End()
	body := []byte(`{"idMensaje":"00000000-0000-4000-8000-000000000001","idCorrelacion":"00000000-0000-4000-8000-000000000002"}`)
	_, h, end := Publish(ctx, amqp.Table{"x-intentos": int32(2)}, body, "cuenta.compensacion.solicitada")
	err = ch.PublishWithContext(ctx, "", q.Name, false, false, amqp.Publishing{Headers: h, Body: body})
	end(err)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case d := <-deliveries:
		if string(d.Body) != string(body) || d.Headers["x-intentos"] != int32(2) {
			t.Fatal("message changed")
		}
		consumed, finish := Consume(context.Background(), d)
		defer finish(nil)
		if trace.SpanContextFromContext(consumed).TraceID() != root.SpanContext().TraceID() {
			t.Fatal("trace split across real RabbitMQ")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("broker delivery timeout")
	}
}
