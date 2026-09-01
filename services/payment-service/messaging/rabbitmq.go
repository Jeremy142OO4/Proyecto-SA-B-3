package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

const (
	IntercambioComandos = "banco.comandos"
	IntercambioEventos  = "banco.eventos"
	IntercambioFallidos = "banco.fallidos"
	ColaComandos        = "pagos.comandos"
	ColaEventosCuenta   = "pagos.eventos-cuenta"
	ColaFallidos        = "pagos.fallidos"
)

func Conectar(url string) (*amqp.Connection, error) {
	c, e := amqp.Dial(url)
	if e != nil {
		return nil, fmt.Errorf("conectar RabbitMQ pagos: %w", e)
	}
	return c, nil
}
func DeclararTopologia(ch *amqp.Channel) error {
	for _, x := range []string{IntercambioComandos, IntercambioEventos, IntercambioFallidos} {
		if e := ch.ExchangeDeclare(x, "topic", true, false, false, false, nil); e != nil {
			return e
		}
	}
	args := amqp.Table{"x-dead-letter-exchange": IntercambioFallidos, "x-dead-letter-routing-key": "pago.mensaje.fallido"}
	for _, q := range []string{ColaComandos, ColaEventosCuenta} {
		if _, e := ch.QueueDeclare(q, true, false, false, false, args); e != nil {
			return e
		}
	}
	if _, e := ch.QueueDeclare(ColaFallidos, true, false, false, false, nil); e != nil {
		return e
	}
	if e := ch.QueueBind(ColaComandos, events.ComandoProcesarPago, IntercambioComandos, false, nil); e != nil {
		return e
	}
	for _, clave := range []string{events.ComandoConsultarPago, events.ComandoListarPagos} {
		if e := ch.QueueBind(ColaComandos, clave, IntercambioComandos, false, nil); e != nil {
			return e
		}
	}
	for _, k := range []string{events.EventoCuentaDebitada, events.EventoDebitoRechazado, events.EventoCuentaCompensada} {
		if e := ch.QueueBind(ColaEventosCuenta, k, IntercambioEventos, false, nil); e != nil {
			return e
		}
	}
	return ch.QueueBind(ColaFallidos, "pago.mensaje.fallido", IntercambioFallidos, false, nil)
}

type Publicador struct{ canal *amqp.Channel }

func NuevoPublicador(c *amqp.Connection) (*Publicador, error) {
	ch, e := c.Channel()
	if e != nil {
		return nil, e
	}
	if e = ch.Confirm(false); e != nil {
		return nil, e
	}
	return &Publicador{ch}, nil
}
func (p *Publicador) Cerrar() error { return p.canal.Close() }
func (p *Publicador) Publicar(ctx context.Context, m events.SobreMensaje, comando bool) error {
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	x := IntercambioEventos
	if comando {
		x = IntercambioComandos
	}
	conf, e := p.canal.PublishWithDeferredConfirmWithContext(ctx, x, m.Tipo, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.Now().UTC(), MessageId: uuid.NewString(), Body: b})
	if e != nil {
		return e
	}
	if conf == nil || !conf.Wait() {
		return fmt.Errorf("RabbitMQ no confirmo %s", m.Tipo)
	}
	return nil
}

func (p *Publicador) Republicar(ctx context.Context, intercambio, clave string, cuerpo []byte, encabezados amqp.Table) error {
	confirmacion, err := p.canal.PublishWithDeferredConfirmWithContext(ctx, intercambio, clave, false, false, amqp.Publishing{Headers: encabezados, ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.Now().UTC(), MessageId: uuid.NewString(), Body: cuerpo})
	if err != nil {
		return err
	}
	if confirmacion == nil || !confirmacion.Wait() {
		return fmt.Errorf("RabbitMQ no confirmo el reintento")
	}
	return nil
}

func (p *Publicador) EnviarFallido(ctx context.Context, cuerpo []byte, encabezados amqp.Table) error {
	return p.Republicar(ctx, IntercambioFallidos, "pago.mensaje.fallido", cuerpo, encabezados)
}
