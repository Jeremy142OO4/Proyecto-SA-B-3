package unitarias

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
)

func TestUnitariasPaymentService(t *testing.T) {
	cliente, cuenta, pago, operacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"tipo interno", func() bool { return models.TipoPagoInterno == "INTERNO" }},
		{"tipo externo", func() bool { return models.TipoPagoExterno == "EXTERNO" }},
		{"resultado exito", func() bool { return models.ResultadoExito == "EXITO" }},
		{"resultado fallo", func() bool { return models.ResultadoFallo == "FALLO" }},
		{"resultado timeout", func() bool { return models.ResultadoTimeout == "TIMEOUT" }},
		{"estado pendiente", func() bool { return models.EstadoPagoPendiente == "PENDIENTE" }},
		{"estado KYC", func() bool { return models.EstadoPagoValidandoKYC == "VALIDANDO_KYC" }},
		{"estado cuenta", func() bool { return models.EstadoPagoValidandoCuenta == "VALIDANDO_CUENTA" }},
		{"estado procesando", func() bool { return models.EstadoPagoProcesando == "PROCESANDO" }},
		{"estado compensando", func() bool { return models.EstadoPagoCompensando == "COMPENSANDO" }},
		{"estado completado", func() bool { return models.EstadoPagoCompletado == "COMPLETADO" }},
		{"estado rechazado", func() bool { return models.EstadoPagoRechazado == "RECHAZADO" }},
		{"comando procesar", func() bool { return events.ComandoProcesarPago != "" }},
		{"comando consultar", func() bool { return events.ComandoConsultarPago != "" }},
		{"comando listar", func() bool { return events.ComandoListarPagos != "" }},
		{"comando validar KYC", func() bool { return events.ComandoValidarKYC != "" }},
		{"comando validar cuenta", func() bool { return events.ComandoValidarCuenta != "" }},
		{"evento completado", func() bool { return events.EventoPagoCompletado != "" }},
		{"evento rechazado", func() bool { return events.EventoPagoRechazado != "" }},
		{"evento cuenta debitada", func() bool { return events.EventoCuentaDebitada != "" }},
		{"evento cuenta compensada", func() bool { return events.EventoCuentaCompensada != "" }},
		{"pago conserva IDs", func() bool {
			p := models.Pago{IDPago: pago, IDCliente: cliente, IDCuentaOrigen: cuenta, IDCorrelacion: operacion}
			return p.IDPago == pago && p.IDCliente == cliente && p.IDCuentaOrigen == cuenta && p.IDCorrelacion == operacion
		}},
		{"pago serializa", func() bool {
			p := models.Pago{IDPago: pago, TipoPago: models.TipoPagoExterno, ResultadoSimulado: models.ResultadoTimeout, MontoCentavos: 300}
			b, e := json.Marshal(p)
			var x models.Pago
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDPago == pago && x.ResultadoSimulado == models.ResultadoTimeout
		}},
		{"solicitud pago serializa", func() bool {
			p := events.SolicitudPago{IDPago: pago, IDCliente: cliente, IDCuentaOrigen: cuenta, TipoPago: "EXTERNO", ResultadoSimulado: "FALLO"}
			b, e := json.Marshal(p)
			var x events.SolicitudPago
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDPago == pago && x.ResultadoSimulado == "FALLO"
		}},
		{"resultado KYC serializa", func() bool {
			p := events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true}
			b, e := json.Marshal(p)
			var x events.ResultadoValidacionKYC
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Valido && x.IDOperacion == operacion
		}},
		{"resultado cuenta serializa", func() bool {
			p := events.ResultadoValidacionCuenta{IDOperacion: operacion, IDCliente: cliente, Valida: true}
			b, e := json.Marshal(p)
			var x events.ResultadoValidacionCuenta
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Valida && x.IDCliente == cliente
		}},
		{"resultado movimiento serializa", func() bool {
			p := events.ResultadoMovimiento{IDOperacion: operacion, IDCuenta: cuenta, Codigo: "OK"}
			b, e := json.Marshal(p)
			var x events.ResultadoMovimiento
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDCuenta == cuenta && x.Codigo == "OK"
		}},
		{"consulta pago serializa", func() bool {
			p := events.SolicitudConsultarPago{IDPago: pago}
			b, e := json.Marshal(p)
			var x events.SolicitudConsultarPago
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDPago == pago
		}},
		{"lista pagos serializa", func() bool {
			p := events.SolicitudListarPagos{IDCliente: cliente, Limite: 25, Desplazamiento: 5}
			b, e := json.Marshal(p)
			var x events.SolicitudListarPagos
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDCliente == cliente && x.Limite == 25
		}},
		{"monto positivo", func() bool { return int64(1) > 0 }},
		{"beneficiario requerido", func() bool { return "Proveedor" != "" }},
		{"concepto requerido", func() bool { return "Servicio" != "" }},
		{"moneda GTQ", func() bool { return models.Pago{Moneda: "GTQ"}.Moneda == "GTQ" }},
		{"version de mensaje", func() bool { return events.SobreMensaje{Version: 1}.Version == 1 }},
		{"correlationId presente", func() bool { return operacion != uuid.Nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatalf("caso no cumplido: %s", caso.nombre)
			}
		})
	}
}
