package messaging

import (
	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/services"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

type queryService struct {
	services.AuditService
	filter models.AuditFilter
	slo    models.SLOAlert
}

func (s *queryService) QueryAudits(_ context.Context, f models.AuditFilter) ([]*models.AuditLog, error) {
	s.filter = f
	return []*models.AuditLog{}, nil
}
func (s *queryService) ReceiveSLOAlert(_ context.Context, a models.SLOAlert) error {
	s.slo = a
	return nil
}
func TestAuditQueryCommandValidatesAndForwardsAllFilters(t *testing.T) {
	svc := &queryService{}
	consumer := &RabbitMQConsumer{auditSvc: svc}
	corr := uuid.New()
	payload, _ := json.Marshal(map[string]any{"correlationId": corr, "traceId": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "servicio": "payment-service", "nivel": "ERROR", "limite": 10, "desplazamiento": 2})
	status, body := consumer.executeCommand(context.Background(), events.EventEnvelope{Type: events.ComandoRegistros, Payload: payload})
	if status != 200 || string(body) != "[]" || svc.filter.CorrelationID == nil || *svc.filter.CorrelationID != corr || svc.filter.TraceID != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || svc.filter.Service != "payment-service" || svc.filter.Severity != models.EventError || svc.filter.Offset != 2 {
		t.Fatal("query contract not forwarded")
	}
	status, _ = consumer.executeCommand(context.Background(), events.EventEnvelope{Type: events.ComandoRegistros, Payload: json.RawMessage(`{"nivel":"DEBUG"}`)})
	if status != 400 {
		t.Fatal("invalid query accepted")
	}
	alert := models.SLOAlert{AlertID: "availability", Service: "api-gateway", Summary: "SLO failed", StartedAt: time.Now()}
	payload, _ = json.Marshal(alert)
	status, _ = consumer.executeCommand(context.Background(), events.EventEnvelope{Type: events.ComandoSLO, Payload: payload})
	if status != 202 || svc.slo.AlertID != "availability" {
		t.Fatal("SLO command not registered")
	}
}
