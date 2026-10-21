package telemetry

import (
	"context"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"time"
)

func initializeTransferResults(ctx context.Context) {
	counter, err := otel.Meter("bank-usac").Int64Counter("bank_transfer_results", metric.WithDescription("Committed terminal transfer outcomes"))
	if err != nil {
		return
	}
	for _, outcome := range []string{"success", "failure"} {
		labels := append([]attribute.KeyValue{}, metricIdentity...)
		labels = append(labels, attribute.String("outcome", outcome))
		counter.Add(ctx, 0, metric.WithAttributes(labels...))
	}
}

func Phase2State(state models.Estado) string {
	switch state {
	case models.Completada:
		return "APPROVED"
	case models.Rechazada, models.Compensada, models.CompensacionFallida:
		return "FAILED"
	default:
		return "PENDING"
	}
}

func StartTransfer(ctx context.Context, id, correlation string) (context.Context, func()) {
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "transferencia.inicio", trace.WithAttributes(attribute.String("transaction.id", id)))
	ctx = WithCorrelation(ctx, correlation)
	return ctx, func() { span.End() }
}

// Called only after a real state change has committed, never for duplicates.
func TransferCommitted(ctx context.Context, id, correlation, saved string, state models.Estado, elapsed time.Duration, accountIDs ...string) {
	original := Restore(ctx, saved)
	if sc := trace.SpanContextFromContext(original); sc.IsValid() && sc.TraceID() != trace.SpanContextFromContext(ctx).TraceID() {
		ctx = original
	}
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "transferencia.estado")
	defer span.End()
	ctx = WithCorrelation(ctx, correlation)
	phase := Phase2State(state)
	attrs := []attribute.KeyValue{attribute.String("transaction.id", id), attribute.String("transaction.state", string(state)), attribute.String("transaction.phase2.state", phase)}
	span.SetAttributes(attrs...)
	span.AddEvent(phase, trace.WithAttributes(attrs...))
	logAttrs := []any{"transactionId", id, "state", string(state), "phase2State", phase}
	if len(accountIDs) > 0 {
		logAttrs = append(logAttrs, "accountId", accountIDs[0])
	}
	slog.InfoContext(ctx, "transfer state committed", logAttrs...)
	if phase == "PENDING" {
		return
	}
	outcome := "success"
	if phase == "FAILED" {
		outcome = "failure"
		span.SetStatus(codes.Error, "transfer failed")
	}
	meter := otel.Meter("bank-usac")
	options := metric.WithAttributes(append(metricIdentity, attribute.String("outcome", outcome))...)
	results, _ := meter.Int64Counter("bank_transfer_results", metric.WithDescription("Committed terminal transfer outcomes"))
	if results != nil {
		results.Add(ctx, 1, options)
	}
	duration, _ := meter.Float64Histogram("bank_transfer_flow_duration", metric.WithUnit("s"), metric.WithDescription("Creation through terminal Saga result, including compensation and broker waits"), metric.WithExplicitBucketBoundaries(buckets...))
	if duration != nil && elapsed >= 0 {
		duration.Record(ctx, elapsed.Seconds(), options)
	}
}
