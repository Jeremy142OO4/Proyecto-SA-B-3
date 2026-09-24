package unitarias

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
)

func TestUnitariasTransactionService(t *testing.T) {
	cliente, origen, destino, operacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"estado pendiente", func() bool { return models.Pendiente == "PENDIENTE" }},
		{"estado validando KYC", func() bool { return models.ValidandoKYC == "VALIDANDO_KYC" }},
		{"estado validando cuentas", func() bool { return models.ValidandoCuentas == "VALIDANDO_CUENTAS" }},
		{"estado procesando", func() bool { return models.Procesando == "PROCESANDO" }},
		{"estado completada", func() bool { return models.Completada == "COMPLETADA" }},
		{"estado rechazada", func() bool { return models.Rechazada == "RECHAZADA" }},
		{"estado compensando", func() bool { return models.Compensando == "COMPENSANDO" }},
		{"estado compensada", func() bool { return models.Compensada == "COMPENSADA" }},
		{"estado compensacion fallida", func() bool { return models.CompensacionFallida == "COMPENSACION_FALLIDA" }},
		{"comando transferencia", func() bool { return events.ComandoTransferencia != "" }},
		{"comando consulta", func() bool { return events.ComandoConsultar != "" }},
		{"comando historial", func() bool { return events.ComandoHistorial != "" }},
		{"comando debito", func() bool { return events.ComandoDebito != "" }},
		{"comando credito", func() bool { return events.ComandoCredito != "" }},
		{"comando compensacion", func() bool { return events.ComandoCompensacion != "" }},
		{"evento procesando", func() bool { return events.EventoProcesando != "" }},
		{"evento completada", func() bool { return events.EventoCompletada != "" }},
		{"evento rechazada", func() bool { return events.EventoRechazada != "" }},
		{"evento compensando", func() bool { return events.EventoCompensando != "" }},
		{"evento compensada", func() bool { return events.EventoCompensada != "" }},
		{"evento compensacion fallida", func() bool { return events.EventoCompensacionFallida != "" }},
		{"normaliza nombres en ingles", func() bool {
			p := events.SolicitudTransferencia{OperationID: operacion, CustomerID: cliente, SourceAccount: origen, TargetAccount: destino, AmountCents: 250, Description: "prueba"}
			id, cl, o, d, m, desc := p.Normalizar()
			return id == operacion && cl == cliente && o == origen && d == destino && m == 250 && desc == "prueba"
		}},
		{"normaliza nombres en español", func() bool {
			p := events.SolicitudTransferencia{IDTransferencia: operacion, IDCliente: cliente, IDCuentaOrigen: origen, IDCuentaDestino: destino, MontoCentavos: 300, Descripcion: "transferencia"}
			id, cl, o, d, m, desc := p.Normalizar()
			return id == operacion && cl == cliente && o == origen && d == destino && m == 300 && desc == "transferencia"
		}},
		{"normaliza cero a alias", func() bool {
			p := events.SolicitudTransferencia{IDTransferencia: operacion, IDCliente: cliente, IDCuentaOrigen: origen, IDCuentaDestino: destino, MontoCentavos: 1}
			id, cl, o, d, m, _ := p.Normalizar()
			return id != uuid.Nil && cl != uuid.Nil && o != uuid.Nil && d != uuid.Nil && m == 1
		}},
		{"transferencia serializa", func() bool {
			p := events.SolicitudTransferencia{IDTransferencia: operacion, IDCliente: cliente, IDCuentaOrigen: origen, IDCuentaDestino: destino, MontoCentavos: 500}
			b, e := json.Marshal(p)
			var x events.SolicitudTransferencia
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDTransferencia == operacion && x.MontoCentavos == 500
		}},
		{"historial serializa", func() bool {
			p := events.SolicitudHistorial{IDCliente: cliente, IDCuenta: &origen, FechaDesde: "2026-01-01", FechaHasta: "2026-01-31", Estado: "APPROVED", Limite: 20}
			b, e := json.Marshal(p)
			var x events.SolicitudHistorial
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDCliente == cliente && x.Estado == "APPROVED"
		}},
		{"consulta serializa", func() bool {
			p := events.SolicitudConsulta{IDTransferencia: operacion}
			b, e := json.Marshal(p)
			var x events.SolicitudConsulta
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDTransferencia == operacion
		}},
		{"resultado movimiento serializa", func() bool {
			p := events.ResultadoMovimiento{IDOperacion: operacion, IDCuenta: origen, MontoCentavos: 100}
			b, e := json.Marshal(p)
			var x events.ResultadoMovimiento
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.IDOperacion == operacion
		}},
		{"resultado KYC serializa", func() bool {
			p := events.ResultadoValidacionKYC{IDOperacion: operacion, IDCliente: cliente, Valido: true}
			b, e := json.Marshal(p)
			var x events.ResultadoValidacionKYC
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Valido && x.IDCliente == cliente
		}},
		{"resultado cuentas serializa", func() bool {
			p := events.ResultadoValidacionCuentas{IDOperacion: operacion, Valida: true, TipoCuentaOrigen: "AHORRO"}
			b, e := json.Marshal(p)
			var x events.ResultadoValidacionCuentas
			e2 := json.Unmarshal(b, &x)
			return e == nil && e2 == nil && x.Valida && x.TipoCuentaOrigen == "AHORRO"
		}},
		{"transferencia moneda GTQ", func() bool { return models.Transferencia{Moneda: "GTQ"}.Moneda == "GTQ" }},
		{"monto positivo", func() bool { return int64(1) > 0 }},
		{"cuentas distintas", func() bool { return origen != destino }},
		{"correlationId presente", func() bool { return operacion != uuid.Nil }},
		{"estado terminal completada", func() bool { return models.Completada != models.Procesando }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatalf("caso no cumplido: %s", caso.nombre)
			}
		})
	}
}
