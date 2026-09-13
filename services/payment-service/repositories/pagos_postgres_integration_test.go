package repositories

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegracionEscenariosSimulados(t *testing.T) {
	url := os.Getenv("URL_BASE_DATOS_PRUEBAS")
	if url == "" {
		t.Skip("URL_BASE_DATOS_PRUEBAS no configurada")
	}
	ctx := context.Background()
	conexion, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Close()
	repositorio := NuevoRepositorioPagosPostgres(conexion)

	for _, prueba := range []struct {
		resultado models.ResultadoSimulado
		estado    models.EstadoPago
		motivo    string
	}{
		{models.ResultadoExito, models.EstadoPagoCompletado, ""},
		{models.ResultadoFallo, models.EstadoPagoRechazado, "fallo simulado"},
		{models.ResultadoTimeout, models.EstadoPagoRechazado, "timeout simulado"},
	} {
		t.Run(string(prueba.resultado), func(t *testing.T) {
			idPago, idCliente, idCuenta, correlacion := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			t.Cleanup(func() {
				_, _ = conexion.Exec(ctx, "DELETE FROM mensajes_salida WHERE id_correlacion=$1", correlacion)
				_, _ = conexion.Exec(ctx, "DELETE FROM mensajes_procesados WHERE id_correlacion=$1", correlacion)
				_, _ = conexion.Exec(ctx, "DELETE FROM intentos_pago WHERE id_pago=$1", idPago)
				_, _ = conexion.Exec(ctx, "DELETE FROM pagos WHERE id_pago=$1", idPago)
			})
			solicitud := events.SolicitudPago{IDPago: idPago, IDCliente: idCliente, IDCuentaOrigen: idCuenta, Beneficiario: "Proveedor", Concepto: "Prueba", MontoCentavos: 1000, TipoPago: "EXTERNO", ResultadoSimulado: string(prueba.resultado)}
			_, creado, err := repositorio.Iniciar(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.ComandoProcesarPago}, solicitud)
			if err != nil || !creado {
				t.Fatalf("iniciar pago: creado=%v error=%v", creado, err)
			}
			_, err = repositorio.ProcesarResultadoCuenta(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoCuentaDebitada}, events.ResultadoMovimiento{IDOperacion: idPago})
			if err != nil {
				t.Fatalf("procesar débito: %v", err)
			}
			if prueba.resultado != models.ResultadoExito {
				_, err = repositorio.ProcesarResultadoCuenta(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoCuentaCompensada}, events.ResultadoMovimiento{IDOperacion: idPago})
				if err != nil {
					t.Fatalf("procesar compensación: %v", err)
				}
			}
			pago, err := repositorio.BuscarPorID(ctx, idPago)
			if err != nil || pago.Estado != prueba.estado || pago.ResultadoSimulado != prueba.resultado || !strings.Contains(pago.MotivoRechazo, prueba.motivo) {
				t.Fatalf("resultado inesperado: pago=%+v error=%v", pago, err)
			}
		})
	}
}
