package repositories

import (
	"context"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"
	"time"
)

func commitEstado(ctx context.Context, tx pgx.Tx, id uuid.UUID) (bool, error) {
	var state models.Estado
	var correlation uuid.UUID
	var saved string
	var seconds float64
	var accountID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT estado,id_correlacion,trace_context,EXTRACT(EPOCH FROM (clock_timestamp()-fecha_creacion))::float8,id_cuenta_origen FROM transferencias WHERE id_transferencia=$1`, id).Scan(&state, &correlation, &saved, &seconds, &accountID)
	if err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	telemetry.TransferCommitted(ctx, id.String(), correlation.String(), saved, state, time.Duration(seconds*float64(time.Second)), accountID.String())
	return true, nil
}

// Keep outgoing Outbox messages on the creation trace even when an upstream
// response arrives with missing or unrelated trace headers.
func resumeTransfer(ctx context.Context, tx pgx.Tx, id uuid.UUID) (context.Context, error) {
	var saved string
	var correlation uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT trace_context,id_correlacion FROM transferencias WHERE id_transferencia=$1`, id).Scan(&saved, &correlation); err != nil {
		return ctx, err
	}
	restored := telemetry.Restore(ctx, saved)
	if original := trace.SpanContextFromContext(restored); original.IsValid() && original.TraceID() != trace.SpanContextFromContext(ctx).TraceID() {
		ctx = restored
	}
	return telemetry.WithCorrelation(ctx, correlation.String()), nil
}
