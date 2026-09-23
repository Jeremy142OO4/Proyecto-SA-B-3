package unitarias

import (
	"encoding/json"
	"testing"
	"time"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/services"
	"github.com/google/uuid"
)

func TestUnitariasNotificationAuditService(t *testing.T) {
	mensaje, correlacion := uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"severidad INFO", func() bool { return models.EventInfo == "INFO" }},
		{"severidad WARNING", func() bool { return models.EventWarning == "WARNING" }},
		{"severidad ERROR", func() bool { return models.EventError == "ERROR" }},
		{"notificacion pendiente", func() bool { return models.NotificationPending == "PENDING" }},
		{"notificacion enviada", func() bool { return models.NotificationSent == "SENT" }},
		{"notificacion fallida", func() bool { return models.NotificationFailed == "FAILED" }},
		{"comando registros", func() bool { return events.ComandoRegistros != "" }},
		{"comando traza", func() bool { return events.ComandoTraza != "" }},
		{"comando notificaciones", func() bool { return events.ComandoNotificaciones != "" }},
		{"clasifica evento normal como INFO", func() bool { return services.ClassifyEvent("cliente.creado") == models.EventInfo }},
		{"clasifica rechazo como WARNING", func() bool { return services.ClassifyEvent("pago.rechazado") == models.EventWarning }},
		{"clasifica timeout como ERROR", func() bool { return services.ClassifyEvent("pago.timeout") == models.EventError }},
		{"clasifica DLQ como ERROR", func() bool { return services.ClassifyEvent("notification-audit.dlq") == models.EventError }},
		{"clasifica compensando como WARNING", func() bool { return services.ClassifyEvent("transferencia.compensando") == models.EventWarning }},
		{"clasifica compensada como WARNING", func() bool { return services.ClassifyEvent("transferencia.compensada") == models.EventWarning }},
		{"payload correo conserva cliente", func() bool {
			p := events.ActivationEmailPayload{CustomerID: mensaje, Email: "ana@example.com"}
			return p.CustomerID == mensaje
		}},
		{"payload correo conserva enlace", func() bool {
			p := events.ActivationEmailPayload{ActivationLink: "http://localhost/activar"}
			return p.ActivationLink != ""
		}},
		{"payload correo conserva vencimiento", func() bool { p := events.ActivationEmailPayload{ExpiresAt: time.Now()}; return !p.ExpiresAt.IsZero() }},
		{"audit conserva evento", func() bool {
			l := models.AuditLog{EventID: mensaje, CorrelationID: correlacion, EventType: "cliente.creado"}
			return l.EventID == mensaje && l.CorrelationID == correlacion
		}},
		{"audit conserva productor", func() bool { l := models.AuditLog{Producer: "customer-service"}; return l.Producer != "" }},
		{"audit conserva version", func() bool { l := models.AuditLog{Version: 1}; return l.Version == 1 }},
		{"notificacion conserva receptor", func() bool { n := models.NotificationLog{Recipient: "ana@example.com"}; return n.Recipient != "" }},
		{"notificacion conserva estado", func() bool {
			n := models.NotificationLog{Status: models.NotificationSent}
			return n.Status == models.NotificationSent
		}},
		{"filtro conserva limite", func() bool { f := models.NotificationFilter{Limit: 25}; return f.Limit == 25 }},
		{"filtro conserva destinatario", func() bool { f := models.NotificationFilter{Recipient: "ana@example.com"}; return f.Recipient != "" }},
		{"filtro conserva estado", func() bool {
			f := models.NotificationFilter{Status: models.NotificationFailed}
			return f.Status == models.NotificationFailed
		}},
		{"sobre serializa", func() bool {
			s := events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "pago.completado", Version: 1}
			b, e := json.Marshal(s)
			var x events.EventEnvelope
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.MessageID == mensaje && x.Type == s.Type
		}},
		{"payload correo serializa", func() bool {
			p := events.ActivationEmailPayload{CustomerID: mensaje, Email: "ana@example.com"}
			b, e := json.Marshal(p)
			var x events.ActivationEmailPayload
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.CustomerID == mensaje
		}},
		{"audit serializa", func() bool {
			l := models.AuditLog{ID: mensaje, Severity: models.EventError, EventType: "pago.rechazado"}
			b, e := json.Marshal(l)
			var x models.AuditLog
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Severity == models.EventError
		}},
		{"notificacion serializa", func() bool {
			n := models.NotificationLog{ID: mensaje, Status: models.NotificationSent, Recipient: "a@b.com"}
			b, e := json.Marshal(n)
			var x models.NotificationLog
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Status == models.NotificationSent
		}},
		{"correlationId no nulo", func() bool { return correlacion != uuid.Nil }},
		{"messageId no nulo", func() bool { return mensaje != uuid.Nil }},
		{"evento pago completado INFO", func() bool { return services.ClassifyEvent("pago.completado") == models.EventInfo }},
		{"evento pago rechazado WARNING", func() bool { return services.ClassifyEvent("pago.rechazado") == models.EventWarning }},
		{"evento transferencia completada INFO", func() bool { return services.ClassifyEvent("transferencia.completada") == models.EventInfo }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatalf("caso no cumplido: %s", caso.nombre)
			}
		})
	}
}
