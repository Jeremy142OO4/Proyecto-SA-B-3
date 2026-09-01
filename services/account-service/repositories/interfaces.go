package repositories

import (
	"context"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
)

type RepositorioCuentas interface {
	Crear(ctx context.Context, cuenta *models.Cuenta) error
	BuscarPorID(ctx context.Context, idCuenta uuid.UUID) (*models.Cuenta, error)
	BuscarPorNumero(ctx context.Context, numeroCuenta string) (*models.Cuenta, error)
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]models.Cuenta, error)
	ProcesarMovimiento(ctx context.Context, solicitud SolicitudMovimientoCuenta) (ResultadoMovimiento, error)
	ListarMovimientos(ctx context.Context, idCuenta uuid.UUID, limite, desplazamiento int) ([]models.MovimientoCuenta, error)
	DesactivarCuentasInactivas(ctx context.Context, fechaLimite time.Time, saldoMaximoCentavos int64) (int64, error)
}

type RepositorioMensajes interface {
	FueProcesado(ctx context.Context, idMensaje uuid.UUID, nombreConsumidor string) (bool, error)
}

type MensajeSalida struct {
	IDMensaje        uuid.UUID
	TipoEvento       string
	VersionEvento    int
	Contenido        []byte
	IDCorrelacion    uuid.UUID
	FechaCreacion    time.Time
	CantidadIntentos int
}

type RepositorioSalida interface {
	ListarPendientes(ctx context.Context, limite int) ([]MensajeSalida, error)
	MarcarPublicado(ctx context.Context, idMensaje uuid.UUID) error
	RegistrarFalloPublicacion(ctx context.Context, idMensaje uuid.UUID) error
	RegistrarRechazo(ctx context.Context, mensaje events.SobreMensaje, consumidor, tipoEvento string, contenido any) (bool, error)
	RegistrarRespuesta(ctx context.Context, mensaje events.SobreMensaje, consumidor, tipoEvento string, contenido any) (bool, error)
}

type RepositorioSolicitudesCuenta interface {
	Iniciar(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudCrearCuenta) (bool, error)
	Completar(ctx context.Context, mensaje events.SobreMensaje, resultado events.ResultadoValidacionCliente) (*models.Cuenta, bool, error)
	Rechazar(ctx context.Context, mensaje events.SobreMensaje, resultado events.ResultadoValidacionCliente) (bool, error)
}

type SolicitudMovimientoCuenta struct {
	IDMensaje         uuid.UUID
	IDCorrelacion     uuid.UUID
	IDCuenta          uuid.UUID
	IDOperacion       uuid.UUID
	TipoMovimiento    models.TipoMovimiento
	MontoCentavos     int64
	Descripcion       string
	NombreConsumidor  string
	TipoEventoExitoso string
}

type ResultadoMovimiento struct {
	Movimiento models.MovimientoCuenta
	Duplicado  bool
}
