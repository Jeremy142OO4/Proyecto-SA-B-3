package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
	GetNotifications(ctx context.Context, filter models.NotificationFilter) ([]*models.NotificationLog, error)
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
		Severity:      ClassifyEvent(envelope.Type),
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
	default:
		rule, ok := notificationRuleFor(envelope.Type)
		if !ok {
			return
		}

		recipient := extractRecipient(envelope.Payload, rule.defaultRecipient)
		status := models.NotificationSent
		if err := s.saveGeneratedNotification(ctx, envelope, rule, recipient, status, ""); err != nil {
			log.Printf("[notification-audit-service] error registrando notificacion: correlationId=%s error=%v", envelope.CorrelationID, err)
		}
	}
}

type notificationRule struct {
	notificationType string
	subject          string
	bodySummary      string
	defaultRecipient  string
}

func notificationRuleFor(eventType string) (notificationRule, bool) {
	rules := map[string]notificationRule{
		"cliente.creado":                    {"CLIENT_CREATED", "Registro de cliente recibido", "El cliente fue registrado correctamente.", "customer"},
		"cliente.activado":                  {"CLIENT_ACTIVATED", "Cliente activado", "El cliente fue activado correctamente.", "customer"},
		"cliente.rechazado":                 {"CLIENT_VALIDATION_REJECTED", "Validación de cliente rechazada", "El cliente no pudo validarse para la operación solicitada.", "customer"},
		"cliente.kyc.verificado":            {"KYC_VERIFIED", "Validación KYC aprobada", "La validación KYC fue aprobada.", "customer"},
		"cliente.kyc.rechazado":             {"KYC_REJECTED", "Validación KYC rechazada", "La validación KYC fue rechazada.", "customer"},
		"cliente.kyc.estado.actualizado":    {"KYC_UPDATED", "Estado KYC actualizado", "El estado KYC fue actualizado.", "customer"},
		"cuenta.creada":                     {"ACCOUNT_CREATED", "Cuenta creada", "La cuenta bancaria fue creada correctamente.", "account-owner"},
		"cuenta.creacion.rechazada":         {"ACCOUNT_REJECTED", "Creación de cuenta rechazada", "La solicitud de cuenta no pudo completarse.", "account-owner"},
		"cuenta.debitada":                   {"ACCOUNT_DEBITED", "Débito aplicado", "El débito fue aplicado correctamente.", "account-owner"},
		"cuenta.debito.rechazado":           {"ACCOUNT_DEBIT_REJECTED", "Débito rechazado", "El débito no pudo aplicarse.", "account-owner"},
		"cuenta.acreditada":                 {"ACCOUNT_CREDITED", "Crédito aplicado", "El crédito fue aplicado correctamente.", "account-owner"},
		"cuenta.credito.rechazado":          {"ACCOUNT_CREDIT_REJECTED", "Crédito rechazado", "El crédito no pudo aplicarse.", "account-owner"},
		"cuenta.compensada":                 {"ACCOUNT_COMPENSATED", "Cuenta compensada", "La operación fue compensada correctamente.", "account-owner"},
		"cuenta.compensacion.rechazada":     {"ACCOUNT_COMPENSATION_REJECTED", "Compensación rechazada", "La compensación no pudo aplicarse.", "account-owner"},
		"cuenta.desactivada":                {"ACCOUNT_DEACTIVATED", "Cuenta desactivada", "La cuenta fue desactivada por inactividad.", "account-owner"},
		"cuenta.transferencia.rechazada":    {"ACCOUNT_TRANSFER_REJECTED", "Validación de transferencia rechazada", "La transferencia no cumplió las reglas de las cuentas.", "accounts-involved"},
		"transferencia.completada":          {"TRANSFER_COMPLETED", "Transferencia completada", "La transferencia fue completada satisfactoriamente.", "accounts-involved"},
		"transferencia.rechazada":           {"TRANSFER_REJECTED", "Transferencia rechazada", "La transferencia fue rechazada sin aplicar cambios definitivos.", "accounts-involved"},
		"transferencia.compensando":         {"TRANSFER_COMPENSATING", "Transferencia en compensación", "La transferencia está siendo compensada.", "accounts-involved"},
		"transferencia.compensada":          {"TRANSFER_COMPENSATED", "Transferencia compensada", "La transferencia fue compensada y los fondos regresaron a la cuenta origen.", "accounts-involved"},
		"transferencia.compensacion.fallida": {"TRANSFER_COMPENSATION_FAILED", "Error al compensar transferencia", "La compensación requiere revisión administrativa.", "accounts-involved"},
		"pago.completado":                   {"PAYMENT_COMPLETED", "Pago completado", "El pago fue procesado correctamente.", "customer"},
		"pago.rechazado":                    {"PAYMENT_REJECTED", "Pago rechazado", "El pago no pudo completarse.", "customer"},
	}
	rule, ok := rules[eventType]
	return rule, ok
}

func extractRecipient(payload json.RawMessage, fallback string) string {
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err == nil {
		for _, key := range []string{"correo", "email", "recipient", "destinatario", "idCliente", "customerId", "idCuenta", "accountId"} {
			if value, ok := fields[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return fallback
}

func (s *auditService) saveGeneratedNotification(
	ctx context.Context,
	envelope *events.EventEnvelope,
	rule notificationRule,
	recipient string,
	status models.NotificationStatus,
	errorDetail string,
) error {
	return s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
		ID:               uuid.New(),
		CorrelationID:    envelope.CorrelationID,
		Recipient:        recipient,
		NotificationType: rule.notificationType,
		Subject:          rule.subject,
		BodySummary:      rule.bodySummary,
		Status:           status,
		ErrorDetail:      optionalString(errorDetail),
		SentAt:           time.Now().UTC(),
	})
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

// ClassifyEvent follows the project convention: successful and in-progress
// events are INFO, recoverable business outcomes are WARNING, and operational
// failures or dead-lettered messages are ERROR.
func ClassifyEvent(eventType string) models.EventSeverity {
	typo := strings.ToLower(strings.TrimSpace(eventType))
	if strings.Contains(typo, "fallid") || strings.Contains(typo, "timeout") || strings.Contains(typo, "dlq") {
		return models.EventError
	}
	if strings.Contains(typo, "rechaz") || strings.Contains(typo, "compensando") || strings.Contains(typo, "compensada") {
		return models.EventWarning
	}
	return models.EventInfo
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
		detail := "payload inválido para correo de activación"
		if saveErr := s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
			ID:               uuid.New(),
			CorrelationID:    envelope.CorrelationID,
			Recipient:        "unknown",
			NotificationType: "ACTIVATION_EMAIL",
			Subject:          "Activa tu cuenta en Bank USAC",
			BodySummary:      "No se pudo generar el correo de activación.",
			Status:           models.NotificationFailed,
			ErrorDetail:      &detail,
			SentAt:           time.Now().UTC(),
		}); saveErr != nil {
			log.Printf("[notification-audit-service] no se pudo registrar payload inválido: correlationId=%s error=%v", envelope.CorrelationID, saveErr)
		}
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

	status := models.NotificationSent
	var errorDetail string
	if s.emailSender == nil {
		status = models.NotificationFailed
		errorDetail = "SMTP no configurado"
	} else if err := s.emailSender.Send(payload.Email, subject, body); err != nil {
		status = models.NotificationFailed
		errorDetail = err.Error()
		log.Printf(
			"[notification-audit-service] fallo al enviar correo de activación: messageId=%s correlationId=%s recipient=%s error=%v",
			envelope.MessageID,
			envelope.CorrelationID,
			payload.Email,
			err,
		)
	}

	if err := s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{
		ID:               uuid.New(),
		CorrelationID:    envelope.CorrelationID,
		Recipient:        payload.Email,
		NotificationType: "ACTIVATION_EMAIL",
		Subject:          subject,
		BodySummary:      fmt.Sprintf("Correo de activación para %s", payload.FullName),
		Status:           status,
		ErrorDetail:      optionalString(errorDetail),
		SentAt:           time.Now().UTC(),
	}); err != nil {
		log.Printf(
			"[notification-audit-service] no se pudo registrar el resultado de notificación: correlationId=%s error=%v",
			envelope.CorrelationID,
			err,
		)
	}

	if status == models.NotificationSent {
		log.Printf("[notification-audit-service] correo de activación enviado: messageId=%s correlationId=%s recipient=%s", envelope.MessageID, envelope.CorrelationID, payload.Email)
	}
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

func (s *auditService) GetNotifications(
	ctx context.Context,
	filter models.NotificationFilter,
) ([]*models.NotificationLog, error) {
	return s.notificationRepo.GetNotificationLogs(ctx, filter)
}
