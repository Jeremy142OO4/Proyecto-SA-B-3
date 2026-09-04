package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/messaging"
	"github.com/Proyecto-SA-B-3/transaction-service/repositories"
	"github.com/Proyecto-SA-B-3/transaction-service/services"
	amqp "github.com/rabbitmq/amqp091-go"
	"log/slog"
)

type Consumidor struct {
	ch         *amqp.Channel
	servicio   *services.Servicio
	publicador *messaging.Publicador
	max        int
}

func Nuevo(c *amqp.Connection, s *services.Servicio, p *messaging.Publicador, max int) (*Consumidor, error) {
	ch, e := c.Channel()
	if e != nil {
		return nil, e
	}
	if e = ch.Qos(10, 0, false); e != nil {
		ch.Close()
		return nil, e
	}
	return &Consumidor{ch, s, p, max}, nil
}
func (c *Consumidor) Cerrar() error { return c.ch.Close() }
func (c *Consumidor) Ejecutar(ctx context.Context) error {
	for _, q := range []string{messaging.ColaComandos, messaging.ColaEventosCuenta} {
		d, e := c.ch.ConsumeWithContext(ctx, q, "transaction-service."+q, false, false, false, false, nil)
		if e != nil {
			return e
		}
		go func(entregas <-chan amqp.Delivery) {
			for d := range entregas {
				if e := c.procesar(ctx, d); e != nil {
					slog.Error("mensaje no resuelto", "tipo", d.RoutingKey, "error", e)
				}
			}
		}(d)
	}
	<-ctx.Done()
	return nil
}
func (c *Consumidor) procesar(ctx context.Context, d amqp.Delivery) error {
	m, e := events.DecodificarSobre(d.Body)
	if e != nil {
		return c.fallido(ctx, d, e)
	}
	if m.Tipo == "" {
		m.Tipo = d.RoutingKey
	}
	switch d.RoutingKey {
	case events.ComandoTransferencia, events.ComandoTransferenciaPlan:
		var p events.SolicitudTransferencia
		e = json.Unmarshal(m.Contenido, &p)
		if e == nil {
			_, e = c.servicio.Solicitar(ctx, m, p)
		}
	case events.ComandoConsultar:
		var p events.SolicitudConsulta
		e = json.Unmarshal(m.Contenido, &p)
		if e == nil {
			_, e = c.servicio.Consultar(ctx, m, p)
		}
	case events.ComandoHistorial:
		var p events.SolicitudHistorial
		e = json.Unmarshal(m.Contenido, &p)
		if e == nil {
			_, e = c.servicio.Historial(ctx, m, p)
		}
	case events.EventoDebitada, events.EventoDebitoRechazado, events.EventoAcreditada, events.EventoCreditoRechazado, events.EventoCuentaCompensada, events.EventoCompensacionRechazada:
		var p events.ResultadoMovimiento
		e = json.Unmarshal(m.Contenido, &p)
		if e == nil {
			_, e = c.servicio.Resultado(ctx, m, p)
		}
	default:
		e = fmt.Errorf("tipo no soportado %s", d.RoutingKey)
	}
	if e == nil {
		return d.Ack(false)
	}
	// Los resultados de cuenta tambien son consumidos por payment-service. Una
	// operacion ajena no es una transferencia fallida y solo debe ignorarse.
	if esEventoCuenta(d.RoutingKey) && errors.Is(e, repositories.ErrNoEncontrada) {
		return d.Ack(false)
	}
	if errors.Is(e, services.ErrSolicitudInvalida) || errors.Is(e, repositories.ErrNoEncontrada) {
		return c.fallido(ctx, d, e)
	}
	return c.reintentar(ctx, d, e)
}

func esEventoCuenta(tipo string) bool {
	switch tipo {
	case events.EventoDebitada, events.EventoDebitoRechazado, events.EventoAcreditada, events.EventoCreditoRechazado, events.EventoCuentaCompensada, events.EventoCompensacionRechazada:
		return true
	default:
		return false
	}
}
func (c *Consumidor) reintentar(ctx context.Context, d amqp.Delivery, causa error) error {
	n := intentos(d.Headers)
	h := copiar(d.Headers)
	h["x-intentos"] = int32(n + 1)
	h["x-ultimo-error"] = causa.Error()
	var e error
	if n < c.max {
		x := messaging.IntercambioEventos
		if d.RoutingKey == events.ComandoTransferencia || d.RoutingKey == events.ComandoTransferenciaPlan || d.RoutingKey == events.ComandoConsultar || d.RoutingKey == events.ComandoHistorial {
			x = messaging.IntercambioComandos
		}
		e = c.publicador.Republicar(ctx, x, d.RoutingKey, d.Body, h)
	} else {
		e = c.publicador.Fallido(ctx, d.Body, h)
	}
	if e != nil {
		d.Nack(false, true)
		return e
	}
	return d.Ack(false)
}
func (c *Consumidor) fallido(ctx context.Context, d amqp.Delivery, causa error) error {
	h := copiar(d.Headers)
	h["x-ultimo-error"] = causa.Error()
	if e := c.publicador.Fallido(ctx, d.Body, h); e != nil {
		d.Nack(false, true)
		return e
	}
	return d.Ack(false)
}
func intentos(h amqp.Table) int {
	switch v := h["x-intentos"].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	}
	return 0
}
func copiar(h amqp.Table) amqp.Table {
	o := amqp.Table{}
	for k, v := range h {
		o[k] = v
	}
	return o
}
