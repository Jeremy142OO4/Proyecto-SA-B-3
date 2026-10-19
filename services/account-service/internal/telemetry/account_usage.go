package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log/slog"
)

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
