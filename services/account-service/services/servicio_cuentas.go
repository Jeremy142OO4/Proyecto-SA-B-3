package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
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
	if solicitud.SaldoMinimoCentavos < 0 || solicitud.ComisionTransaccionCentavos < 0 {
		return nil, ErrReglaCuentaInvalida
	}

	tipoCuenta := models.TipoCuenta(strings.ToUpper(strings.TrimSpace(solicitud.TipoCuenta)))
	if tipoCuenta != models.TipoCuentaMonetaria && tipoCuenta != models.TipoCuentaAhorro && tipoCuenta != models.TipoCuentaCorriente {
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
		SaldoMinimoCentavos: solicitud.SaldoMinimoCentavos,
		ComisionTransaccionCentavos: solicitud.ComisionTransaccionCentavos,
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

func (s *servicioCuentas) ValidarTransferencia(ctx context.Context, solicitud events.SolicitudValidacionTransferencia) events.ResultadoValidacionTransferencia {
	resultado := events.ResultadoValidacionTransferencia{IDOperacion: solicitud.IDOperacion, IDCliente: solicitud.IDCliente}
	if solicitud.MontoCentavos <= 0 {
		resultado.Codigo, resultado.Motivo = "MONTO_INVALIDO", "el monto debe ser mayor que cero"
		return resultado
	}
	origen, err := s.repositorio.BuscarPorID(ctx, solicitud.IDCuentaOrigen)
	if err != nil {
		resultado.Codigo, resultado.Motivo = "CUENTA_ORIGEN_INVALIDA", err.Error()
		return resultado
	}
	resultado.TipoCuentaOrigen = string(origen.TipoCuenta)
	resultado.SaldoMinimoCentavos = origen.SaldoMinimoCentavos
	resultado.ComisionTransaccionCentavos = origen.ComisionTransaccionCentavos
	resultado.MontoTotalCentavos = solicitud.MontoCentavos + origen.ComisionTransaccionCentavos
	if origen.IDCliente != solicitud.IDCliente {
		resultado.Codigo, resultado.Motivo = "CUENTA_ORIGEN_AJENA", "la cuenta origen no pertenece al cliente"
		return resultado
	}
	if origen.Estado != models.EstadoCuentaActiva {
		resultado.Codigo, resultado.Motivo = "CUENTA_NO_ACTIVA", "la cuenta origen debe estar activa"
		return resultado
	}
	// Los pagos solo envían la cuenta origen; las transferencias envían ambas.
	if solicitud.IDCuentaDestino != uuid.Nil {
		destino, err := s.repositorio.BuscarPorID(ctx, solicitud.IDCuentaDestino)
		if err != nil {
			resultado.Codigo, resultado.Motivo = "CUENTA_DESTINO_INVALIDA", err.Error()
			return resultado
		}
		resultado.TipoCuentaDestino = string(destino.TipoCuenta)
		if destino.Estado != models.EstadoCuentaActiva {
			resultado.Codigo, resultado.Motivo = "CUENTA_NO_ACTIVA", "ambas cuentas deben estar activas"
			return resultado
		}
	}
	totalDebito := solicitud.MontoCentavos + origen.ComisionTransaccionCentavos
	if totalDebito < solicitud.MontoCentavos || origen.SaldoCentavos-totalDebito < origen.SaldoMinimoCentavos {
		resultado.Codigo, resultado.Motivo = "SALDO_MINIMO", "la operacion dejaria la cuenta por debajo del saldo minimo permitido"
		return resultado
	}
	resultado.Valida = true
	return resultado
}

func (s *servicioCuentas) ConsultarCuenta(ctx context.Context, idCuenta uuid.UUID) (*models.Cuenta, error) {
	if idCuenta == uuid.Nil {
		return nil, repositories.ErrCuentaNoEncontrada
	}
	return s.repositorio.BuscarPorID(ctx, idCuenta)
}

func (s *servicioCuentas) ListarCuentas(ctx context.Context, idCliente uuid.UUID) ([]models.Cuenta, error) {
	if idCliente == uuid.Nil {
		return nil, ErrClienteInvalido
	}
	return s.repositorio.ListarPorCliente(ctx, idCliente)
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
