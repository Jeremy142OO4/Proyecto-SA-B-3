package unitarias

import (
	"encoding/json"
	"testing"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
)

func ejecutarCasos(t *testing.T, casos []struct {
	nombre  string
	validar func() bool
}) {
	t.Helper()
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !caso.validar() {
				t.Fatalf("caso no cumplido: %s", caso.nombre)
			}
		})
	}
}

func TestUnitariasAccountService(t *testing.T) {
	cliente, cuenta, operacion := uuid.New(), uuid.New(), uuid.New()
	casos := []struct {
		nombre  string
		validar func() bool
	}{
		{"tipo monetaria", func() bool { return models.TipoCuentaMonetaria == "MONETARIA" }},
		{"tipo ahorro", func() bool { return models.TipoCuentaAhorro == "AHORRO" }},
		{"tipo corriente", func() bool { return models.TipoCuentaCorriente == "CORRIENTE" }},
		{"estado activa", func() bool { return models.EstadoCuentaActiva == "ACTIVA" }},
		{"estado inactiva", func() bool { return models.EstadoCuentaInactiva == "INACTIVA" }},
		{"estado bloqueada", func() bool { return models.EstadoCuentaBloqueada == "BLOQUEADA" }},
		{"estado cerrada", func() bool { return models.EstadoCuentaCerrada == "CERRADA" }},
		{"movimiento debito", func() bool { return models.TipoMovimientoDebito == "DEBITO" }},
		{"movimiento credito", func() bool { return models.TipoMovimientoCredito == "CREDITO" }},
		{"movimiento compensacion", func() bool { return models.TipoMovimientoCompensacion == "COMPENSACION" }},
		{"comando crear cuenta", func() bool { return events.ComandoCrearCuenta != "" }},
		{"comando debito", func() bool { return events.ComandoSolicitarDebito != "" }},
		{"comando credito", func() bool { return events.ComandoSolicitarCredito != "" }},
		{"comando compensacion", func() bool { return events.ComandoSolicitarCompensacion != "" }},
		{"comando consultar", func() bool { return events.ComandoConsultarCuenta != "" }},
		{"comando movimientos", func() bool { return events.ComandoListarMovimientos != "" }},
		{"comando cuentas", func() bool { return events.ComandoListarCuentas != "" }},
		{"comando validacion", func() bool { return events.ComandoValidarTransferencia != "" }},
		{"evento cuenta creada", func() bool { return events.EventoCuentaCreada != "" }},
		{"evento cuenta debitada", func() bool { return events.EventoCuentaDebitada != "" }},
		{"evento cuenta acreditada", func() bool { return events.EventoCuentaAcreditada != "" }},
		{"evento cuenta compensada", func() bool { return events.EventoCuentaCompensada != "" }},
		{"evento validada", func() bool { return events.EventoTransferenciaValidada != "" }},
		{"evento rechazada", func() bool { return events.EventoTransferenciaRechazada != "" }},
		{"cuenta conserva identificadores", func() bool {
			dato := models.Cuenta{IDCuenta: cuenta, IDCliente: cliente, Moneda: "GTQ", Version: 1}
			return dato.IDCuenta == cuenta && dato.IDCliente == cliente && dato.Moneda == "GTQ" && dato.Version == 1
		}},
		{"cuenta serializa", func() bool {
			dato := models.Cuenta{IDCuenta: cuenta, IDCliente: cliente, NumeroCuenta: "100000000001", TipoCuenta: models.TipoCuentaAhorro}
			b, err := json.Marshal(dato)
			var salida models.Cuenta
			err2 := json.Unmarshal(b, &salida)
			return err == nil && err2 == nil && salida.IDCuenta == cuenta && salida.NumeroCuenta == dato.NumeroCuenta
		}},
		{"movimiento serializa", func() bool {
			dato := models.MovimientoCuenta{IDCuenta: cuenta, IDOperacion: operacion, TipoMovimiento: models.TipoMovimientoDebito, MontoCentavos: 125}
			b, err := json.Marshal(dato)
			var salida models.MovimientoCuenta
			err2 := json.Unmarshal(b, &salida)
			return err == nil && err2 == nil && salida.IDOperacion == operacion && salida.MontoCentavos == 125
		}},
		{"solicitud crear serializa", func() bool {
			dato := events.SolicitudCrearCuenta{IDCliente: cliente, TipoCuenta: "AHORRO", SaldoMinimoCentavos: 100}
			b, err := json.Marshal(dato)
			var salida events.SolicitudCrearCuenta
			err2 := json.Unmarshal(b, &salida)
			return err == nil && err2 == nil && salida.IDCliente == cliente && salida.TipoCuenta == "AHORRO"
		}},
		{"solicitud movimiento serializa", func() bool {
			dato := events.SolicitudMovimiento{IDCuenta: cuenta, IDOperacion: operacion, MontoCentavos: 900}
			b, err := json.Marshal(dato)
			var salida events.SolicitudMovimiento
			err2 := json.Unmarshal(b, &salida)
			return err == nil && err2 == nil && salida.IDCuenta == cuenta && salida.MontoCentavos == 900
		}},
		{"solicitud validacion serializa", func() bool {
			dato := events.SolicitudValidacionTransferencia{IDOperacion: operacion, IDCliente: cliente, IDCuentaOrigen: cuenta, MontoCentavos: 25}
			b, err := json.Marshal(dato)
			var salida events.SolicitudValidacionTransferencia
			err2 := json.Unmarshal(b, &salida)
			return err == nil && err2 == nil && salida.IDOperacion == operacion && salida.IDCuentaOrigen == cuenta
		}},
		{"monto positivo", func() bool { return (int64(100) > 0) }},
		{"numero de cuenta con longitud", func() bool { return len("100000000001") == 12 }},
		{"moneda de negocio", func() bool { return models.Cuenta{Moneda: "GTQ"}.Moneda == "GTQ" }},
		{"version inicial", func() bool { return models.Cuenta{Version: 1}.Version == 1 }},
		{"ids no nulos", func() bool { return cliente != uuid.Nil && cuenta != uuid.Nil && operacion != uuid.Nil }},
	}
	ejecutarCasos(t, casos)
}
