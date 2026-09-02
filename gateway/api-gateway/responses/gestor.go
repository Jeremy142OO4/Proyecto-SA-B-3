package responses

import (
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/google/uuid"
	"sync"
)

type Gestor struct {
	mu      sync.Mutex
	esperas map[uuid.UUID]chan events.SobreMensaje
}

func Nuevo() *Gestor { return &Gestor{esperas: map[uuid.UUID]chan events.SobreMensaje{}} }
func (g *Gestor) Registrar(id uuid.UUID) (<-chan events.SobreMensaje, func()) {
	g.mu.Lock()
	ch := make(chan events.SobreMensaje, 1)
	g.esperas[id] = ch
	g.mu.Unlock()
	return ch, func() { g.mu.Lock(); delete(g.esperas, id); g.mu.Unlock() }
}
func (g *Gestor) Entregar(m events.SobreMensaje) {
	g.mu.Lock()
	ch := g.esperas[m.IDCorrelacion]
	g.mu.Unlock()
	if ch != nil {
		select {
		case ch <- m:
		default:
		}
	}
}
