package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log/slog"
)

// Export a baseline before the first movement so a fresh deployment has real
// zero series and Prometheus can observe the first subsequent increment.
func initializeAccountUsage(ctx context.Context) {
	counter, err := otel.Meter("bank-usac").Int64Counter("bank_account_movements")
	if err != nil {
		return
	}
	anomalies, err := otel.Meter("bank-usac").Int64Counter("bank_account_anomalies")
	if err != nil {
		return
	}
	for _, accountType := range []string{"AHORRO", "CORRIENTE", "MONETARIA"} {
		labels := append([]attribute.KeyValue{}, metricIdentity...)
		labels = append(labels, attribute.String("account_type", accountType))
		anomalies.Add(ctx, 0, metric.WithAttributes(labels...))
		for _, direction := range []string{"DEBITO", "CREDITO"} {
			labels := append([]attribute.KeyValue{}, metricIdentity...)
			labels = append(labels, attribute.String("account_type", accountType), attribute.String("direction", direction))
			counter.Add(ctx, 0, metric.WithAttributes(labels...))
		}
	}
}

func AccountMovementCommitted(ctx context.Context, accountType, operation, accountID string, anomaly bool) {
	slog.InfoContext(ctx, "account movement committed", "accountId", accountID, "accountType", accountType, "operation", operation)
	if operation == "DEBITO" || operation == "CREDITO" {
		counter, _ := otel.Meter("bank-usac").Int64Counter("bank_account_movements")
		labels := append([]attribute.KeyValue{}, metricIdentity...)
		labels = append(labels, attribute.String("account_type", accountType), attribute.String("direction", operation))
		counter.Add(ctx, 1, metric.WithAttributes(labels...))
	}
	if anomaly {
		slog.WarnContext(ctx, "high account operation frequency detected", "accountId", accountID, "eventType", "AccountAnomalyDetected")
		counter, _ := otel.Meter("bank-usac").Int64Counter("bank_account_anomalies")
		labels := append([]attribute.KeyValue{}, metricIdentity...)
		labels = append(labels, attribute.String("account_type", accountType))
		counter.Add(ctx, 1, metric.WithAttributes(labels...))
	}
}
