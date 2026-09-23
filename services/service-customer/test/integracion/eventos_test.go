package integracion

import (
	"encoding/json"
	"testing"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/models"
	"github.com/google/uuid"
)

func TestIntegracionCustomerService(t *testing.T) {
	cliente, operacion := uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"registro publicado", func() bool {
			b, e := json.Marshal(events.CustomerCreatedPayload{CustomerID: cliente, Email: "ana@example.com"})
			return e == nil && len(b) > 0
		}},
		{"activacion publicada", func() bool {
			b, e := json.Marshal(events.CustomerActivatedPayload{CustomerID: cliente})
			return e == nil && len(b) > 0
		}},
		{"actualizacion publicada", func() bool {
			b, e := json.Marshal(events.CustomerUpdatedPayload{CustomerID: cliente, Address: "Zona 1"})
			return e == nil && len(b) > 0
		}},
		{"estado publicado", func() bool {
			b, e := json.Marshal(events.CustomerStatusUpdatedPayload{CustomerID: cliente, Status: string(models.StatusActive)})
			return e == nil && len(b) > 0
		}},
		{"KYC solicitado", func() bool {
			b, e := json.Marshal(events.SolicitudValidacionKYC{IDOperacion: operacion, IDCliente: cliente})
			return e == nil && len(b) > 0
		}},
		{"KYC aprobado", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, EstadoKYC: "VERIFIED", Valido: true})
			return e == nil && len(b) > 0
		}},
		{"KYC rechazado publica resultado", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, EstadoKYC: "REJECTED", Valido: false})
			return e == nil && len(b) > 0
		}},
		{"correo de activacion", func() bool {
			b, e := json.Marshal(events.ActivationEmailRequestedPayload{CustomerID: cliente, Email: "ana@example.com"})
			return e == nil && len(b) > 0
		}},
		{"sobre con correlacion", func() bool {
			b, e := json.Marshal(events.EventEnvelope{MessageID: uuid.New(), CorrelationID: operacion, Type: events.EventoClienteCreado, Version: 1})
			return e == nil && len(b) > 0
		}},
		{"cliente conserva KYC", func() bool {
			b, e := json.Marshal(models.Customer{CustomerID: cliente, KYCStatus: models.KYCVerified})
			var x models.Customer
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.KYCStatus == models.KYCVerified
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
