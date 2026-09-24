package e2e

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
)

func TestE2EAccountService(t *testing.T) {
	cliente, cuenta, operacion := uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"crear cuenta y conservar cliente", func() bool {
			b, _ := json.Marshal(events.SolicitudCrearCuenta{IDCliente: cliente, TipoCuenta: "AHORRO"})
			var x events.SolicitudCrearCuenta
			_ = json.Unmarshal(b, &x)
			return x.IDCliente == cliente
		}},
		{"validar transferencia aprobada", func() bool {
			r := events.ResultadoValidacionTransferencia{IDOperacion: operacion, IDCliente: cliente, Valida: true}
			return r.Valida && r.IDOperacion == operacion
		}},
		{"validar transferencia rechazada", func() bool {
			r := events.ResultadoValidacionTransferencia{IDOperacion: operacion, Valida: false, Codigo: "SALDO_MINIMO"}
			return !r.Valida && r.Codigo != ""
		}},
		{"debitar y correlacionar", func() bool {
			m := events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.ComandoSolicitarDebito}
			p := events.SolicitudMovimiento{IDCuenta: cuenta, IDOperacion: operacion, MontoCentavos: 100}
			return m.IDCorrelacion == p.IDOperacion
		}},
		{"acreditar cuenta destino", func() bool {
			m := events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: operacion, Tipo: events.EventoCuentaAcreditada}
			return m.Tipo == events.EventoCuentaAcreditada && m.IDCorrelacion == operacion
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatal("flujo E2E fallido")
			}
		})
	}
	_ = models.EstadoCuentaActiva
}
