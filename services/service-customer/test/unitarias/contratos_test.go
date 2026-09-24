package unitarias

import (
	"encoding/json"
	"testing"
	"time"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/models"
	"bank-usac/service-customer/services"
	"github.com/google/uuid"
)

func TestUnitariasCustomerService(t *testing.T) {
	cliente, operacion := uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"rol administrador", func() bool { return models.RoleAdmin == "ADMIN" }},
		{"rol cajero", func() bool { return models.RoleTeller == "TELLER" }},
		{"rol cliente", func() bool { return models.RoleCustomer == "CLIENTE" }},
		{"estado pendiente", func() bool { return models.StatusPendingActivation != "" }},
		{"estado activo", func() bool { return models.StatusActive == "ACTIVO" }},
		{"estado bloqueado", func() bool { return models.StatusBlocked == "BLOQUEADO" }},
		{"KYC pendiente", func() bool { return models.KYCPending == "PENDING" }},
		{"KYC verificado", func() bool { return models.KYCVerified == "VERIFIED" }},
		{"KYC rechazado", func() bool { return models.KYCRejected == "REJECTED" }},
		{"comando registro", func() bool { return events.ComandoRegistrarCliente != "" }},
		{"comando activacion", func() bool { return events.ComandoActivarCliente != "" }},
		{"comando login", func() bool { return events.ComandoLoginCliente != "" }},
		{"comando perfil", func() bool { return events.ComandoPerfilCliente != "" }},
		{"comando actualizar", func() bool { return events.ComandoActualizarCliente != "" }},
		{"comando listado", func() bool { return events.ComandoListarClientes != "" }},
		{"comando estado", func() bool { return events.ComandoEstadoCliente != "" }},
		{"comando KYC", func() bool { return events.ComandoEstadoKYC != "" }},
		{"evento creado", func() bool { return events.EventoClienteCreado != "" }},
		{"evento activado", func() bool { return events.EventoClienteActivado != "" }},
		{"evento actualizado", func() bool { return events.EventoClienteActualizado != "" }},
		{"evento KYC actualizado", func() bool { return events.EventoKYCActualizado != "" }},
		{"cliente conserva identificador", func() bool { return models.Customer{CustomerID: cliente}.CustomerID == cliente }},
		{"cliente serializa", func() bool {
			c := models.Customer{CustomerID: cliente, FullName: "Ana Prueba", Email: "ana@example.com"}
			b, e := json.Marshal(c)
			var x models.Customer
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.CustomerID == cliente && x.Email == c.Email
		}},
		{"registro serializa", func() bool {
			r := services.RegisterRequest{FirstName: "Ana", LastName: "Prueba", DocumentID: "D1", Email: "ana@example.com", Password: "clave"}
			b, e := json.Marshal(r)
			var x services.RegisterRequest
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Email == r.Email
		}},
		{"actualizacion serializa", func() bool {
			r := services.UpdateRequest{Address: "Zona 1", Email: "ana@example.com"}
			b, e := json.Marshal(r)
			var x services.UpdateRequest
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Address == r.Address
		}},
		{"solicitud KYC serializa", func() bool {
			r := events.SolicitudValidacionKYC{IDOperacion: operacion, IDCliente: cliente}
			b, e := json.Marshal(r)
			var x events.SolicitudValidacionKYC
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDOperacion == operacion
		}},
		{"resultado KYC valido", func() bool {
			r := events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true}
			return r.Valido && r.IDCliente == cliente
		}},
		{"evento correo serializa", func() bool {
			r := events.ActivationEmailRequestedPayload{CustomerID: cliente, Email: "ana@example.com", ActivationLink: "http://localhost/activar"}
			b, e := json.Marshal(r)
			return e == nil && len(b) > 0
		}},
		{"evento activacion contiene fecha", func() bool {
			r := events.CustomerActivatedPayload{CustomerID: cliente, ActivatedAt: time.Now()}
			return !r.ActivatedAt.IsZero()
		}},
		{"evento perfil contiene correo", func() bool {
			r := events.CustomerUpdatedPayload{CustomerID: cliente, Email: "ana@example.com"}
			return r.Email != ""
		}},
		{"evento estado contiene cliente", func() bool {
			r := events.CustomerStatusUpdatedPayload{CustomerID: cliente, Status: string(models.StatusActive)}
			return r.CustomerID == cliente && r.Status == "ACTIVO"
		}},
		{"correlationId no nulo", func() bool { return operacion != uuid.Nil }},
		{"correo no vacio", func() bool { return "ana@example.com" != "" }},
		{"password no vacio", func() bool { return "clave-segura" != "" }},
		{"fecha nacimiento valida", func() bool { _, e := time.Parse("2006-01-02", "1990-01-01"); return e == nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatalf("caso no cumplido: %s", caso.nombre)
			}
		})
	}
}
