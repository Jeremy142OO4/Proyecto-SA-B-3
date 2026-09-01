package publishers

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/messaging"
	"github.com/Proyecto-SA-B-3/payment-service/repositories"
	"log/slog"
	"time"
)

type PublicadorOutbox struct {
	repo       repositories.RepositorioPagos
	publicador *messaging.Publicador
	intervalo  time.Duration
}

func NuevoPublicadorOutbox(r repositories.RepositorioPagos, p *messaging.Publicador, i time.Duration) *PublicadorOutbox {
	return &PublicadorOutbox{r, p, i}
}
func (p *PublicadorOutbox) Ejecutar(ctx context.Context) {
	t := time.NewTicker(p.intervalo)
	defer t.Stop()
	for {
		p.publicar(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
func (p *PublicadorOutbox) publicar(ctx context.Context) {
	lista, e := p.repo.ListarSalida(ctx, 50)
	if e != nil {
		slog.Error("listar Outbox pagos", "error", e)
		return
	}
	for _, s := range lista {
		m := events.SobreMensaje{IDMensaje: s.IDMensaje, IDCorrelacion: s.IDCorrelacion, Tipo: s.TipoEvento, Version: s.VersionEvento, OcurridoEn: s.FechaCreacion, Productor: "payment-service", Contenido: json.RawMessage(s.Contenido)}
		comando := s.TipoEvento == events.ComandoSolicitarDebito || s.TipoEvento == events.ComandoSolicitarCompensacion
		if e = p.publicador.Publicar(ctx, m, comando); e != nil {
			_ = p.repo.RegistrarFallo(ctx, s.IDMensaje)
			continue
		}
		_ = p.repo.MarcarPublicado(ctx, s.IDMensaje)
	}
}
