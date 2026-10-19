package services

import (
	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"html"
	"net/mail"
	"strings"
	"time"
)

func sloEnvelope(a models.SLOAlert) (*events.EventEnvelope, error) {
	if err := a.Normalize(); err != nil {
		return nil, err
	}
	id := a.EventID()
	corr := uuid.NewSHA1(uuid.NameSpaceOID, []byte("slo-correlation:"+id.String()))
	if a.CorrelationID != nil {
		corr = *a.CorrelationID
	}
	payload, _ := json.Marshal(a)
	return &events.EventEnvelope{MessageID: id, CorrelationID: corr, Type: "SLOAlertTriggered", Version: 1, OccurredAt: a.StartedAt.UTC(), Producer: "slo-monitor", Payload: payload}, nil
}
func (s *auditService) ReceiveSLOAlert(ctx context.Context, a models.SLOAlert) error {
	e, err := sloEnvelope(a)
	if err != nil {
		return err
	}
	return s.ProcessEvent(ctx, e)
}
func (s *auditService) notifySLO(ctx context.Context, e *events.EventEnvelope) error {
	var alert models.SLOAlert
	if err := json.Unmarshal(e.Payload, &alert); err != nil {
		return err
	}
	if err := alert.Normalize(); err != nil {
		return err
	}
	status := models.NotificationPending
	recipient := "ADMIN"
	var detail *string
	subject := "Alerta SLO - " + alert.Service
	if configured := strings.TrimSpace(s.sloRecipient); configured != "" {
		parsed, err := mail.ParseAddress(configured)
		if err != nil || parsed.Address != configured {
			msg := "SLO_NOTIFICATION_RECIPIENT invalido"
			detail = &msg
			status = models.NotificationFailed
		} else {
			recipient = configured
			status = models.NotificationFailed
			if s.emailSender != nil {
				body := "<h2>Alerta SLO</h2><p>" + html.EscapeString(alert.Summary) + "</p><p>Servicio: " + html.EscapeString(alert.Service) + "</p><p>Nivel: " + string(alert.Severity) + "</p><p>Alerta: " + html.EscapeString(alert.AlertID) + "</p><p>Correlación: " + e.CorrelationID.String() + "</p>"
				if err := s.emailSender.Send(recipient, subject, body); err != nil {
					msg := "No fue posible enviar el correo SLO"
					detail = &msg
				} else {
					status = models.NotificationSent
				}
			} else {
				msg := "Proveedor de correo no configurado"
				detail = &msg
			}
		}
	}
	return s.notificationRepo.SaveNotificationLog(ctx, &models.NotificationLog{ID: idempotentID("notification", e.MessageID), CorrelationID: e.CorrelationID, Recipient: recipient, NotificationType: "SLO_ALERT", Severity: alert.Severity, Subject: subject, BodySummary: alert.Summary, Status: status, ErrorDetail: detail, Details: e.Payload, SentAt: time.Now().UTC()})
}
