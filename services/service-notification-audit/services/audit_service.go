package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/repositories"

	"github.com/google/uuid"
)

type AuditService interface {
	ProcessEvent(ctx context.Context, envelope *events.EventEnvelope) error
	GetAuditByCorrelation(ctx context.Context, correlationID uuid.UUID) ([]*models.AuditLog, error)
	GetRecentAudits(ctx context.Context, limit int) ([]*models.AuditLog, error)
	GetRecentNotifications(ctx context.Context, limit int) ([]*models.NotificationLog, error)
}

type auditService struct {
	auditRepo        repositories.AuditRepository
	notificationRepo repositories.NotificationRepository
	idempotencyRepo  repositories.IdempotencyRepository
}

func NewAuditService(
	auditRepo repositories.AuditRepository,
	notificationRepo repositories.NotificationRepository,
	idempotencyRepo repositories.IdempotencyRepository,
) AuditService {
	return &auditService{
		auditRepo:        auditRepo,
		notificationRepo: notificationRepo,
		idempotencyRepo:  idempotencyRepo,
	}
}

func (s *auditService) ProcessEvent(ctx context.Context, envelope *events.EventEnvelope) error {
	// 1. Verificar idempotencia
	processed, err := s.idempotencyRepo.IsMessageProcessed(ctx, envelope.MessageID)
	if err != nil {
		return fmt.Errorf("error verificando idempotencia: %w", err)
	}
	if processed {
		log.Printf("[AuditService] Mensaje %s ya procesado previamente. Omitiendo duplicado.", envelope.MessageID)
		return nil
	}

	// 2. Registrar en Auditoría General
	auditLog := &models.AuditLog{
		ID:            uuid.New(),
		EventID:       envelope.MessageID,
		CorrelationID: envelope.CorrelationID,
		CausationID:   envelope.CausationID,
		EventType:     envelope.Type,
		Producer:      envelope.Producer,
		Version:       envelope.Version,
		Payload:       envelope.Payload,
		OccurredAt:    envelope.OccurredAt,
		RecordedAt:    time.Now().UTC(),
	}

	if err := s.auditRepo.SaveAuditLog(ctx, auditLog); err != nil {
		return fmt.Errorf("error guardando audit log: %w", err)
	}

	// 3. Procesamiento específico de Notificaciones según tipo de evento
	s.handleNotificationDispatch(ctx, envelope)

	// 4. Marcar mensaje como procesado
	return s.idempotencyRepo.MarkMessageProcessed(ctx, envelope.MessageID, "notification-audit-consumer", envelope.Type)
}

func (s *auditService) handleNotificationDispatch(ctx context.Context, envelope *events.EventEnvelope) {
	switch envelope.Type {
	case "notification.activation-email.requested":
		var p events.ActivationEmailPayload
		if err := json.Unmarshal(envelope.Payload, &p); err == nil {
			log.Printf("=========================================================")
			log.Printf("[SIMULACIÓN DE CORREO]")
			log.Printf("Para: %s (%s)", p.Email, p.FullName)
			log.Printf("Asunto: Activa tu cuenta en Bank USAC")
			log.Printf("Enlace: %s", p.ActivationLink)
			log.Printf("Expira: %s", p.ExpiresAt.Format(time.RFC3339))
			log.Printf("=========================================================")

			_ = s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
				ID:               uuid.New(),
				CorrelationID:    envelope.CorrelationID,
				Recipient:        p.Email,
				NotificationType: "ACTIVATION_EMAIL",
				Subject:          "Activa tu cuenta en Bank USAC",
				BodySummary:      fmt.Sprintf("Enlace de activación enviado para %s", p.FullName),
				Status:           models.NotificationSent,
				SentAt:           time.Now().UTC(),
			})
		}

	case "transfer.completed":
		_ = s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
			ID:               uuid.New(),
			CorrelationID:    envelope.CorrelationID,
			Recipient:        "accounts-involved",
			NotificationType: "TRANSFER_ALERT",
			Subject:          "Transferencia bancaria procesada con éxito",
			BodySummary:      "La transferencia ha sido completada satisfactoriamente.",
			Status:           models.NotificationSent,
			SentAt:           time.Now().UTC(),
		})
	}
}

func (s *auditService) GetAuditByCorrelation(ctx context.Context, correlationID uuid.UUID) ([]*models.AuditLog, error) {
	return s.auditRepo.GetAuditLogsByCorrelationID(ctx, correlationID)
}

func (s *auditService) GetRecentAudits(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.auditRepo.GetRecentAuditLogs(ctx, limit)
}

func (s *auditService) GetRecentNotifications(ctx context.Context, limit int) ([]*models.NotificationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.notificationRepo.GetNotificationLogs(ctx, limit)
}
