package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"

	"github.com/google/uuid"
)

type fakeAuditRepository struct {
	logs []*models.AuditLog
}

func (f *fakeAuditRepository) SaveAuditLog(_ context.Context, log *models.AuditLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func (f *fakeAuditRepository) GetAuditLogsByCorrelationID(_ context.Context, _ uuid.UUID) ([]*models.AuditLog, error) {
	return f.logs, nil
}

func (f *fakeAuditRepository) GetRecentAuditLogs(_ context.Context, _ int) ([]*models.AuditLog, error) {
	return f.logs, nil
}

type fakeNotificationRepository struct {
	logs []*models.NotificationLog
}

func (f *fakeNotificationRepository) SaveNotificationLog(_ context.Context, log *models.NotificationLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func (f *fakeNotificationRepository) GetNotificationLogs(_ context.Context, _ models.NotificationFilter) ([]*models.NotificationLog, error) {
	return f.logs, nil
}

type fakeIdempotencyRepository struct {
	processed map[uuid.UUID]bool
}

func (f *fakeIdempotencyRepository) IsMessageProcessed(_ context.Context, messageID uuid.UUID) (bool, error) {
	return f.processed[messageID], nil
}

func (f *fakeIdempotencyRepository) MarkMessageProcessed(_ context.Context, messageID uuid.UUID, _, _ string) error {
	f.processed[messageID] = true
	return nil
}

type fakeEmailSender struct {
	err error
}

func (f fakeEmailSender) Send(_, _, _ string) error {
	return f.err
}

func TestClassifyEvent(t *testing.T) {
	tests := []struct {
		eventType string
		want      models.EventSeverity
	}{
		{eventType: "transferencia.completada", want: models.EventInfo},
		{eventType: "pago.rechazado", want: models.EventWarning},
		{eventType: "transferencia.compensacion.fallida", want: models.EventError},
		{eventType: "notification-audit.dlq", want: models.EventError},
	}

	for _, test := range tests {
		if got := ClassifyEvent(test.eventType); got != test.want {
			t.Errorf("ClassifyEvent(%q) = %q, want %q", test.eventType, got, test.want)
		}
	}
}

func TestProcessEventStoresSeverityAndGeneratedNotification(t *testing.T) {
	auditRepo := &fakeAuditRepository{}
	notificationRepo := &fakeNotificationRepository{}
	idempotencyRepo := &fakeIdempotencyRepository{processed: make(map[uuid.UUID]bool)}
	service := NewAuditService(auditRepo, notificationRepo, idempotencyRepo, nil)

	envelope := newTestEnvelope("transferencia.rechazada", map[string]string{"idCuenta": uuid.NewString()})
	if err := service.ProcessEvent(context.Background(), envelope); err != nil {
		t.Fatalf("ProcessEvent() error = %v", err)
	}

	if len(auditRepo.logs) != 1 || auditRepo.logs[0].Severity != models.EventWarning {
		t.Fatalf("la auditoría no conservó la severidad WARNING: %+v", auditRepo.logs)
	}
	if len(notificationRepo.logs) != 1 || notificationRepo.logs[0].NotificationType != "TRANSFER_REJECTED" {
		t.Fatalf("no se generó la notificación esperada: %+v", notificationRepo.logs)
	}
}

func TestProcessEventIsIdempotentAndKeepsCorrelation(t *testing.T) {
	auditRepo := &fakeAuditRepository{}
	notificationRepo := &fakeNotificationRepository{}
	idempotencyRepo := &fakeIdempotencyRepository{processed: make(map[uuid.UUID]bool)}
	service := NewAuditService(auditRepo, notificationRepo, idempotencyRepo, nil)
	envelope := newTestEnvelope("transferencia.completada", map[string]string{"estado": "COMPLETADA"})

	if err := service.ProcessEvent(context.Background(), envelope); err != nil {
		t.Fatalf("primer procesamiento fallido: %v", err)
	}
	if err := service.ProcessEvent(context.Background(), envelope); err != nil {
		t.Fatalf("reprocesamiento idempotente fallido: %v", err)
	}
	if len(auditRepo.logs) != 1 || len(notificationRepo.logs) != 1 {
		t.Fatalf("el evento duplicado genero efectos adicionales: auditorias=%d notificaciones=%d", len(auditRepo.logs), len(notificationRepo.logs))
	}
	if auditRepo.logs[0].CorrelationID != envelope.CorrelationID || notificationRepo.logs[0].CorrelationID != envelope.CorrelationID {
		t.Fatal("se perdio el CorrelationId en los registros")
	}
}

func TestProcessEventRechazaSobreInvalido(t *testing.T) {
	service := NewAuditService(&fakeAuditRepository{}, &fakeNotificationRepository{}, &fakeIdempotencyRepository{processed: make(map[uuid.UUID]bool)}, nil)
	if err := service.ProcessEvent(context.Background(), &events.EventEnvelope{MessageID: uuid.New(), Type: "transferencia.completada"}); err == nil {
		t.Fatal("se esperaba error para sobre sin CorrelationId")
	}
}

func TestActivationEmailFailureIsRecorded(t *testing.T) {
	notificationRepo := &fakeNotificationRepository{}
	idempotencyRepo := &fakeIdempotencyRepository{processed: make(map[uuid.UUID]bool)}
	service := NewAuditService(
		&fakeAuditRepository{},
		notificationRepo,
		idempotencyRepo,
		fakeEmailSender{err: errors.New("smtp unavailable")},
	)

	envelope := newTestEnvelope("notificacion.correo-activacion.solicitado", events.ActivationEmailPayload{
		Email:          "cliente@example.com",
		FullName:       "Cliente de prueba",
		ActivationLink: "https://example.com/activar",
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	})
	if err := service.ProcessEvent(context.Background(), envelope); err != nil {
		t.Fatalf("ProcessEvent() error = %v", err)
	}

	if len(notificationRepo.logs) != 1 {
		t.Fatalf("se esperaba un registro de notificación, se obtuvieron %d", len(notificationRepo.logs))
	}
	if notificationRepo.logs[0].Status != models.NotificationFailed {
		t.Fatalf("estado de notificación = %q, want FAILED", notificationRepo.logs[0].Status)
	}
	if notificationRepo.logs[0].ErrorDetail == nil {
		t.Fatal("se esperaba el detalle controlado del error SMTP")
	}
}

func newTestEnvelope(eventType string, payload any) *events.EventEnvelope {
	bytes, _ := json.Marshal(payload)
	return &events.EventEnvelope{
		MessageID:     uuid.New(),
		CorrelationID: uuid.New(),
		Type:          eventType,
		Version:       1,
		OccurredAt:    time.Now().UTC(),
		Producer:      "test",
		Payload:       bytes,
	}
}
