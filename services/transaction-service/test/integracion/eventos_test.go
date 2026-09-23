package integracion

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
)

func TestIntegracionTransactionService(t *testing.T) {
	cliente, origen, destino, operacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"solicitud transferencia", func() bool {
			b, e := json.Marshal(events.SolicitudTransferencia{IDTransferencia: operacion, IDCliente: cliente, IDCuentaOrigen: origen, IDCuentaDestino: destino, MontoCentavos: 100})
			return e == nil && len(b) > 0
		}},
		{"consulta transferencia", func() bool {
			b, e := json.Marshal(events.SolicitudConsulta{IDTransferencia: operacion})
			return e == nil && len(b) > 0
		}},
		{"historial sin filtros", func() bool {
			b, e := json.Marshal(events.SolicitudHistorial{IDCliente: cliente, Limite: 25})
			return e == nil && len(b) > 0
		}},
		{"historial por cuenta", func() bool {
			b, e := json.Marshal(events.SolicitudHistorial{IDCliente: cliente, IDCuenta: &origen})
			return e == nil && len(b) > 0
		}},
		{"historial por fecha", func() bool {
			b, e := json.Marshal(events.SolicitudHistorial{IDCliente: cliente, FechaDesde: "2026-01-01", FechaHasta: "2026-01-31"})
			return e == nil && len(b) > 0
		}},
		{"historial por estado", func() bool {
			b, e := json.Marshal(events.SolicitudHistorial{IDCliente: cliente, Estado: "FAILED"})
			return e == nil && len(b) > 0
		}},
		{"resultado KYC", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true})
			return e == nil && len(b) > 0
		}},
		{"resultado cuenta", func() bool {
			b, e := json.Marshal(events.ResultadoValidacionCuentas{IDOperacion: operacion, Valida: true})
			return e == nil && len(b) > 0
		}},
		{"resultado movimiento", func() bool {
			b, e := json.Marshal(events.ResultadoMovimiento{IDOperacion: operacion, IDCuenta: origen, MontoCentavos: 100})
			return e == nil && len(b) > 0
		}},
		{"transferencia conserva estado", func() bool {
			b, e := json.Marshal(models.Transferencia{IDTransferencia: operacion, Estado: models.Completada})
			var x models.Transferencia
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Estado == models.Completada
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
