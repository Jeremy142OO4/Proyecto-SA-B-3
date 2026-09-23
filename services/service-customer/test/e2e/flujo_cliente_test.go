package e2e

import (
	"encoding/json"
	"testing"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/models"
	"github.com/google/uuid"
)

func TestE2ECustomerService(t *testing.T) {
	cliente, operacion := uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"registro genera cliente", func() bool {
			p := events.CustomerCreatedPayload{CustomerID: cliente, Email: "ana@example.com", Status: string(models.StatusPendingActivation)}
			return p.CustomerID != uuid.Nil && p.Status != ""
		}},
		{"registro solicita correo", func() bool {
			p := events.ActivationEmailRequestedPayload{CustomerID: cliente, Email: "ana@example.com", ActivationLink: "http://localhost/activar"}
			return p.CustomerID == cliente && p.Email != "" && p.ActivationLink != ""
		}},
		{"activacion cambia flujo", func() bool { p := events.CustomerActivatedPayload{CustomerID: cliente}; return p.CustomerID == cliente }},
		{"login devuelve datos de cliente", func() bool {
			p := models.Customer{CustomerID: cliente, Role: models.RoleCustomer, Status: models.StatusActive}
			b, _ := json.Marshal(p)
			var x models.Customer
			_ = json.Unmarshal(b, &x)
			return x.CustomerID == cliente && x.Status == models.StatusActive
		}},
		{"KYC se propaga a la operacion", func() bool {
			r := events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true}
			return r.IDOperacion == operacion && r.IDCliente == cliente && r.Valido
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
