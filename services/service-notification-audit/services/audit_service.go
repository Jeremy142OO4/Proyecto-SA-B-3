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
	emailSender      EmailSender
}

func NewAuditService(
	auditRepo repositories.AuditRepository,
	notificationRepo repositories.NotificationRepository,
	idempotencyRepo repositories.IdempotencyRepository,
	emailSender EmailSender,
) AuditService {
	return &auditService{
		auditRepo:        auditRepo,
		notificationRepo: notificationRepo,
		idempotencyRepo:  idempotencyRepo,
		emailSender:      emailSender,
	}
}

func (s *auditService) ProcessEvent(ctx context.Context, envelope *events.EventEnvelope) error {
	processed, err := s.idempotencyRepo.IsMessageProcessed(ctx, envelope.MessageID)
	if err != nil {
		return fmt.Errorf("error verificando idempotencia: %w", err)
	}

	if processed {
		log.Printf(
			"[notification-audit-service] mensaje duplicado omitido: messageId=%s correlationId=%s",
			envelope.MessageID,
			envelope.CorrelationID,
		)
		return nil
	}

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

	// Un error de notificación no revierte ni bloquea el evento de negocio.
	s.handleNotificationDispatch(ctx, envelope)

	if err := s.idempotencyRepo.MarkMessageProcessed(
		ctx,
		envelope.MessageID,
		"notification-audit-consumer",
		envelope.Type,
	); err != nil {
		return fmt.Errorf("error marcando mensaje como procesado: %w", err)
	}

	return nil
}

func (s *auditService) handleNotificationDispatch(
	ctx context.Context,
	envelope *events.EventEnvelope,
) {
	switch envelope.Type {
	case "notificacion.correo-activacion.solicitado":
		s.sendActivationEmail(ctx, envelope)

	case "transferencia.completada":
		if err := s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
			ID:               uuid.New(),
			CorrelationID:    envelope.CorrelationID,
			Recipient:        "accounts-involved",
			NotificationType: "TRANSFER_ALERT",
			Subject:          "Transferencia bancaria procesada con éxito",
			BodySummary:      "La transferencia ha sido completada satisfactoriamente.",
			Status:           models.NotificationSent,
			SentAt:           time.Now().UTC(),
		}); err != nil {
			log.Printf(
				"[notification-audit-service] error registrando alerta de transferencia: correlationId=%s error=%v",
				envelope.CorrelationID,
				err,
			)
		}
	}
}

func (s *auditService) sendActivationEmail(
	ctx context.Context,
	envelope *events.EventEnvelope,
) {
	var payload events.ActivationEmailPayload

	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		log.Printf(
			"[notification-audit-service] payload inválido para correo de activación: messageId=%s correlationId=%s error=%v",
			envelope.MessageID,
			envelope.CorrelationID,
			err,
		)
		return
	}

	subject := "Activa tu cuenta en Bank USAC"

	body := fmt.Sprintf(
		"Hola %s,\n\n"+
			"Tu cuenta de Bank USAC ha sido registrada correctamente.\n\n"+
			"Para activarla, utiliza el siguiente enlace:\n%s\n\n"+
			"El enlace vence el: %s\n\n"+
			"Si no solicitaste este registro, ignora este correo.\n\n"+
			"Bank USAC",
		payload.FullName,
		payload.ActivationLink,
		payload.ExpiresAt.Format(time.RFC1123),
	)

	if s.emailSender == nil {
		log.Printf(
			"[notification-audit-service] SMTP no configurado; correo no enviado: messageId=%s correlationId=%s",
			envelope.MessageID,
			envelope.CorrelationID,
		)
		return
	}

	if err := s.emailSender.Send(payload.Email, subject, body); err != nil {
		log.Printf(
			"[notification-audit-service] fallo al enviar correo de activación: messageId=%s correlationId=%s recipient=%s error=%v",
			envelope.MessageID,
			envelope.CorrelationID,
			payload.Email,
			err,
		)
		return
	}

	if err := s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
		ID:               uuid.New(),
		CorrelationID:    envelope.CorrelationID,
		Recipient:        payload.Email,
		NotificationType: "ACTIVATION_EMAIL",
		Subject:          subject,
		BodySummary:      fmt.Sprintf("Correo de activación enviado para %s", payload.FullName),
		Status:           models.NotificationSent,
		SentAt:           time.Now().UTC(),
	}); err != nil {
		log.Printf(
			"[notification-audit-service] correo enviado, pero no se pudo registrar la notificación: correlationId=%s error=%v",
			envelope.CorrelationID,
			err,
		)
		return
	}

	log.Printf(
		"[notification-audit-service] correo de activación enviado: messageId=%s correlationId=%s recipient=%s",
		envelope.MessageID,
		envelope.CorrelationID,
		payload.Email,
	)
}

func (s *auditService) GetAuditByCorrelation(
	ctx context.Context,
	correlationID uuid.UUID,
) ([]*models.AuditLog, error) {
	return s.auditRepo.GetAuditLogsByCorrelationID(ctx, correlationID)
}

func (s *auditService) GetRecentAudits(
	ctx context.Context,
	limit int,
) ([]*models.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.auditRepo.GetRecentAuditLogs(ctx, limit)
}

func (s *auditService) GetRecentNotifications(
	ctx context.Context,
	limit int,
) ([]*models.NotificationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.notificationRepo.GetNotificationLogs(ctx, limit)
}
