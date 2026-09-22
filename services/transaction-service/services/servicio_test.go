package services

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
	"testing"
)

type repoFalso struct{ transferencia models.Transferencia }

func (r *repoFalso) Iniciar(_ context.Context, _ events.SobreMensaje, t models.Transferencia) (bool, error) {
	r.transferencia = t
	return true, nil
}
func (r *repoFalso) ProcesarResultado(context.Context, events.SobreMensaje, events.ResultadoMovimiento) (bool, error) {
	return true, nil
}
func (r *repoFalso) ProcesarResultadoKYC(context.Context, events.SobreMensaje, events.ResultadoValidacionKYC) (bool, error) {
	return true, nil
}
func (r *repoFalso) ProcesarResultadoCuentas(context.Context, events.SobreMensaje, events.ResultadoValidacionCuentas) (bool, error) {
	return true, nil
}
func (r *repoFalso) Consultar(context.Context, uuid.UUID) (models.Transferencia, error) {
	return r.transferencia, nil
}
func (r *repoFalso) Historial(context.Context, events.SolicitudHistorial) ([]models.Transferencia, error) {
	return nil, nil
}
func (r *repoFalso) ResponderConsulta(context.Context, events.SobreMensaje, string, any) (bool, error) {
	return true, nil
}
func TestSolicitarTransferencia(t *testing.T) {
	r := &repoFalso{}
	s := Nuevo(r)
	m := events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}
	p := events.SolicitudTransferencia{IDCliente: uuid.New(), IDCuentaOrigen: uuid.New(), IDCuentaDestino: uuid.New(), MontoCentavos: 100}
	ok, e := s.Solicitar(context.Background(), m, p)
	if e != nil || !ok || r.transferencia.IDTransferencia == uuid.Nil {
		t.Fatalf("transferencia no iniciada: %v", e)
	}
}
func TestRechazaMismoOrigenDestino(t *testing.T) {
	r := &repoFalso{}
	s := Nuevo(r)
	id := uuid.New()
	_, e := s.Solicitar(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.SolicitudTransferencia{IDCliente: uuid.New(), IDCuentaOrigen: id, IDCuentaDestino: id, MontoCentavos: 100})
	if e == nil {
		t.Fatal("debio rechazar cuentas iguales")
	}
}

func TestAceptaFiltrosDeHistorial(t *testing.T) {
	r := &repoFalso{}
	s := Nuevo(r)
	cuenta := uuid.New()
	ok, e := s.Historial(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.SolicitudHistorial{
		IDCliente: uuid.New(), IDCuenta: &cuenta, FechaDesde: "2026-01-01", FechaHasta: "2026-01-31", Estado: "APPROVED",
	})
	if e != nil || !ok {
		t.Fatalf("filtros de historial rechazados: %v", e)
	}
}

func TestRechazaRangoDeFechasInvalido(t *testing.T) {
	r := &repoFalso{}
	s := Nuevo(r)
	_, e := s.Historial(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.SolicitudHistorial{
		IDCliente: uuid.New(), FechaDesde: "2026-02-01", FechaHasta: "2026-01-01",
	})
	if e == nil {
		t.Fatal("debio rechazar un rango de fechas invertido")
	}
}

func TestRechazaEventosSinIdentificadoresTransversales(t *testing.T) {
	s := Nuevo(&repoFalso{})
	_, err := s.Resultado(context.Background(), events.SobreMensaje{IDMensaje: uuid.New()}, events.ResultadoMovimiento{IDOperacion: uuid.New()})
	if !errors.Is(err, ErrSolicitudInvalida) {
		t.Fatalf("se esperaba rechazar evento sin CorrelationId: %v", err)
	}
	_, err = s.Historial(context.Background(), events.SobreMensaje{IDCorrelacion: uuid.New()}, events.SolicitudHistorial{IDCliente: uuid.New()})
	if !errors.Is(err, ErrSolicitudInvalida) {
		t.Fatalf("se esperaba rechazar consulta sin MessageId: %v", err)
	}
}

func TestConsultaRechazaOperacionInvalidaAntesDelRepositorio(t *testing.T) {
	s := Nuevo(&repoFalso{})
	_, err := s.Consultar(context.Background(), events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}, events.SolicitudConsulta{})
	if !errors.Is(err, ErrSolicitudInvalida) {
		t.Fatalf("se esperaba rechazar consulta sin id de transferencia: %v", err)
	}
}
