package publishers

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/messaging"
	"github.com/Proyecto-SA-B-3/transaction-service/repositories"
	"log/slog"
	"time"
)

type Outbox struct {
	repo      repositories.RepositorioOutbox
	pub       *messaging.Publicador
	intervalo time.Duration
}

func Nuevo(r repositories.RepositorioOutbox, p *messaging.Publicador, i time.Duration) *Outbox {
	return &Outbox{r, p, i}
}
func (o *Outbox) Ejecutar(ctx context.Context) {
	t := time.NewTicker(o.intervalo)
	defer t.Stop()
	o.publicar(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			o.publicar(ctx)
		}
	}
}
func (o *Outbox) publicar(ctx context.Context) {
	lista, e := o.repo.ListarPendientes(ctx, 50)
	if e != nil {
		slog.Error("listar outbox", "error", e)
		return
	}
	for _, x := range lista {
		m := events.SobreMensaje{IDMensaje: x.IDMensaje, IDCorrelacion: x.IDCorrelacion, Tipo: x.Tipo, Version: 1, OcurridoEn: x.FechaCreacion, Productor: "transaction-service", Contenido: json.RawMessage(x.Contenido)}
		if e = o.pub.Publicar(ctx, m, x.EsComando); e != nil {
			slog.Error("publicar outbox", "tipo", x.Tipo, "error", e)
			_ = o.repo.RegistrarFallo(ctx, x.IDMensaje)
			continue
		}
		if e = o.repo.MarcarPublicado(ctx, x.IDMensaje); e != nil {
			slog.Error("marcar outbox", "error", e)
		}
	}
}
