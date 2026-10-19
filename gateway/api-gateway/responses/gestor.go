package responses

import (
	"log/slog"

	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/google/uuid"
	"sync"
)

type Gestor struct {
	mu      sync.Mutex
	esperas map[uuid.UUID]chan events.SobreMensaje
	tipos   map[uuid.UUID]string
}

func Nuevo() *Gestor {
	return &Gestor{esperas: map[uuid.UUID]chan events.SobreMensaje{}, tipos: map[uuid.UUID]string{}}
}
func (g *Gestor) Registrar(id uuid.UUID) (<-chan events.SobreMensaje, func()) {
	return g.RegistrarTipo(id, "")
}
func (g *Gestor) RegistrarTipo(id uuid.UUID, tipo string) (<-chan events.SobreMensaje, func()) {
	g.mu.Lock()
	g.tipos[id] = tipo
	ch := make(chan events.SobreMensaje, 1)
	g.esperas[id] = ch
	g.mu.Unlock()
	return ch, func() { g.mu.Lock(); delete(g.esperas, id); delete(g.tipos, id); g.mu.Unlock() }
}
func (g *Gestor) Entregar(m events.SobreMensaje) {
	g.mu.Lock()
	ch := g.esperas[m.IDCorrelacion]
	tipo := g.tipos[m.IDCorrelacion]
	g.mu.Unlock()
	if tipo != "" && m.Tipo != tipo {
		return
	}
	if ch != nil {
		select {
		case ch <- m:
		default:
		}
		return
	}
	slog.Warn("respuesta sin solicitud activa", "tipo", m.Tipo, "correlacion", m.IDCorrelacion)
}
