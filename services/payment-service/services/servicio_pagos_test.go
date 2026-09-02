package services

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/Proyecto-SA-B-3/payment-service/repositories"
	"github.com/google/uuid"
	"testing"
)

type repoFalso struct {
	iniciado       bool
	errorResultado error
}

func (r *repoFalso) Iniciar(context.Context, events.SobreMensaje, events.SolicitudPago) (*models.Pago, bool, error) {
	r.iniciado = true
	return &models.Pago{}, true, nil
}
func (r *repoFalso) ProcesarResultadoCuenta(context.Context, events.SobreMensaje, events.ResultadoMovimiento) (bool, error) {
	return true, r.errorResultado
}

func TestIgnoraResultadoDeCuentaDeOtraOperacion(t *testing.T) {
	s := NuevoServicioPagos(&repoFalso{errorResultado: repositories.ErrPagoNoEncontrado})
	err := s.ProcesarResultadoCuenta(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.ResultadoMovimiento{IDOperacion: uuid.New()})
	if err != nil {
		t.Fatalf("resultado ajeno no debio fallar: %v", err)
	}
}
func (r *repoFalso) BuscarPorID(context.Context, uuid.UUID) (*models.Pago, error) {
	return &models.Pago{}, nil
}
func (r *repoFalso) ListarPorCliente(context.Context, uuid.UUID, int, int) ([]models.Pago, error) {
	return nil, nil
}
func (r *repoFalso) ListarSalida(context.Context, int) ([]repositories.MensajeSalida, error) {
	return nil, nil
}
func (r *repoFalso) MarcarPublicado(context.Context, uuid.UUID) error { return nil }
func (r *repoFalso) RegistrarFallo(context.Context, uuid.UUID) error  { return nil }
func (r *repoFalso) RegistrarRespuesta(context.Context, events.SobreMensaje, string, any) (bool, error) {
	return true, nil
}
func TestProcesarPagoValido(t *testing.T) {
	r := &repoFalso{}
	s := NuevoServicioPagos(r)
	err := s.Procesar(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.SolicitudPago{IDPago: uuid.New(), IDCliente: uuid.New(), IDCuentaOrigen: uuid.New(), Beneficiario: "USAC", Concepto: "Matricula", MontoCentavos: 10000, TipoPago: "INTERNO"})
	if err != nil || !r.iniciado {
		t.Fatalf("pago valido rechazado: %v", err)
	}
}
func TestRechazaMontoCero(t *testing.T) {
	s := NuevoServicioPagos(&repoFalso{})
	err := s.Procesar(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.SolicitudPago{IDPago: uuid.New(), IDCliente: uuid.New(), IDCuentaOrigen: uuid.New(), Beneficiario: "X", Concepto: "Y", TipoPago: "INTERNO"})
	if !errors.Is(err, ErrSolicitudInvalida) {
		t.Fatalf("error inesperado: %v", err)
	}
}
