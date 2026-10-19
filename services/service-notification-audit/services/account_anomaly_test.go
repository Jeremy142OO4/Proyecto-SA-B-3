package services

import (
	"bank-usac/service-notification-audit/models"
	"context"
	"github.com/google/uuid"
	"strings"
	"testing"
)

type anomalyEmail struct {
	body    string
	subject string
	sent    int
}

func (s *anomalyEmail) Send(to, subject, body string) error {
	s.sent++
	s.body = body
	s.subject = subject
	return nil
}
func TestAccountAnomalyWarningEmailAndInboxAreIdempotent(t *testing.T) {
	audit := &fakeAuditRepository{}
	notifications := &fakeNotificationRepository{}
	processed := &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}
	email := &anomalyEmail{}
	svc := NewAuditService(audit, notifications, processed, email)
	id := uuid.New()
	envelope := newTestEnvelope("AccountAnomalyDetected", map[string]any{"idCuenta": id, "idCliente": uuid.New(), "correo": "test@example.test", "operaciones": 11, "maxOperaciones": 10, "ventanaMinutos": 5, "tipoCuenta": "AHORRO", "nivel": "WARNING"})
	for i := 0; i < 2; i++ {
		if err := svc.ProcessEvent(context.Background(), envelope); err != nil {
			t.Fatal(err)
		}
	}
	if len(audit.logs) != 1 || audit.logs[0].Severity != models.EventWarning || len(notifications.logs) != 1 || email.sent != 1 {
		t.Fatal("anomaly must be WARNING and idempotent")
	}
	n := notifications.logs[0]
	if n.Severity != models.EventWarning || n.AccountID == nil || *n.AccountID != id || len(n.Details) == 0 {
		t.Fatal("inbox metadata missing")
	}
	if !strings.Contains(email.subject, "actividad inusual") || !strings.Contains(email.body, "11") || !strings.Contains(email.body, "Límite configurado") || !strings.Contains(email.body, "Comprobante informativo") {
		t.Fatal("email does not follow the existing receipt structure")
	}
}

func TestBlockedAccountEmailIncludesReasonAndHistoryAccess(t *testing.T) {
	email := &anomalyEmail{}
	svc := NewAuditService(&fakeAuditRepository{}, &fakeNotificationRepository{}, &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}, email)
	reason := "Se han detectado múltiples transacciones en poco tiempo."
	envelope := newTestEnvelope("cuenta.bloqueada", map[string]any{"idCuenta": uuid.New(), "idCliente": uuid.New(), "correo": "test@example.test", "motivo": reason})
	if err := svc.ProcessEvent(context.Background(), envelope); err != nil {
		t.Fatal(err)
	}
	if email.sent != 1 || !strings.Contains(email.body, reason) || !strings.Contains(email.body, "historial") || !strings.Contains(email.body, "BLOQUEADA") {
		t.Fatal("blocked-account receipt must explain reason and read-only access")
	}
}

func TestTransferNotificationHasTransactionID(t *testing.T) {
	notifications := &fakeNotificationRepository{}
	email := &anomalyEmail{}
	svc := NewAuditService(&fakeAuditRepository{}, notifications, &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}, email)
	id := uuid.New()
	envelope := newTestEnvelope("transferencia.completada", map[string]any{"idTransferencia": id, "idCliente": uuid.New(), "correo": "test@example.test", "idCuentaOrigen": uuid.New(), "idCuentaDestino": uuid.New()})
	if e := svc.ProcessEvent(context.Background(), envelope); e != nil {
		t.Fatal(e)
	}
	if email.sent != 1 || !strings.Contains(email.body, "id_transaccion") || !strings.Contains(email.body, id.String()) {
		t.Fatal("transaction ID missing from notification receipt")
	}
	if len(notifications.logs) != 1 || !strings.Contains(string(notifications.logs[0].Details), id.String()) {
		t.Fatal("transaction ID missing from notification details")
	}
}

func TestPaymentCompensationFailureIsRecordedAndNotified(t *testing.T) {
	audit := &fakeAuditRepository{}
	notifications := &fakeNotificationRepository{}
	email := &anomalyEmail{}
	svc := NewAuditService(audit, notifications, &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}, email)
	envelope := newTestEnvelope("pago.compensacion.fallida", map[string]any{"idPago": uuid.New(), "idCliente": uuid.New(), "correo": "test@example.test", "estado": "COMPENSACION_FALLIDA", "codigo": "COMPENSACION_RECHAZADA"})
	if err := svc.ProcessEvent(context.Background(), envelope); err != nil {
		t.Fatal(err)
	}
	if len(audit.logs) != 1 || audit.logs[0].Severity != models.EventError || len(notifications.logs) != 1 || notifications.logs[0].NotificationType != "PAYMENT_COMPENSATION_FAILED" || email.sent != 1 {
		t.Fatal("compensation failure must be audited and notified as error")
	}
}
