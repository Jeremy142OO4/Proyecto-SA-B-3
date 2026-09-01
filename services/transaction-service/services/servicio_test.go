package services

import (
	"context"
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
func (r *repoFalso) Consultar(context.Context, uuid.UUID) (models.Transferencia, error) {
	return r.transferencia, nil
}
func (r *repoFalso) Historial(context.Context, uuid.UUID, int, int) ([]models.Transferencia, error) {
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
