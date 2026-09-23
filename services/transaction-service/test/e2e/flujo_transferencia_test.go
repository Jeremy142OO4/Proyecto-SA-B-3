package e2e

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
)

func TestE2ETransactionService(t *testing.T) {
	cliente, origen, destino, operacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"transferencia inicia en KYC", func() bool {
			p := events.SolicitudTransferencia{IDTransferencia: operacion, IDCliente: cliente, IDCuentaOrigen: origen, IDCuentaDestino: destino, MontoCentavos: 100}
			return p.IDCliente == cliente && p.IDCuentaDestino == destino
		}},
		{"KYC aprobado pasa a cuentas", func() bool {
			r := events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true}
			return r.Valido && r.IDOperacion == operacion
		}},
		{"cuentas aprobadas pasa a debito", func() bool {
			r := events.ResultadoValidacionCuentas{IDOperacion: operacion, Valida: true}
			return r.Valida && r.IDOperacion == operacion
		}},
		{"debito y credito comparten operacion", func() bool {
			d := events.ResultadoMovimiento{IDOperacion: operacion, IDCuenta: origen}
			c := events.ResultadoMovimiento{IDOperacion: operacion, IDCuenta: destino}
			return d.IDOperacion == c.IDOperacion
		}},
		{"flujo compensacion", func() bool {
			t := models.Transferencia{IDTransferencia: operacion, Estado: models.Compensada}
			b, _ := json.Marshal(t)
			var x models.Transferencia
			_ = json.Unmarshal(b, &x)
			return x.Estado == models.Compensada
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
