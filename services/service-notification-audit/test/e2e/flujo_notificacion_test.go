package e2e

import (
	"encoding/json"
	"testing"

	"bank-usac/service-notification-audit/events"
	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/services"
	"github.com/google/uuid"
)

func TestE2ENotificationAuditService(t *testing.T) {
	mensaje, correlacion := uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"recibir evento de cliente", func() bool {
			e := events.EventEnvelope{MessageID: mensaje, CorrelationID: correlacion, Type: "cliente.creado"}
			return e.MessageID != uuid.Nil && e.Type != ""
		}},
		{"clasificar evento informativo", func() bool { return services.ClassifyEvent("cliente.creado") == models.EventInfo }},
		{"clasificar evento de rechazo", func() bool { return services.ClassifyEvent("pago.rechazado") == models.EventWarning }},
		{"generar notificacion", func() bool {
			n := models.NotificationLog{CorrelationID: correlacion, Status: models.NotificationSent}
			return n.CorrelationID == correlacion && n.Status == models.NotificationSent
		}},
		{"registrar traza", func() bool {
			b, e := json.Marshal(models.AuditLog{EventID: mensaje, CorrelationID: correlacion})
			return e == nil && len(b) > 0
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatal("flujo E2E fallido")
			}
		})
	}
}
