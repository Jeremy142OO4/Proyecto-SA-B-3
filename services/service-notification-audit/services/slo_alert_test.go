package services

import (
	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

func TestSLOAlertCreatesOneAdminNotificationAndKeepsTrace(t *testing.T) {
	audit := &fakeAuditRepository{}
	notifications := &fakeNotificationRepository{}
	processed := &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}
	svc := NewAuditService(audit, notifications, processed, nil)
	alert := models.SLOAlert{AlertID: "transfer-error-budget", Service: "transaction-service", Severity: models.EventError, Summary: "Error budget agotado", StartedAt: time.Now().UTC(), TraceID: strings.Repeat("a", 32)}
	for i := 0; i < 2; i++ {
		if e := svc.ReceiveSLOAlert(context.Background(), alert); e != nil {
			t.Fatal(e)
		}
	}
	if len(audit.logs) != 1 || len(notifications.logs) != 1 {
		t.Fatal("duplicate alert was not deduplicated")
	}
	row := audit.logs[0]
	n := notifications.logs[0]
	if row.TraceID != alert.TraceID || row.Severity != models.EventError || n.Severity != models.EventError || n.NotificationType != "SLO_ALERT" || n.Status != models.NotificationPending || n.Recipient != "ADMIN" || len(n.Details) == 0 {
		t.Fatal("missing alert trace or admin notification")
	}
	// A critical escalation must not disappear under the warning's identity.
	alert.Severity = models.EventWarning
	if e := svc.ReceiveSLOAlert(context.Background(), alert); e != nil {
		t.Fatal(e)
	}
	if len(notifications.logs) != 2 {
		t.Fatal("severity transition suppressed")
	}
}
func TestSLOBrokerEventAndOptionalEmail(t *testing.T) {
	for _, recipient := range []string{"ops@example.test", "not an email"} {
		audit := &fakeAuditRepository{}
		notifications := &fakeNotificationRepository{}
		email := &anomalyEmail{}
		svc := NewAuditService(audit, notifications, &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}, email, recipient)
		alert := models.SLOAlert{AlertID: "latency", Service: "payment-service", Severity: models.EventWarning, Summary: "<script>slow</script>", StartedAt: time.Now().UTC()}
		payload, _ := json.Marshal(alert)
		for i := 0; i < 2; i++ {
			e := svc.ProcessEvent(context.Background(), &events.EventEnvelope{MessageID: uuid.New(), CorrelationID: uuid.New(), Type: "SLOAlertTriggered", Version: 1, Producer: "slo-monitor", OccurredAt: time.Now(), Payload: payload})
			if e != nil {
				t.Fatal(e)
			}
		}
		if len(notifications.logs) != 1 {
			t.Fatal("different transport IDs duplicated one alert")
		}
		if recipient == "ops@example.test" {
			if email.sent != 1 || notifications.logs[0].Status != models.NotificationSent || strings.Contains(email.body, "<script>") {
				t.Fatal("unsafe or missing SLO email")
			}
		} else {
			if email.sent != 0 || notifications.logs[0].Status != models.NotificationFailed {
				t.Fatal("invalid email must retain inbox notification")
			}
		}
	}
}
func TestInvalidSLODoesNotCreateNotification(t *testing.T) {
	notifications := &fakeNotificationRepository{}
	svc := NewAuditService(&fakeAuditRepository{}, notifications, &fakeIdempotencyRepository{processed: map[uuid.UUID]bool{}}, nil)
	if e := svc.ReceiveSLOAlert(context.Background(), models.SLOAlert{}); e == nil {
		t.Fatal("invalid SLO accepted")
	}
	if len(notifications.logs) != 0 {
		t.Fatal("invalid SLO notified")
	}
}
