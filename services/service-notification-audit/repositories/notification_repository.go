package repositories

import (
	"context"

	"bank-usac/service-notification-audit/models"

	"github.com/jmoiron/sqlx"
)

type notificationRepo struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) SaveNotificationLog(ctx context.Context, log *models.NotificationLog) error {
	query := `
		INSERT INTO notification_logs (id, correlation_id, recipient, notification_type, subject, body_summary, status, error_detail, sent_at)
		VALUES (:id, :correlation_id, :recipient, :notification_type, :subject, :body_summary, :status, :error_detail, :sent_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, log)
	return err
}

func (r *notificationRepo) GetNotificationLogs(ctx context.Context, limit int) ([]*models.NotificationLog, error) {
	var logs []*models.NotificationLog
	query := `SELECT * FROM notification_logs ORDER BY sent_at DESC LIMIT $1`
	err := r.db.SelectContext(ctx, &logs, query, limit)
	return logs, err
}
