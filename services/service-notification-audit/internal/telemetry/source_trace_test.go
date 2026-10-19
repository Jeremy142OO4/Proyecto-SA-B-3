package telemetry

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestAuditSourceTraceDoesNotUseANewConsumerTrace(t *testing.T) {
	tp, _ := tracing(t)
	ctx, root := tp.Tracer("test").Start(context.Background(), "producer")
	headers := AMQPHeaders(ctx)
	original := root.SpanContext().TraceID().String()
	root.End()
	with, finish := Consume(context.Background(), amqp.Delivery{Headers: headers, RoutingKey: "cuenta.creada"})
	if SourceTraceID(with) != original {
		t.Fatal("producer trace not preserved")
	}
	finish(nil)
	missing, end := Consume(context.Background(), amqp.Delivery{RoutingKey: "cuenta.creada"})
	defer end(nil)
	if !trace.SpanContextFromContext(missing).IsValid() {
		t.Fatal("test consumer trace not generated")
	}
	if SourceTraceID(missing) != "" {
		t.Fatal("new consumer trace falsely assigned to uninstrumented business event")
	}
}
