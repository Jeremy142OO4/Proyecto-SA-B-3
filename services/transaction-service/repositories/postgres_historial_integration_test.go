package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegracionHistorialConFiltros(t *testing.T) {
	url := os.Getenv("URL_BASE_DATOS_PRUEBAS")
	if url == "" {
		t.Skip("URL_BASE_DATOS_PRUEBAS no configurada")
	}

	ctx := context.Background()
	conexion, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("conectar base de pruebas: %v", err)
	}
	defer conexion.Close()
	if err = conexion.Ping(ctx); err != nil {
		t.Fatalf("verificar base de pruebas: %v", err)
	}

	idCliente := uuid.New()
	idCuentaFiltrada := uuid.New()
	idCuentaAlterna := uuid.New()
	defer func() {
		_, _ = conexion.Exec(ctx, `DELETE FROM transferencias WHERE id_cliente=$1`, idCliente)
	}()

	insertarTransferenciaPrueba(t, conexion, idCliente, idCuentaFiltrada, idCuentaAlterna, models.Pendiente, "2026-09-01T10:00:00Z")
	insertarTransferenciaPrueba(t, conexion, idCliente, idCuentaAlterna, idCuentaFiltrada, models.Completada, "2026-09-05T12:00:00Z")
	insertarTransferenciaPrueba(t, conexion, idCliente, idCuentaAlterna, uuid.New(), models.Rechazada, "2026-09-10T15:00:00Z")

	repositorio := NuevoPostgres(conexion)

	porCuenta, err := repositorio.Historial(ctx, events.SolicitudHistorial{IDCliente: idCliente, IDCuenta: &idCuentaFiltrada})
	if err != nil {
		t.Fatalf("filtrar por cuenta: %v", err)
	}
	if len(porCuenta) != 2 {
		t.Fatalf("se esperaban 2 transferencias de la cuenta, se obtuvieron %d", len(porCuenta))
	}

	porFecha, err := repositorio.Historial(ctx, events.SolicitudHistorial{IDCliente: idCliente, FechaDesde: "2026-09-04", FechaHasta: "2026-09-06"})
	if err != nil {
		t.Fatalf("filtrar por fecha: %v", err)
	}
	if len(porFecha) != 1 || porFecha[0].Estado != models.Completada {
		t.Fatalf("el filtro de fecha no devolvio la transferencia esperada: %+v", porFecha)
	}

	porEstado, err := repositorio.Historial(ctx, events.SolicitudHistorial{IDCliente: idCliente, Estado: "APPROVED"})
	if err != nil {
		t.Fatalf("filtrar por estado: %v", err)
	}
	if len(porEstado) != 1 || porEstado[0].Estado != models.Completada {
		t.Fatalf("el filtro de estado no devolvio la transferencia esperada: %+v", porEstado)
	}

	combinado, err := repositorio.Historial(ctx, events.SolicitudHistorial{IDCliente: idCliente, IDCuenta: &idCuentaFiltrada, FechaDesde: "2026-09-05", FechaHasta: "2026-09-05", Estado: "COMPLETADA"})
	if err != nil {
		t.Fatalf("combinar filtros: %v", err)
	}
	if len(combinado) != 1 || combinado[0].Estado != models.Completada {
		t.Fatalf("la combinacion de filtros no devolvio la transferencia esperada: %+v", combinado)
	}
}

func insertarTransferenciaPrueba(t *testing.T, conexion *pgxpool.Pool, idCliente, origen, destino uuid.UUID, estado models.Estado, fechaTexto string) {
	t.Helper()
	fecha, err := time.Parse(time.RFC3339, fechaTexto)
	if err != nil {
		t.Fatalf("fecha de prueba invalida: %v", err)
	}
	_, err = conexion.Exec(context.Background(), `
		INSERT INTO transferencias(
			id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,id_correlacion,
			monto_centavos,moneda,descripcion,estado,fecha_creacion,fecha_actualizacion
		) VALUES($1,$2,$3,$4,$5,1000,'GTQ','prueba de filtros',$6,$7,$7)`,
		uuid.New(), idCliente, origen, destino, uuid.New(), estado, fecha)
	if err != nil {
		t.Fatalf("insertar transferencia de prueba: %v", err)
	}
}

func TestIntegracionSagaFase2(t *testing.T) {
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
	repositorio := NuevoPostgres(conexion)

	for _, escenario := range []struct {
		resultado string
		estado    models.Estado
		codigo    string
	}{{"EXITO", models.Completada, ""}, {"FALLO", models.Compensada, "FALLO_EXTERNO"}, {"TIMEOUT", models.Compensada, "TIMEOUT_EXTERNO"}} {
		t.Run(escenario.resultado, func(t *testing.T) {
			id, cliente, origen, destino, correlacion := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
			t.Cleanup(func() {
				_, _ = conexion.Exec(ctx, `DELETE FROM mensajes_salida WHERE id_correlacion=$1`, correlacion)
				_, _ = conexion.Exec(ctx, `DELETE FROM mensajes_procesados WHERE id_correlacion=$1`, correlacion)
				_, _ = conexion.Exec(ctx, `DELETE FROM transferencias WHERE id_transferencia=$1`, id)
			})
			transferencia := models.Transferencia{IDTransferencia: id, IDCliente: cliente, IDCuentaOrigen: origen, IDCuentaDestino: destino, IDCorrelacion: correlacion, MontoCentavos: 100, Moneda: "GTQ", Estado: models.ValidandoKYC, ResultadoExternoSimulado: escenario.resultado, FechaCreacion: time.Now().UTC()}
			if _, err = repositorio.Iniciar(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.ComandoTransferencia}, transferencia); err != nil {
				t.Fatal(err)
			}
			if _, err = repositorio.ProcesarResultadoKYC(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoKYCVerificado}, events.ResultadoValidacionKYC{IDOperacion: id, IDCliente: cliente, EstadoKYC: "VERIFIED", Valido: true}); err != nil {
				t.Fatal(err)
			}
			if _, err = repositorio.ProcesarResultadoCuentas(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoCuentasValidadas}, events.ResultadoValidacionCuentas{IDOperacion: id, IDCliente: cliente, Valida: true, TipoCuentaOrigen: "AHORRO", TipoCuentaDestino: "CORRIENTE"}); err != nil {
				t.Fatal(err)
			}
			if _, err = repositorio.ProcesarResultado(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoDebitada}, events.ResultadoMovimiento{IDOperacion: id, IDCuenta: origen, MontoCentavos: 100}); err != nil {
				t.Fatal(err)
			}
			if escenario.resultado == "EXITO" {
				_, err = repositorio.ProcesarResultado(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoAcreditada}, events.ResultadoMovimiento{IDOperacion: id, IDCuenta: destino, MontoCentavos: 100})
			} else {
				_, err = repositorio.ProcesarResultado(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: correlacion, Tipo: events.EventoCuentaCompensada}, events.ResultadoMovimiento{IDOperacion: id, IDCuenta: origen, MontoCentavos: 100})
			}
			if err != nil {
				t.Fatal(err)
			}
			obtenida, err := repositorio.Consultar(ctx, id)
			if err != nil || obtenida.Estado != escenario.estado || obtenida.CodigoError != escenario.codigo {
				t.Fatalf("resultado inesperado: %+v error=%v", obtenida, err)
			}
		})
	}
}
