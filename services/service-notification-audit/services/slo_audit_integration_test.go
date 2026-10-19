package services

import (
	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/internal/telemetry"
	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/repositories"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditAndSLONotificationsDatabase(t *testing.T) {
	address := os.Getenv("TEST_AUDIT_DATABASE_URL")
	if address == "" {
		t.Skip("isolated TEST_AUDIT_DATABASE_URL required")
	}
	admin, e := sqlx.Connect("postgres", address)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := "audit_service_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec("CREATE SCHEMA " + schema); e != nil {
		t.Fatal(e)
	}
	defer admin.Exec("DROP SCHEMA " + schema + " CASCADE")
	parsed, _ := url.Parse(address)
	q := parsed.Query()
	q.Set("search_path", schema+",public")
	parsed.RawQuery = q.Encode()
	db, e := sqlx.Connect("postgres", parsed.String())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := filepath.Glob("../database/migrations/*.up.sql")
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if _, e = db.Exec(string(b)); e != nil {
			t.Fatal(e)
		}
	}
	repo := repositories.NewAuditRepository(db)
	notifications := repositories.NewNotificationRepository(db)
	svc := NewAuditService(repo, notifications, repositories.NewIdempotencyRepository(db), nil)
	old := otel.GetTracerProvider()
	provider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(provider)
	defer func() { provider.Shutdown(context.Background()); otel.SetTracerProvider(old) }()
	rootCtx, span := provider.Tracer("test").Start(context.Background(), "business producer")
	original := span.SpanContext().TraceID().String()
	headers := telemetry.AMQPHeaders(rootCtx)
	span.End()
	ctx, finish := telemetry.Consume(context.Background(), amqp.Delivery{Headers: headers, RoutingKey: "AccountAnomalyDetected"})
	defer finish(nil)
	corr := uuid.New()
	payload, _ := json.Marshal(map[string]any{"idCuenta": uuid.New(), "idCliente": uuid.New(), "operaciones": 11, "maxOperaciones": 10, "ventanaMinutos": 5})
	envelope := &events.EventEnvelope{MessageID: uuid.New(), CorrelationID: corr, Type: "AccountAnomalyDetected", Version: 1, OccurredAt: time.Now(), Producer: "account-service", Payload: payload}
	for i := 0; i < 2; i++ {
		if e := svc.ProcessEvent(ctx, envelope); e != nil {
			t.Fatal(e)
		}
	}
	alert := models.SLOAlert{AlertID: "latency-budget", Service: "payment-service", Severity: models.EventError, Summary: "p95 threshold exceeded", StartedAt: time.Now(), CorrelationID: &corr, TraceID: original}
	for i := 0; i < 2; i++ {
		if e := svc.ReceiveSLOAlert(context.Background(), alert); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := svc.QueryAudits(context.Background(), models.AuditFilter{CorrelationID: &corr, TraceID: original})
	if e != nil || len(rows) != 2 {
		t.Fatalf("business/SLO trace lost: %d %v", len(rows), e)
	}
	errors, e := svc.QueryAudits(context.Background(), models.AuditFilter{Severity: models.EventError, TraceID: original})
	if e != nil || len(errors) != 1 || errors[0].EventType != "SLOAlertTriggered" {
		t.Fatal("SLO severity not searchable")
	}
	affected, e := svc.QueryAudits(context.Background(), models.AuditFilter{Service: "payment-service", Severity: models.EventError})
	if e != nil || len(affected) != 1 || affected[0].Producer != "slo-monitor" {
		t.Fatal("affected SLO service not searchable")
	}
	inbox, e := svc.GetNotifications(context.Background(), models.NotificationFilter{CorrelationID: &corr, Limit: 100})
	if e != nil || len(inbox) != 2 {
		t.Fatal("anomaly or SLO notification missing/duplicated")
	}
	found := map[string]bool{}
	for _, n := range inbox {
		found[n.NotificationType] = true
		if n.NotificationType == "SLO_ALERT" && (n.Status != models.NotificationPending || n.Recipient != "ADMIN") {
			t.Fatal("admin inbox notification missing")
		}
		if n.NotificationType == "ACCOUNT_ANOMALY_DETECTED" && n.Severity != models.EventWarning {
			t.Fatal("anomaly not WARNING")
		}
	}
	if !found["SLO_ALERT"] || !found["ACCOUNT_ANOMALY_DETECTED"] {
		t.Fatal("wrong notification types")
	}
	// A message with no upstream trace is still auditable, without an invented trace ID.
	missing, end := telemetry.Consume(context.Background(), amqp.Delivery{RoutingKey: "test.uninstrumented"})
	defer end(nil)
	noTrace := &events.EventEnvelope{MessageID: uuid.New(), CorrelationID: uuid.New(), Type: "test.uninstrumented", Version: 1, Producer: "legacy-service", OccurredAt: time.Now(), Payload: json.RawMessage(`{}`)}
	if e := svc.ProcessEvent(missing, noTrace); e != nil {
		t.Fatal(e)
	}
	oldRows, e := svc.QueryAudits(context.Background(), models.AuditFilter{Service: "legacy-service"})
	if e != nil || len(oldRows) != 1 || oldRows[0].TraceID != "" {
		t.Fatal("trace fabricated for legacy event")
	}
}
