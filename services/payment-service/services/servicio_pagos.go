package services

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/Proyecto-SA-B-3/payment-service/repositories"
	"github.com/google/uuid"
)

var (
	ErrSolicitudInvalida = errors.New("solicitud de pago invalida")
	ErrTipoPagoInvalido  = errors.New("tipo de pago invalido")
)

type ServicioPagos interface {
	Procesar(context.Context, events.SobreMensaje, events.SolicitudPago) error
	ProcesarResultadoCuenta(context.Context, events.SobreMensaje, events.ResultadoMovimiento) error
	Consultar(context.Context, uuid.UUID) (*models.Pago, error)
	ListarCliente(context.Context, uuid.UUID, int, int) ([]models.Pago, error)
	RegistrarRespuesta(context.Context, events.SobreMensaje, string, any) error
}
type servicioPagos struct{ repositorio repositories.RepositorioPagos }

func NuevoServicioPagos(r repositories.RepositorioPagos) ServicioPagos {
	return &servicioPagos{repositorio: r}
}
func (s *servicioPagos) Procesar(ctx context.Context, m events.SobreMensaje, p events.SolicitudPago) error {
	if m.IDMensaje == uuid.Nil || m.IDCorrelacion == uuid.Nil || p.IDPago == uuid.Nil || p.IDCliente == uuid.Nil || p.IDCuentaOrigen == uuid.Nil || p.MontoCentavos <= 0 || p.Beneficiario == "" || p.Concepto == "" {
		return ErrSolicitudInvalida
	}
	if models.TipoPago(p.TipoPago) != models.TipoPagoInterno && models.TipoPago(p.TipoPago) != models.TipoPagoExterno {
		return ErrTipoPagoInvalido
	}
	_, _, err := s.repositorio.Iniciar(ctx, m, p)
	return err
}
func (s *servicioPagos) ProcesarResultadoCuenta(ctx context.Context, m events.SobreMensaje, r events.ResultadoMovimiento) error {
	if m.IDMensaje == uuid.Nil || m.IDCorrelacion == uuid.Nil || r.IDOperacion == uuid.Nil {
		return ErrSolicitudInvalida
	}
	_, err := s.repositorio.ProcesarResultadoCuenta(ctx, m, r)
	return err
}
func (s *servicioPagos) Consultar(ctx context.Context, id uuid.UUID) (*models.Pago, error) {
	return s.repositorio.BuscarPorID(ctx, id)
}
func (s *servicioPagos) ListarCliente(ctx context.Context, id uuid.UUID, l, o int) ([]models.Pago, error) {
	if l <= 0 || l > 100 {
		l = 25
	}
	if o < 0 {
		o = 0
	}
	return s.repositorio.ListarPorCliente(ctx, id, l, o)
}

func (s *servicioPagos) RegistrarRespuesta(ctx context.Context, mensaje events.SobreMensaje, tipo string, contenido any) error {
	_, err := s.repositorio.RegistrarRespuesta(ctx, mensaje, tipo, contenido)
	return err
}
