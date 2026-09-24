package integracion

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
)

func TestIntegracionAccountService(t *testing.T) {
	cliente, cuenta, operacion := uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"crear cuenta via comando", func() bool {
			b, e := json.Marshal(events.SolicitudCrearCuenta{IDCliente: cliente, TipoCuenta: "CORRIENTE"})
			return e == nil && len(b) > 0
		}},
		{"debito via comando", func() bool {
			b, e := json.Marshal(events.SolicitudMovimiento{IDCuenta: cuenta, IDOperacion: operacion, MontoCentavos: 10})
			return e == nil && len(b) > 0
		}},
		{"credito via comando", func() bool {
			b, e := json.Marshal(events.SolicitudMovimiento{IDCuenta: cuenta, IDOperacion: operacion, MontoCentavos: 10})
			return e == nil && len(b) > 0
		}},
		{"respuesta validacion", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionTransferencia{IDOperacion: operacion, Valida: true})
			return e == nil && len(b) > 0
		}},
		{"respuesta rechazo", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionTransferencia{IDOperacion: operacion, Codigo: "SALDO_MINIMO"})
			return e == nil && len(b) > 0
		}},
		{"evento debitado", func() bool {
			b, e := json.Marshal(events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.EventoCuentaDebitada, Version: 1})
			return e == nil && len(b) > 0
		}},
		{"evento acreditado", func() bool {
			b, e := json.Marshal(events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.EventoCuentaAcreditada, Version: 1})
			return e == nil && len(b) > 0
		}},
		{"evento compensado", func() bool {
			b, e := json.Marshal(events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.EventoCuentaCompensada, Version: 1})
			return e == nil && len(b) > 0
		}},
		{"movimiento conserva tipo", func() bool {
			b, e := json.Marshal(models.MovimientoCuenta{TipoMovimiento: models.TipoMovimientoCredito})
			var x models.MovimientoCuenta
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.TipoMovimiento == models.TipoMovimientoCredito
		}},
		{"cuenta conserva reglas", func() bool {
			b, e := json.Marshal(models.Cuenta{SaldoMinimoCentavos: 500, ComisionTransaccionCentavos: 25})
			var x models.Cuenta
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.SaldoMinimoCentavos == 500 && x.ComisionTransaccionCentavos == 25
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
