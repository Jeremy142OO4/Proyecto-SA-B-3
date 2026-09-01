package services

import (
	"context"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
)

type ServicioCuentas interface {
	CrearCuenta(ctx context.Context, solicitud events.SolicitudCrearCuenta) (*models.Cuenta, error)
	ConsultarCuenta(ctx context.Context, idCuenta uuid.UUID) (*models.Cuenta, error)
	ProcesarDebito(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudMovimiento) error
	ProcesarCredito(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudMovimiento) error
	ProcesarCompensacion(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudMovimiento) error
	ListarMovimientos(ctx context.Context, idCuenta uuid.UUID, limite, desplazamiento int) ([]models.MovimientoCuenta, error)
	DesactivarCuentasInactivas(ctx context.Context) (int64, error)
}

type ServicioCreacionCuentas interface {
	SolicitarCreacion(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudCrearCuenta) error
	ProcesarValidacionCliente(ctx context.Context, mensaje events.SobreMensaje, resultado events.ResultadoValidacionCliente) error
}
