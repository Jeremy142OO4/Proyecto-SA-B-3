package telemetry

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// StartCustomerLifecycle resumes the registration trace across requests and
// restarts. The current request stays reachable as a span link; its cancellation
// and deadline are preserved. No customer payload becomes a span attribute.
func StartCustomerLifecycle(ctx context.Context, saved, name string, correlationID uuid.UUID) (context.Context, trace.Span) {
	request := trace.SpanContextFromContext(ctx)
	ctx = Restore(ctx, saved)
	options := []trace.SpanStartOption{trace.WithAttributes(attribute.String("correlationId", correlationID.String()))}
	if request.IsValid() && !request.Equal(trace.SpanContextFromContext(ctx)) {
		options = append(options, trace.WithLinks(trace.Link{SpanContext: request}))
	}
	ctx, span := otel.Tracer("bank-usac").Start(ctx, name, options...)
	return WithCorrelation(ctx, correlationID.String()), span
}

// CustomerLifecycleCommitted must only run AFTER the business transaction commits.
// IDs belong in logs/traces, never in metric labels. KYC checks made by a Saga
// do not call this function: they are not new KYC decisions.
func CustomerLifecycleCommitted(ctx context.Context, customerID uuid.UUID, eventType, kind, previous, state string, verificationDuration *time.Duration) {
	attrs := []any{"customerId", customerID.String(), "eventType", eventType,
		"stateKind", kind, "previousState", previous, "state", state}
	if state == "REJECTED" || state == "BLOQUEADO" {
		slog.WarnContext(ctx, "customer lifecycle changed", attrs...)
	} else {
		slog.InfoContext(ctx, "customer lifecycle changed", attrs...)
	}
	span := trace.SpanFromContext(ctx)
	span.AddEvent(eventType, trace.WithAttributes(attribute.String("customer.state.kind", kind),
		attribute.String("customer.state.previous", previous), attribute.String("customer.state", state)))
	if kind == "KYC" && (state == "VERIFIED" || state == "REJECTED") {
		counter, _ := otel.Meter("bank-usac").Int64Counter("bank_kyc_decisions",
			metric.WithDescription("Committed KYC state changes, excluding repeated identical decisions"))
		labels := append([]attribute.KeyValue{}, metricIdentity...)
		labels = append(labels, attribute.String("outcome", strings.ToLower(state)))
		counter.Add(ctx, 1, metric.WithAttributes(labels...))
	}
	if verificationDuration != nil && *verificationDuration >= 0 {
		histogram, _ := otel.Meter("bank-usac").Float64Histogram("bank_kyc_duration",
			metric.WithUnit("s"), metric.WithDescription("Time from registration to first observed KYC verification"),
			metric.WithExplicitBucketBoundaries(1, 10, 60, 300, 900, 3600, 21600, 86400, 259200, 604800, 2592000))
		histogram.Record(ctx, verificationDuration.Seconds(), metric.WithAttributes(append(CustomerMetricAttributes(), attribute.String("outcome", "verified"))...))
	}
}

// CustomerMetricAttributes returns a copy so callbacks cannot mutate shared labels.
func CustomerMetricAttributes() []attribute.KeyValue {
	return append([]attribute.KeyValue{}, metricIdentity...)
}
