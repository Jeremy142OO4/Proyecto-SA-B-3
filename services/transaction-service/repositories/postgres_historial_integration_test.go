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
