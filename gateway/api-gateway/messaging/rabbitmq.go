package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

const (
	IntercambioComandos   = "banco.comandos"
	IntercambioEventos    = "banco.eventos"
	IntercambioRespuestas = "banco.respuestas"
	IntercambioFallidos   = "banco.fallidos"
	ColaRespuestas        = "gateway.respuestas"
)

type Publicador struct{ canal *amqp.Channel }

func Conectar(url string) (*amqp.Connection, error) {
	c, e := amqp.Dial(url)
	if e != nil {
		return nil, fmt.Errorf("conectar RabbitMQ: %w", e)
	}
	return c, nil
}
func DeclararTopologia(ch *amqp.Channel) error {
	for _, x := range []string{IntercambioComandos, IntercambioEventos, IntercambioRespuestas, IntercambioFallidos} {
		if e := ch.ExchangeDeclare(x, "topic", true, false, false, false, nil); e != nil {
			return e
		}
	}
	args := amqp.Table{"x-dead-letter-exchange": IntercambioFallidos, "x-dead-letter-routing-key": "gateway.respuesta.fallida"}
	if _, e := ch.QueueDeclare(ColaRespuestas, true, false, false, false, args); e != nil {
		return e
	}
	for _, k := range []string{"cuenta.#", "pago.#", "transferencia.#"} {
		if e := ch.QueueBind(ColaRespuestas, k, IntercambioEventos, false, nil); e != nil {
			return e
		}
	}
	for _, k := range []string{"cliente.#", "auditoria.#"} {
		if e := ch.QueueBind(ColaRespuestas, k, IntercambioRespuestas, false, nil); e != nil {
			return e
		}
	}
	return nil
}
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
func (p *Publicador) Publicar(ctx context.Context, m events.SobreMensaje) error {
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	conf, e := p.canal.PublishWithDeferredConfirmWithContext(ctx, IntercambioComandos, m.Tipo, true, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.Now().UTC(), MessageId: m.IDMensaje.String(), CorrelationId: m.IDCorrelacion.String(), Body: b})
	if e != nil {
		return e
	}
	if conf == nil || !conf.Wait() {
		return fmt.Errorf("RabbitMQ no confirmo %s", m.Tipo)
	}
	return nil
}
func (p *Publicador) Cerrar() error { return p.canal.Close() }
func ConsumirRespuestas(c *amqp.Connection, ops *operations.Store, gestor *responses.Gestor) error {
	if c == nil || c.IsClosed() {
		return fmt.Errorf("conexion RabbitMQ cerrada")
	}
	// El consumidor se supervisa en segundo plano. Si RabbitMQ cierra el
	// canal (reinicio, failover o una desconexion temporal), se vuelve a crear
	// el canal y el consumidor sin tener que reiniciar el API Gateway.
	go func() {
		const esperaReconectar = 2 * time.Second
		for {
			if c.IsClosed() {
				log.Printf("RabbitMQ cerrado; consumidor de respuestas detenido")
				return
			}
			ch, e := c.Channel()
			if e != nil {
				log.Printf("abrir canal de respuestas: %v; reintentando", e)
				time.Sleep(esperaReconectar)
				continue
			}
			if e = ch.Qos(20, 0, false); e != nil {
				_ = ch.Close()
				log.Printf("configurar QoS de respuestas: %v; reintentando", e)
				time.Sleep(esperaReconectar)
				continue
			}
			// Exchanges, cola y bindings son idempotentes; declararlos al
			// reconectar también cubre una recreacion de RabbitMQ.
			if e = DeclararTopologia(ch); e != nil {
				_ = ch.Close()
				log.Printf("declarar topologia de respuestas: %v; reintentando", e)
				time.Sleep(esperaReconectar)
				continue
			}
			ds, e := ch.Consume(ColaRespuestas, "api-gateway", false, false, false, false, nil)
			if e != nil {
				_ = ch.Close()
				log.Printf("consumir respuestas: %v; reintentando", e)
				time.Sleep(esperaReconectar)
				continue
			}

			for d := range ds {
				m, e := events.Decodificar(d.Body)
				if e != nil {
					log.Printf("respuesta invalida: %v", e)
					_ = d.Nack(false, false)
					continue
				}
				gestor.Entregar(m)
				estado := estadoOperacion(m.Tipo)
				if estado != "" {
					var detalle struct {
						Codigo string `json:"codigo"`
					}
					_ = json.Unmarshal(m.Contenido, &detalle)
					ops.Actualizar(m.IDCorrelacion.String(), estado, detalle.Codigo)
				}
				if e = d.Ack(false); e != nil {
					log.Printf("confirmar respuesta %s: %v", m.IDCorrelacion, e)
				}
			}
			_ = ch.Close()
			log.Printf("canal de respuestas cerrado; reconectando")
			time.Sleep(esperaReconectar)
		}
	}()
	return nil
}
func estadoOperacion(t string) string {
	switch t {
	case "cuenta.creada", "cuenta.acreditada", "pago.completado", "transferencia.completada":
		return "COMPLETADO"
	case "cuenta.creacion.rechazada", "cuenta.credito.rechazado", "pago.rechazado", "transferencia.rechazada":
		return "RECHAZADO"
	case "pago.procesando", "transferencia.procesando":
		return "PROCESANDO"
	case "transferencia.compensando":
		return "COMPENSANDO"
	case "transferencia.compensada":
		return "COMPENSADA"
	case "transferencia.compensacion.fallida":
		return "COMPENSACION_FALLIDA"
	}
	return ""
}

var _ = uuid.Nil
