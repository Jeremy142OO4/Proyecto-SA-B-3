package repositories

import (
	"context"
	"strconv"
	"strings"

	"bank-usac/service-notification-audit/models"

	"github.com/google/uuid"
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
		ON CONFLICT (id) DO NOTHING
	`
	_, err := r.db.NamedExecContext(ctx, query, log)
	return err
}

func (r *notificationRepo) GetNotificationLogs(ctx context.Context, filter models.NotificationFilter) ([]*models.NotificationLog, error) {
	var logs []*models.NotificationLog
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 4)
	argNumber := 1

	if strings.TrimSpace(filter.Recipient) != "" {
		conditions = append(conditions, "recipient = $"+strconv.Itoa(argNumber))
		args = append(args, strings.TrimSpace(filter.Recipient))
		argNumber++
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = $"+strconv.Itoa(argNumber))
		args = append(args, filter.Status)
		argNumber++
	}
	if filter.CorrelationID != nil && *filter.CorrelationID != uuid.Nil {
		conditions = append(conditions, "correlation_id = $"+strconv.Itoa(argNumber))
		args = append(args, *filter.CorrelationID)
		argNumber++
	}

	query := `SELECT * FROM notification_logs`
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY sent_at DESC LIMIT $` + strconv.Itoa(argNumber)
	args = append(args, normalizeLimit(filter.Limit))

	err := r.db.SelectContext(ctx, &logs, query, args...)
	return logs, err
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}
