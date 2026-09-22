package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
)

type repositorioSolicitudesFalso struct {
	iniciada   bool
	completada bool
	rechazada  bool
}

func (r *repositorioSolicitudesFalso) Iniciar(context.Context, events.SobreMensaje, events.SolicitudCrearCuenta) (bool, error) {
	r.iniciada = true
	return true, nil
}

func (r *repositorioSolicitudesFalso) Completar(context.Context, events.SobreMensaje, events.ResultadoValidacionCliente) (*models.Cuenta, bool, error) {
	r.completada = true
	return &models.Cuenta{}, true, nil
}

func (r *repositorioSolicitudesFalso) Rechazar(context.Context, events.SobreMensaje, events.ResultadoValidacionCliente) (bool, error) {
	r.rechazada = true
	return true, nil
}

func TestProcesarValidacionClienteRechazaMensajeSinCorrelationId(t *testing.T) {
	repositorio := &repositorioSolicitudesFalso{}
	servicio := NuevoServicioCreacionCuentas(repositorio)
	err := servicio.ProcesarValidacionCliente(context.Background(), events.SobreMensaje{IDMensaje: uuid.New()}, events.ResultadoValidacionCliente{
		IDSolicitud: uuid.New(), IDCliente: uuid.New(), Activo: true,
	})
	if !errors.Is(err, ErrMensajeInvalido) {
		t.Fatalf("se esperaba ErrMensajeInvalido, se obtuvo %v", err)
	}
	if repositorio.completada || repositorio.rechazada {
		t.Fatal("no se debio modificar una solicitud con sobre invalido")
	}
}

func TestProcesarValidacionClienteCompletaClienteActivo(t *testing.T) {
	repositorio := &repositorioSolicitudesFalso{}
	servicio := NuevoServicioCreacionCuentas(repositorio)
	err := servicio.ProcesarValidacionCliente(context.Background(), events.SobreMensaje{
		IDMensaje: uuid.New(), IDCorrelacion: uuid.New(), Tipo: events.EventoClienteValidado,
	}, events.ResultadoValidacionCliente{IDSolicitud: uuid.New(), IDCliente: uuid.New(), Activo: true})
	if err != nil || !repositorio.completada {
		t.Fatalf("el cliente activo debio completar la solicitud: %v", err)
	}
}

func TestProcesarValidacionClienteRechazaClienteInactivo(t *testing.T) {
	repositorio := &repositorioSolicitudesFalso{}
	servicio := NuevoServicioCreacionCuentas(repositorio)
	err := servicio.ProcesarValidacionCliente(context.Background(), events.SobreMensaje{
		IDMensaje: uuid.New(), IDCorrelacion: uuid.New(), Tipo: events.EventoClienteRechazado,
	}, events.ResultadoValidacionCliente{IDSolicitud: uuid.New(), IDCliente: uuid.New(), Activo: false})
	if err != nil || !repositorio.rechazada {
		t.Fatalf("el cliente inactivo debio rechazar la solicitud: %v", err)
	}
}
