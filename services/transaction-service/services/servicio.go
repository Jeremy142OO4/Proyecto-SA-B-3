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
	resultadoExterno := strings.ToUpper(strings.TrimSpace(p.ResultadoExternoSimulado))
	if resultadoExterno == "" {
		resultadoExterno = "EXITO"
	}
	if resultadoExterno != "EXITO" && resultadoExterno != "FALLO" && resultadoExterno != "TIMEOUT" {
		return false, ErrSolicitudInvalida
	}
	t := models.Transferencia{IDTransferencia: id, IDCliente: cl, IDCuentaOrigen: o, IDCuentaDestino: d, IDCorrelacion: m.IDCorrelacion, MontoCentavos: monto, Moneda: "GTQ", Descripcion: strings.TrimSpace(desc), Estado: models.ValidandoKYC, ResultadoExternoSimulado: resultadoExterno, FechaCreacion: ahora, FechaActualizacion: ahora}
	return s.repo.Iniciar(ctx, m, t)
}

func (s *Servicio) ResultadoKYC(ctx context.Context, m events.SobreMensaje, p events.ResultadoValidacionKYC) (bool, error) {
	if m.IDMensaje == uuid.Nil || m.IDCorrelacion == uuid.Nil || p.IDOperacion == uuid.Nil || p.IDCliente == uuid.Nil {
		return false, ErrSolicitudInvalida
	}
	return s.repo.ProcesarResultadoKYC(ctx, m, p)
}

func (s *Servicio) ResultadoCuentas(ctx context.Context, m events.SobreMensaje, p events.ResultadoValidacionCuentas) (bool, error) {
	if m.IDMensaje == uuid.Nil || m.IDCorrelacion == uuid.Nil || p.IDOperacion == uuid.Nil {
		return false, ErrSolicitudInvalida
	}
	return s.repo.ProcesarResultadoCuentas(ctx, m, p)
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
	if e := validarHistorial(p); e != nil {
		return false, e
	}
	l, e := s.repo.Historial(ctx, p)
	if e != nil {
		return false, e
	}
	return s.repo.ResponderConsulta(ctx, m, events.EventoHistorial, map[string]any{"idCliente": p.IDCliente, "transferencias": l})
}

func validarHistorial(p events.SolicitudHistorial) error {
	if p.IDCliente == uuid.Nil {
		return ErrSolicitudInvalida
	}
	if p.IDCuenta != nil && *p.IDCuenta == uuid.Nil {
		return ErrSolicitudInvalida
	}
	if _, e := fechaFiltro(p.FechaDesde, false); e != nil {
		return ErrSolicitudInvalida
	}
	if _, e := fechaFiltro(p.FechaHasta, true); e != nil {
		return ErrSolicitudInvalida
	}
	if p.FechaDesde != "" && p.FechaHasta != "" {
		desde, _ := fechaFiltro(p.FechaDesde, false)
		hasta, _ := fechaFiltro(p.FechaHasta, true)
		if !desde.Before(hasta) {
			return ErrSolicitudInvalida
		}
	}
	if p.Estado != "" {
		estado := strings.ToUpper(strings.TrimSpace(p.Estado))
		switch estado {
		case "PENDING", "APPROVED", "FAILED", string(models.ValidandoKYC), string(models.ValidandoCuentas), string(models.Pendiente), string(models.Procesando), string(models.Completada), string(models.Rechazada), string(models.Compensando), string(models.Compensada), string(models.CompensacionFallida):
		default:
			return ErrSolicitudInvalida
		}
	}
	return nil
}

func fechaFiltro(valor string, fin bool) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, nil
	}
	if len(valor) == len("2006-01-02") {
		fecha, e := time.ParseInLocation("2006-01-02", valor, time.UTC)
		if e != nil {
			return time.Time{}, e
		}
		if fin {
			fecha = fecha.Add(24 * time.Hour)
		}
		return fecha, nil
	}
	return time.Parse(time.RFC3339, valor)
}
