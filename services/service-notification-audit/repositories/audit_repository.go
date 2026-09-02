package repositories

import (
	"context"

	"bank-usac/service-notification-audit/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type auditRepo struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) AuditRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) SaveAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, event_id, correlation_id, causation_id, event_type, producer, version, payload, occurred_at, recorded_at)
		VALUES (:id, :event_id, :correlation_id, :causation_id, :event_type, :producer, :version, :payload, :occurred_at, :recorded_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, log)
	return err
}

func (r *auditRepo) GetAuditLogsByCorrelationID(ctx context.Context, correlationID uuid.UUID) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	query := `SELECT * FROM audit_logs WHERE correlation_id = $1 ORDER BY occurred_at ASC`
	err := r.db.SelectContext(ctx, &logs, query, correlationID)
	return logs, err
}

func (r *auditRepo) GetRecentAuditLogs(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	query := `SELECT * FROM audit_logs ORDER BY occurred_at DESC LIMIT $1`
	err := r.db.SelectContext(ctx, &logs, query, limit)
	return logs, err
}
