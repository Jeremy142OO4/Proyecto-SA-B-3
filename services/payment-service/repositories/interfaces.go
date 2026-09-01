package repositories

import (
	"context"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
	"time"
)

type MensajeSalida struct {
	IDMensaje     uuid.UUID
	TipoEvento    string
	VersionEvento int
	Contenido     []byte
	IDCorrelacion uuid.UUID
	FechaCreacion time.Time
}
type RepositorioPagos interface {
	Iniciar(context.Context, events.SobreMensaje, events.SolicitudPago) (*models.Pago, bool, error)
	ProcesarResultadoCuenta(context.Context, events.SobreMensaje, events.ResultadoMovimiento) (bool, error)
	BuscarPorID(context.Context, uuid.UUID) (*models.Pago, error)
	ListarPorCliente(context.Context, uuid.UUID, int, int) ([]models.Pago, error)
	ListarSalida(context.Context, int) ([]MensajeSalida, error)
	MarcarPublicado(context.Context, uuid.UUID) error
	RegistrarFallo(context.Context, uuid.UUID) error
	RegistrarRespuesta(context.Context, events.SobreMensaje, string, any) (bool, error)
}
