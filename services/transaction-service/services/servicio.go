package services

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/Proyecto-SA-B-3/transaction-service/repositories"
	"github.com/google/uuid"
	"strings"
	"time"
)

var ErrSolicitudInvalida = errors.New("solicitud de transferencia invalida")

type Servicio struct{ repo repositories.Repositorio }

func Nuevo(repo repositories.Repositorio) *Servicio { return &Servicio{repo} }
func (s *Servicio) Solicitar(ctx context.Context, m events.SobreMensaje, p events.SolicitudTransferencia) (bool, error) {
	id, cl, o, d, monto, desc := p.Normalizar()
	if id == uuid.Nil {
		id = uuid.New()
	}
	if m.IDMensaje == uuid.Nil || m.IDCorrelacion == uuid.Nil || cl == uuid.Nil || o == uuid.Nil || d == uuid.Nil || o == d || monto <= 0 {
		return false, ErrSolicitudInvalida
	}
	ahora := time.Now().UTC()
	t := models.Transferencia{IDTransferencia: id, IDCliente: cl, IDCuentaOrigen: o, IDCuentaDestino: d, IDCorrelacion: m.IDCorrelacion, MontoCentavos: monto, Moneda: "GTQ", Descripcion: strings.TrimSpace(desc), Estado: models.Pendiente, FechaCreacion: ahora, FechaActualizacion: ahora}
	return s.repo.Iniciar(ctx, m, t)
}
func (s *Servicio) Resultado(ctx context.Context, m events.SobreMensaje, p events.ResultadoMovimiento) (bool, error) {
	if m.IDMensaje == uuid.Nil || m.IDCorrelacion == uuid.Nil || p.IDOperacion == uuid.Nil {
		return false, ErrSolicitudInvalida
	}
	return s.repo.ProcesarResultado(ctx, m, p)
}
func (s *Servicio) Consultar(ctx context.Context, m events.SobreMensaje, p events.SolicitudConsulta) (bool, error) {
	t, e := s.repo.Consultar(ctx, p.IDTransferencia)
	if e != nil {
		return false, e
	}
	return s.repo.ResponderConsulta(ctx, m, events.EventoConsultada, t)
}
func (s *Servicio) Historial(ctx context.Context, m events.SobreMensaje, p events.SolicitudHistorial) (bool, error) {
	l, e := s.repo.Historial(ctx, p.IDCliente, p.Limite, p.Desplazamiento)
	if e != nil {
		return false, e
	}
	return s.repo.ResponderConsulta(ctx, m, events.EventoHistorial, map[string]any{"idCliente": p.IDCliente, "transferencias": l})
}
