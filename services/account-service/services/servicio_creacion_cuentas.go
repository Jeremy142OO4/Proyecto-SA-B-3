package services

import (
	"context"
	"fmt"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
	"github.com/google/uuid"
)

type servicioCreacionCuentas struct {
	repositorio repositories.RepositorioSolicitudesCuenta
}

func NuevoServicioCreacionCuentas(repositorio repositories.RepositorioSolicitudesCuenta) ServicioCreacionCuentas {
	return &servicioCreacionCuentas{repositorio: repositorio}
}

func (s *servicioCreacionCuentas) SolicitarCreacion(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudCrearCuenta) error {
	if mensaje.IDMensaje == uuid.Nil || mensaje.IDCorrelacion == uuid.Nil || solicitud.IDSolicitud == uuid.Nil || solicitud.IDCliente == uuid.Nil {
		return ErrMensajeInvalido
	}
	tipo := models.TipoCuenta(solicitud.TipoCuenta)
	if tipo != models.TipoCuentaMonetaria && tipo != models.TipoCuentaAhorro {
		return ErrTipoCuentaInvalido
	}
	_, err := s.repositorio.Iniciar(ctx, mensaje, solicitud)
	return err
}

func (s *servicioCreacionCuentas) ProcesarValidacionCliente(ctx context.Context, mensaje events.SobreMensaje, resultado events.ResultadoValidacionCliente) error {
	if resultado.IDSolicitud == uuid.Nil || resultado.IDCliente == uuid.Nil {
		return ErrMensajeInvalido
	}
	if resultado.Activo && mensaje.Tipo == events.EventoClienteValidado {
		_, _, err := s.repositorio.Completar(ctx, mensaje, resultado)
		return err
	}
	if resultado.Motivo == "" {
		resultado.Motivo = "cliente inexistente o no activo"
	}
	_, err := s.repositorio.Rechazar(ctx, mensaje, resultado)
	if err != nil {
		return fmt.Errorf("rechazar creacion de cuenta: %w", err)
	}
	return nil
}
