package repositories

import (
	"context"

	"bank-usac/service-notification-audit/models"

	"github.com/google/uuid"
)

type AuditRepository interface {
	SaveAuditLog(ctx context.Context, log *models.AuditLog) error
	GetAuditLogsByCorrelationID(ctx context.Context, correlationID uuid.UUID) ([]*models.AuditLog, error)
	GetRecentAuditLogs(ctx context.Context, limit int) ([]*models.AuditLog, error)
}

type NotificationRepository interface {
	SaveNotificationLog(ctx context.Context, log *models.NotificationLog) error
	GetNotificationLogs(ctx context.Context, limit int) ([]*models.NotificationLog, error)
}

type IdempotencyRepository interface {
	IsMessageProcessed(ctx context.Context, messageID uuid.UUID) (bool, error)
	MarkMessageProcessed(ctx context.Context, messageID uuid.UUID, consumerName, ref string) error
}
