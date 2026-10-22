package telemetry

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestRabbitMQSemanticsKeepW3CParentAndCorrelation(t *testing.T) {
	tp, exporter := tracing(t)
	ctx, root := tp.Tracer("test").Start(context.Background(), "request")
	body := []byte(`{"idMensaje":"00000000-0000-4000-8000-000000000001","idCorrelacion":"00000000-0000-4000-8000-000000000002"}`)
	producer, headers, finish := Publish(ctx, nil, body, "transferencia.solicitada")
	finish(nil)
	if _, ok := headers["traceparent"].(string); !ok {
		t.Fatal("missing W3C traceparent header")
	}
	consumer, done := Consume(context.Background(), amqp.Delivery{Headers: headers, Body: body, RoutingKey: "transferencia.solicitada", Exchange: "banco.comandos"})
	done(nil)
	if trace.SpanContextFromContext(producer).TraceID() != trace.SpanContextFromContext(consumer).TraceID() {
		t.Fatal("producer/consumer trace split")
	}
	root.End()
	count := 0
	for _, span := range exporter.GetSpans() {
		if span.SpanKind != trace.SpanKindProducer && span.SpanKind != trace.SpanKindConsumer {
			continue
		}
		count++
		attrs := map[string]string{}
		for _, a := range span.Attributes {
			if a.Value.Type() == attribute.STRING {
				attrs[string(a.Key)] = a.Value.AsString()
			}
		}
		if attrs["messaging.system"] != "rabbitmq" || attrs["messaging.rabbitmq.destination.routing_key"] != "transferencia.solicitada" || attrs["correlationId"] != "00000000-0000-4000-8000-000000000002" {
			t.Fatalf("semantic identity missing: %v", attrs)
		}
		if span.SpanKind == trace.SpanKindProducer && attrs["messaging.operation.type"] != "send" {
			t.Fatal("wrong producer operation")
		}
		if span.SpanKind == trace.SpanKindConsumer {
			if attrs["messaging.operation.type"] != "process" || attrs["messaging.destination.name"] != "banco.comandos" {
				t.Fatal("wrong consumer semantics")
			}
			if span.Parent.SpanID() != trace.SpanContextFromContext(producer).SpanID() {
				t.Fatal("consumer is not a child of publisher")
			}
		}
	}
	if count != 2 {
		t.Fatalf("expected producer and consumer spans, got %d", count)
	}
}
