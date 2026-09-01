package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
	"github.com/google/uuid"
)

type repositorioCuentasFalso struct {
	cuentaCreada       *models.Cuenta
	movimientoRecibido repositories.SolicitudMovimientoCuenta
	errorMovimiento    error
}

func (r *repositorioCuentasFalso) Crear(_ context.Context, cuenta *models.Cuenta) error {
	r.cuentaCreada = cuenta
	return nil
}
func (r *repositorioCuentasFalso) BuscarPorID(_ context.Context, _ uuid.UUID) (*models.Cuenta, error) {
	return r.cuentaCreada, nil
}
func (r *repositorioCuentasFalso) BuscarPorNumero(_ context.Context, _ string) (*models.Cuenta, error) {
	return nil, nil
}
func (r *repositorioCuentasFalso) ListarPorCliente(_ context.Context, _ uuid.UUID) ([]models.Cuenta, error) {
	return nil, nil
}
func (r *repositorioCuentasFalso) ProcesarMovimiento(_ context.Context, solicitud repositories.SolicitudMovimientoCuenta) (repositories.ResultadoMovimiento, error) {
	r.movimientoRecibido = solicitud
	return repositories.ResultadoMovimiento{}, r.errorMovimiento
}
func (r *repositorioCuentasFalso) ListarMovimientos(_ context.Context, _ uuid.UUID, _, _ int) ([]models.MovimientoCuenta, error) {
	return []models.MovimientoCuenta{}, nil
}
func (r *repositorioCuentasFalso) DesactivarCuentasInactivas(_ context.Context, _ time.Time, _ int64) (int64, error) {
	return 0, nil
}

func TestCrearCuentaMonetaria(t *testing.T) {
	repositorio := &repositorioCuentasFalso{}
	servicio := NuevoServicioCuentas(repositorio)
	idCliente := uuid.New()

	cuenta, err := servicio.CrearCuenta(context.Background(), events.SolicitudCrearCuenta{
		IDCliente:  idCliente,
		TipoCuenta: string(models.TipoCuentaMonetaria),
	})

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if cuenta.IDCliente != idCliente || cuenta.Estado != models.EstadoCuentaActiva {
		t.Fatalf("la cuenta creada no contiene los valores esperados: %+v", cuenta)
	}
	if len(cuenta.NumeroCuenta) != 12 {
		t.Fatalf("el numero de cuenta debe tener 12 caracteres, se obtuvo %q", cuenta.NumeroCuenta)
	}
}

func TestCrearCuentaRechazaTipoInvalido(t *testing.T) {
	servicio := NuevoServicioCuentas(&repositorioCuentasFalso{})
	_, err := servicio.CrearCuenta(context.Background(), events.SolicitudCrearCuenta{
		IDCliente: uuid.New(), TipoCuenta: "INVERSION",
	})
	if !errors.Is(err, ErrTipoCuentaInvalido) {
		t.Fatalf("se esperaba ErrTipoCuentaInvalido, se obtuvo %v", err)
	}
}

func TestProcesarDebitoValidaMonto(t *testing.T) {
	repositorio := &repositorioCuentasFalso{}
	servicio := NuevoServicioCuentas(repositorio)
	err := servicio.ProcesarDebito(context.Background(), events.SobreMensaje{
		IDMensaje: uuid.New(), IDCorrelacion: uuid.New(),
	}, events.SolicitudMovimiento{
		IDCuenta: uuid.New(), IDOperacion: uuid.New(), MontoCentavos: 0,
	})
	if !errors.Is(err, ErrMontoInvalido) {
		t.Fatalf("se esperaba ErrMontoInvalido, se obtuvo %v", err)
	}
}

func TestProcesarCreditoConstruyeMovimiento(t *testing.T) {
	repositorio := &repositorioCuentasFalso{}
	servicio := NuevoServicioCuentas(repositorio)
	mensaje := events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New()}
	solicitud := events.SolicitudMovimiento{IDCuenta: uuid.New(), IDOperacion: uuid.New(), MontoCentavos: 12500}

	if err := servicio.ProcesarCredito(context.Background(), mensaje, solicitud); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if repositorio.movimientoRecibido.TipoMovimiento != models.TipoMovimientoCredito {
		t.Fatalf("se esperaba un credito, se obtuvo %s", repositorio.movimientoRecibido.TipoMovimiento)
	}
	if repositorio.movimientoRecibido.TipoEventoExitoso != events.EventoCuentaAcreditada {
		t.Fatalf("evento de salida inesperado: %s", repositorio.movimientoRecibido.TipoEventoExitoso)
	}
}
