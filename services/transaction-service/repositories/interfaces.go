package repositories

import (
	"context"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
	"time"
)

type Repositorio interface {
	Iniciar(context.Context, events.SobreMensaje, models.Transferencia) (bool, error)
	ProcesarResultado(context.Context, events.SobreMensaje, events.ResultadoMovimiento) (bool, error)
	Consultar(context.Context, uuid.UUID) (models.Transferencia, error)
	Historial(context.Context, uuid.UUID, int, int) ([]models.Transferencia, error)
	ResponderConsulta(context.Context, events.SobreMensaje, string, any) (bool, error)
}
type MensajeSalida struct {
	IDMensaje     uuid.UUID
	Tipo          string
	Contenido     []byte
	IDCorrelacion uuid.UUID
	EsComando     bool
	FechaCreacion time.Time
	Intentos      int
}
type RepositorioOutbox interface {
	ListarPendientes(context.Context, int) ([]MensajeSalida, error)
	MarcarPublicado(context.Context, uuid.UUID) error
	RegistrarFallo(context.Context, uuid.UUID) error
}
