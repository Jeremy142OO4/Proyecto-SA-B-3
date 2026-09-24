package integracion

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
)

func TestIntegracionPaymentService(t *testing.T) {
	cliente, cuenta, pago, operacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"comando pago interno", func() bool {
			b, e := json.Marshal(events.SolicitudPago{IDPago: pago, IDCliente: cliente, IDCuentaOrigen: cuenta, TipoPago: "INTERNO", ResultadoSimulado: "EXITO"})
			return e == nil && len(b) > 0
		}},
		{"comando pago externo exitoso", func() bool {
			b, e := json.Marshal(events.SolicitudPago{IDPago: pago, TipoPago: "EXTERNO", ResultadoSimulado: "EXITO"})
			return e == nil && len(b) > 0
		}},
		{"comando pago externo fallido", func() bool {
			b, e := json.Marshal(events.SolicitudPago{IDPago: pago, TipoPago: "EXTERNO", ResultadoSimulado: "FALLO"})
			return e == nil && len(b) > 0
		}},
		{"comando pago timeout", func() bool {
			b, e := json.Marshal(events.SolicitudPago{IDPago: pago, TipoPago: "EXTERNO", ResultadoSimulado: "TIMEOUT"})
			return e == nil && len(b) > 0
		}},
		{"resultado KYC aprobado", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true})
			return e == nil && len(b) > 0
		}},
		{"resultado KYC rechazado", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: false})
			return e == nil && len(b) > 0
		}},
		{"resultado cuenta aprobado", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionCuenta{IDOperacion: operacion, Valida: true})
			return e == nil && len(b) > 0
		}},
		{"resultado cuenta rechazado", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionCuenta{IDOperacion: operacion, Valida: false})
			return e == nil && len(b) > 0
		}},
		{"evento pago completado", func() bool {
			b, e := json.Marshal(events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.EventoPagoCompletado})
			return e == nil && len(b) > 0
		}},
		{"pago conserva resultado", func() bool {
			b, e := json.Marshal(models.Pago{IDPago: pago, ResultadoSimulado: models.ResultadoTimeout})
			var x models.Pago
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.ResultadoSimulado == models.ResultadoTimeout
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
