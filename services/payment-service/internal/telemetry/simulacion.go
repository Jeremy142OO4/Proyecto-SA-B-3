package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

func ProveedorSimulado(ctx context.Context, id, result string, version int64, delay int) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("payment.id", id), attribute.Bool("external.simulated", true), attribute.String("external.outcome", result), attribute.Int64("simulation.version", version), attribute.Int("simulation.delay_ms", delay))
	if result != "EXITO" {
		span.SetStatus(codes.Error, "controlled simulated external failure")
		span.AddEvent("external.simulated_failure")
		slog.WarnContext(ctx, "simulated external error", "paymentId", id, "simulationVersion", version, "outcome", result, "errorCode", "SIMULATED_EXTERNAL_"+result)
	}
}
func ProveedorConfirmado(ctx context.Context, result string, seconds float64) {
	outcome := "success"
	if result == "FALLO" {
		outcome = "failure"
	}
	if result == "TIMEOUT" {
		outcome = "timeout"
	}
	meter := otel.Meter("bank-usac")
	opts := metric.WithAttributes(append(metricIdentity, attribute.String("outcome", outcome))...)
	count, _ := meter.Int64Counter("bank_payment_external_results", metric.WithDescription("Committed logical simulated provider results"))
	if count != nil {
		count.Add(ctx, 1, opts)
	}
	latency, _ := meter.Float64Histogram("bank_payment_external_duration", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(buckets...))
	if latency != nil && seconds >= 0 {
		latency.Record(ctx, seconds, opts)
	}
}
func PagoEstado(ctx context.Context, id, state string, accountIDs ...string) {
	_, span := otel.Tracer("bank-usac").Start(ctx, "pago.estado", trace.WithAttributes(attribute.String("payment.id", id), attribute.String("payment.state", state)))
	span.AddEvent(state)
	span.End()
	attrs := []any{"paymentId", id, "state", state}
	if len(accountIDs) > 0 {
		attrs = append(attrs, "accountId", accountIDs[0])
	}
	slog.InfoContext(ctx, "payment state committed", attrs...)
}
