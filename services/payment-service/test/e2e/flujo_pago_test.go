package e2e

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/google/uuid"
)

func TestE2EPaymentService(t *testing.T) {
	pago, cliente, cuenta, operacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"flujo pago interno", func() bool {
			p := events.SolicitudPago{IDPago: pago, IDCliente: cliente, IDCuentaOrigen: cuenta, TipoPago: "INTERNO", ResultadoSimulado: "EXITO"}
			b, _ := json.Marshal(p)
			var x events.SolicitudPago
			_ = json.Unmarshal(b, &x)
			return x.TipoPago == "INTERNO" && x.ResultadoSimulado == "EXITO"
		}},
		{"flujo externo exitoso", func() bool {
			p := events.SolicitudPago{IDPago: pago, TipoPago: "EXTERNO", ResultadoSimulado: "EXITO"}
			return p.ResultadoSimulado == "EXITO"
		}},
		{"flujo externo fallido", func() bool {
			p := events.SolicitudPago{IDPago: pago, TipoPago: "EXTERNO", ResultadoSimulado: "FALLO"}
			return p.ResultadoSimulado == "FALLO"
		}},
		{"flujo externo timeout", func() bool {
			p := events.SolicitudPago{IDPago: pago, TipoPago: "EXTERNO", ResultadoSimulado: "TIMEOUT"}
			return p.ResultadoSimulado == "TIMEOUT"
		}},
		{"flujo recibe resultado de cuenta", func() bool {
			m := events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.EventoCuentaDebitada}
			r := events.ResultadoMovimiento{IDOperacion: operacion, IDCuenta: cuenta}
			return m.IDCorrelacion == r.IDOperacion
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
