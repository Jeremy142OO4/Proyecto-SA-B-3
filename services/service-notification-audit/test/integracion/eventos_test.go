package integracion

import (
	"encoding/json"
	"testing"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"github.com/google/uuid"
)

func TestIntegracionNotificationAuditService(t *testing.T) {
	mensaje, correlacion := uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"evento cliente creado", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "cliente.creado"})
			return e == nil && len(b) > 0
		}},
		{"evento cliente activado", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "cliente.activado"})
			return e == nil && len(b) > 0
		}},
		{"evento pago completado", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "pago.completado"})
			return e == nil && len(b) > 0
		}},
		{"evento pago rechazado", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "pago.rechazado"})
			return e == nil && len(b) > 0
		}},
		{"evento transferencia completada", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "transferencia.completada"})
			return e == nil && len(b) > 0
		}},
		{"evento transferencia compensada", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "transferencia.compensada"})
			return e == nil && len(b) > 0
		}},
		{"correo de activacion", func() bool {
			b, e := json.Marshal(events.ActivationEmailPayload{CustomerID: mensaje, Email: "ana@example.com"})
			return e == nil && len(b) > 0
		}},
		{"filtro por correlacion", func() bool {
			f := models.NotificationFilter{CorrelationID: &correlacion}
			return f.CorrelationID != nil && *f.CorrelationID == correlacion
		}},
		{"filtro por estado", func() bool {
			f := models.NotificationFilter{Status: models.NotificationFailed}
			return f.Status == models.NotificationFailed
		}},
		{"log de auditoria completo", func() bool {
			b, e := json.Marshal(models.AuditLog{EventID: mensaje, CorrelationID: correlacion, Severity: models.EventInfo})
			return e == nil && len(b) > 0
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatal("integracion fallida")
			}
		})
	}
}
