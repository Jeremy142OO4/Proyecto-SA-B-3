package publishers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/messaging"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
)

type PublicadorOutbox struct {
	repositorio repositories.RepositorioSalida
	publicador  *messaging.Publicador
	intervalo   time.Duration
}

func NuevoPublicadorOutbox(repositorio repositories.RepositorioSalida, publicador *messaging.Publicador, intervalo time.Duration) *PublicadorOutbox {
	return &PublicadorOutbox{repositorio: repositorio, publicador: publicador, intervalo: intervalo}
}

func (p *PublicadorOutbox) Ejecutar(ctx context.Context) {
	temporizador := time.NewTicker(p.intervalo)
	defer temporizador.Stop()

	p.publicarPendientes(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-temporizador.C:
			p.publicarPendientes(ctx)
		}
	}
}

func (p *PublicadorOutbox) publicarPendientes(ctx context.Context) {
	mensajes, err := p.repositorio.ListarPendientes(ctx, 50)
	if err != nil {
		slog.Error("no se pudieron consultar los mensajes Outbox", "error", err)
		return
	}
	for _, salida := range mensajes {
		mensaje := events.SobreMensaje{
			IDMensaje: salida.IDMensaje, IDCorrelacion: salida.IDCorrelacion,
			Tipo: salida.TipoEvento, Version: salida.VersionEvento,
			OcurridoEn: salida.FechaCreacion, Productor: "account-service",
			Contenido: json.RawMessage(salida.Contenido),
		}
		var errorPublicacion error
		if salida.TipoEvento == events.ComandoValidarCliente {
			errorPublicacion = p.publicador.PublicarComando(ctx, mensaje)
		} else {
			errorPublicacion = p.publicador.PublicarEvento(ctx, mensaje)
		}
		if errorPublicacion != nil {
			slog.Error("fallo la publicacion Outbox", "idMensaje", salida.IDMensaje, "error", errorPublicacion)
			if errorRegistro := p.repositorio.RegistrarFalloPublicacion(ctx, salida.IDMensaje); errorRegistro != nil {
				slog.Error("no se pudo registrar el fallo de Outbox", "error", errorRegistro)
			}
			continue
		}
		if err := p.repositorio.MarcarPublicado(ctx, salida.IDMensaje); err != nil {
			slog.Error("el evento fue publicado pero no se pudo marcar", "idMensaje", salida.IDMensaje, "error", err)
		}
	}
}
