package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/internal/telemetry"
	"bank-usac/service-customer/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// saveLifecycle writes a redacted, durable record and the existing domain event
// in the SAME transaction as the customer change. RPC/Saga request correlation
// IDs remain untouched; only lifecycle events use the registration correlation.
func saveLifecycle(ctx context.Context, tx *sqlx.Tx, customer *models.Customer, event *models.OutboxMessage, eventType, kind, previous, state string) error {
	if customer.LifecycleCorrelationID == nil {
		return errors.New("customer lifecycle correlation is required")
	}
	id := *customer.LifecycleCorrelationID
	var occurredAt time.Time
	if err := tx.QueryRowContext(ctx, `INSERT INTO customer_lifecycle_events
		(id,customer_id,event_type,state_kind,previous_state,state,correlation_id,trace_context)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING occurred_at`, uuid.New(), customer.CustomerID, eventType, kind, previous, state, id, telemetry.Serialize(ctx)).Scan(&occurredAt); err != nil {
		return err
	}
	if event == nil {
		return nil
	}
	var envelope events.EventEnvelope
	if err := json.Unmarshal(event.Payload, &envelope); err != nil {
		return err
	}
	envelope.CorrelationID = id
	envelope.OccurredAt = occurredAt
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO outbox_messages
		(id,event_type,payload,correlation_id,causation_id,created_at,trace_context)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, event.ID, event.EventType, payload, id, event.CausationID, occurredAt, telemetry.Serialize(ctx))
	return err
}

// Existing customers acquire an identity on their first observed change without
// fabricating a registration event or a historical verification timestamp.
func ensureLifecycle(ctx context.Context, tx *sqlx.Tx, customer *models.Customer, event *models.OutboxMessage) error {
	if customer.LifecycleCorrelationID != nil {
		return nil
	}
	id := uuid.New()
	if event != nil && event.CorrelationID != uuid.Nil {
		id = event.CorrelationID
	}
	customer.LifecycleCorrelationID = &id
	customer.LifecycleTraceContext = telemetry.Serialize(ctx)
	_, err := tx.ExecContext(ctx, `UPDATE customers SET lifecycle_correlation_id=$1,lifecycle_trace_context=$2 WHERE customer_id=$3`, id, customer.LifecycleTraceContext, customer.CustomerID)
	return err
}

// RegisterCustomerKYCMetric exposes real current customer counts after restarts,
// including pre-existing customers. Failed reads produce no fabricated zeros.
func RegisterCustomerKYCMetric(db *sqlx.DB) (func(), error) {
	meter := otel.Meter("bank-usac")
	gauge, err := meter.Int64ObservableGauge("bank_kyc_customers", metric.WithDescription("Current customers by KYC status"))
	if err != nil {
		return nil, err
	}
	registration, err := meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		var rows []struct {
			Status string `db:"kyc_status"`
			Count  int64  `db:"count"`
		}
		if err := db.SelectContext(ctx, &rows, `SELECT kyc_status,COUNT(*) FROM customers WHERE role='CLIENTE' GROUP BY kyc_status`); err != nil {
			return err
		}
		counts := map[string]int64{"PENDING": 0, "VERIFIED": 0, "REJECTED": 0}
		for _, row := range rows {
			counts[row.Status] = row.Count
		}
		for _, status := range []string{"PENDING", "VERIFIED", "REJECTED"} {
			observer.ObserveInt64(gauge, counts[status], metric.WithAttributes(append(telemetry.CustomerMetricAttributes(), attribute.String("status", status))...))
		}
		return nil
	}, gauge)
	if err != nil {
		return nil, err
	}
	return func() { _ = registration.Unregister() }, nil
}
