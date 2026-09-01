package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
	"github.com/google/uuid"
)

const consumidorMovimientos = "account-service.movimientos"

type servicioCuentas struct {
	repositorio repositories.RepositorioCuentas
}

func NuevoServicioCuentas(repositorio repositories.RepositorioCuentas) ServicioCuentas {
	return &servicioCuentas{repositorio: repositorio}
}

func (s *servicioCuentas) CrearCuenta(ctx context.Context, solicitud events.SolicitudCrearCuenta) (*models.Cuenta, error) {
	if solicitud.IDCliente == uuid.Nil {
		return nil, ErrClienteInvalido
	}

	tipoCuenta := models.TipoCuenta(solicitud.TipoCuenta)
	if tipoCuenta != models.TipoCuentaMonetaria && tipoCuenta != models.TipoCuentaAhorro {
		return nil, ErrTipoCuentaInvalido
	}

	numeroCuenta, err := generarNumeroCuenta()
	if err != nil {
		return nil, err
	}
	ahora := time.Now().UTC()
	cuenta := &models.Cuenta{
		IDCuenta:           uuid.New(),
		IDCliente:          solicitud.IDCliente,
		NumeroCuenta:       numeroCuenta,
		TipoCuenta:         tipoCuenta,
		SaldoCentavos:      0,
		Moneda:             "GTQ",
		Estado:             models.EstadoCuentaActiva,
		FechaCreacion:      ahora,
		FechaActualizacion: ahora,
		Version:            1,
	}
	if err := s.repositorio.Crear(ctx, cuenta); err != nil {
		return nil, fmt.Errorf("guardar nueva cuenta: %w", err)
	}
	return cuenta, nil
}

func (s *servicioCuentas) ConsultarCuenta(ctx context.Context, idCuenta uuid.UUID) (*models.Cuenta, error) {
	if idCuenta == uuid.Nil {
		return nil, repositories.ErrCuentaNoEncontrada
	}
	return s.repositorio.BuscarPorID(ctx, idCuenta)
}

func (s *servicioCuentas) ListarMovimientos(ctx context.Context, idCuenta uuid.UUID, limite, desplazamiento int) ([]models.MovimientoCuenta, error) {
	if limite <= 0 || limite > 100 {
		limite = 25
	}
	if desplazamiento < 0 {
		desplazamiento = 0
	}
	return s.repositorio.ListarMovimientos(ctx, idCuenta, limite, desplazamiento)
}

func (s *servicioCuentas) DesactivarCuentasInactivas(ctx context.Context) (int64, error) {
	return s.repositorio.DesactivarCuentasInactivas(ctx, time.Now().UTC().AddDate(0, -6, 0), 5000)
}

func (s *servicioCuentas) ProcesarDebito(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudMovimiento) error {
	return s.procesarMovimiento(ctx, mensaje, solicitud, models.TipoMovimientoDebito, events.EventoCuentaDebitada, "debito de cuenta")
}

func (s *servicioCuentas) ProcesarCredito(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudMovimiento) error {
	return s.procesarMovimiento(ctx, mensaje, solicitud, models.TipoMovimientoCredito, events.EventoCuentaAcreditada, "credito de cuenta")
}

func (s *servicioCuentas) ProcesarCompensacion(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudMovimiento) error {
	return s.procesarMovimiento(ctx, mensaje, solicitud, models.TipoMovimientoCompensacion, events.EventoCuentaCompensada, "compensacion de debito")
}

func (s *servicioCuentas) procesarMovimiento(
	ctx context.Context,
	mensaje events.SobreMensaje,
	solicitud events.SolicitudMovimiento,
	tipo models.TipoMovimiento,
	eventoExitoso string,
	descripcion string,
) error {
	if mensaje.IDMensaje == uuid.Nil || mensaje.IDCorrelacion == uuid.Nil || solicitud.IDCuenta == uuid.Nil || solicitud.IDOperacion == uuid.Nil {
		return ErrMensajeInvalido
	}
	if solicitud.MontoCentavos <= 0 {
		return ErrMontoInvalido
	}

	_, err := s.repositorio.ProcesarMovimiento(ctx, repositories.SolicitudMovimientoCuenta{
		IDMensaje:         mensaje.IDMensaje,
		IDCorrelacion:     mensaje.IDCorrelacion,
		IDCuenta:          solicitud.IDCuenta,
		IDOperacion:       solicitud.IDOperacion,
		TipoMovimiento:    tipo,
		MontoCentavos:     solicitud.MontoCentavos,
		Descripcion:       descripcion,
		NombreConsumidor:  consumidorMovimientos,
		TipoEventoExitoso: eventoExitoso,
	})
	return err
}

func generarNumeroCuenta() (string, error) {
	limite := new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil)
	valor, err := rand.Int(rand.Reader, limite)
	if err != nil {
		return "", fmt.Errorf("generar numero de cuenta: %w", err)
	}
	return fmt.Sprintf("10%010d", valor), nil
}
