package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

const (
	IntercambioComandos = "banco.comandos"
	IntercambioEventos  = "banco.eventos"
	IntercambioFallidos = "banco.fallidos"
	ColaComandos        = "transacciones.comandos"
	ColaEventosCuenta   = "transacciones.eventos-cuenta"
	ColaFallidos        = "transacciones.fallidos"
)

func Conectar(url string) (*amqp.Connection, error) {
	c, e := amqp.Dial(url)
	if e != nil {
		return nil, fmt.Errorf("conectar RabbitMQ: %w", e)
	}
	return c, nil
}
func DeclararTopologia(ch *amqp.Channel) error {
	for _, x := range []string{IntercambioComandos, IntercambioEventos, IntercambioFallidos} {
		if e := ch.ExchangeDeclare(x, "topic", true, false, false, false, nil); e != nil {
			return e
		}
	}
	args := amqp.Table{"x-dead-letter-exchange": IntercambioFallidos, "x-dead-letter-routing-key": "transaccion.mensaje.fallido"}
	for _, q := range []string{ColaComandos, ColaEventosCuenta} {
		if _, e := ch.QueueDeclare(q, true, false, false, false, args); e != nil {
			return e
		}
	}
	if _, e := ch.QueueDeclare(ColaFallidos, true, false, false, false, nil); e != nil {
		return e
	}
	for _, k := range []string{events.ComandoTransferencia, events.ComandoTransferenciaPlan, events.ComandoConsultar, events.ComandoHistorial} {
		if e := ch.QueueBind(ColaComandos, k, IntercambioComandos, false, nil); e != nil {
			return e
		}
	}
	for _, k := range []string{events.EventoDebitada, events.EventoDebitoRechazado, events.EventoAcreditada, events.EventoCreditoRechazado, events.EventoCuentaCompensada, events.EventoCompensacionRechazada} {
		if e := ch.QueueBind(ColaEventosCuenta, k, IntercambioEventos, false, nil); e != nil {
			return e
		}
	}
	return ch.QueueBind(ColaFallidos, "transaccion.mensaje.fallido", IntercambioFallidos, false, nil)
}

type Publicador struct{ ch *amqp.Channel }

func NuevoPublicador(c *amqp.Connection) (*Publicador, error) {
	ch, e := c.Channel()
	if e != nil {
		return nil, e
	}
	if e = ch.Confirm(false); e != nil {
		ch.Close()
		return nil, e
	}
	return &Publicador{ch}, nil
}
func (p *Publicador) Cerrar() error { return p.ch.Close() }
func (p *Publicador) Publicar(ctx context.Context, m events.SobreMensaje, comando bool) error {
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	x := IntercambioEventos
	if comando {
		x = IntercambioComandos
	}
	return p.publicar(ctx, x, m.Tipo, b, nil, true)
}
func (p *Publicador) Republicar(ctx context.Context, x, k string, b []byte, h amqp.Table) error {
	return p.publicar(ctx, x, k, b, h, true)
}
func (p *Publicador) Fallido(ctx context.Context, b []byte, h amqp.Table) error {
	return p.publicar(ctx, IntercambioFallidos, "transaccion.mensaje.fallido", b, h, true)
}
func (p *Publicador) publicar(ctx context.Context, x, k string, b []byte, h amqp.Table, mandatory bool) error {
	conf, e := p.ch.PublishWithDeferredConfirmWithContext(ctx, x, k, mandatory, false, amqp.Publishing{Headers: h, ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.Now().UTC(), MessageId: uuid.NewString(), Body: b})
	if e != nil {
		return e
	}
	if conf == nil || !conf.Wait() {
		return fmt.Errorf("RabbitMQ no confirmo %s", k)
	}
	return nil
}
