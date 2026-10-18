package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	IntercambioComandos   = "banco.comandos"
	IntercambioEventos    = "banco.eventos"
	IntercambioRespuestas = "banco.respuestas"
	IntercambioFallidos   = "banco.fallidos"
	ColaRespuestas        = "gateway.respuestas"
)

type Publicador struct{ sesion *sesionAMQP }

func NuevoCliente(url string, ops *operations.Store, gestor *responses.Gestor) (*Publicador, error) {
	s, err := nuevaSesion(url, ColaRespuestas, 20, DeclararTopologia, func(ctx context.Context, d amqp.Delivery) error {
		m, err := events.Decodificar(d.Body)
		if err != nil {
			return fmt.Errorf("respuesta invalida: %w", err)
		}
		gestor.Entregar(m)
		if estado := estadoOperacion(m.Tipo); estado != "" {
			var detalle struct {
				Codigo string `json:"codigo"`
			}
			_ = json.Unmarshal(m.Contenido, &detalle)
			ops.Actualizar(m.IDCorrelacion.String(), estado, detalle.Codigo)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	p := &Publicador{sesion: s}
	s.iniciar()
	return p, nil
}

func (p *Publicador) Listo() bool   { return p.sesion.listo() }
func (p *Publicador) Cerrar() error { p.sesion.cerrar(); return nil }
func (p *Publicador) Publicar(ctx context.Context, m events.SobreMensaje) error {
	return p.publicar(ctx, IntercambioComandos, m)
}
func (p *Publicador) PublicarEvento(ctx context.Context, m events.SobreMensaje) error {
	return p.publicar(ctx, IntercambioEventos, m)
}
func (p *Publicador) publicar(ctx context.Context, intercambio string, m events.SobreMensaje) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return p.sesion.publicar(ctx, intercambio, m.Tipo, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent,
		Timestamp: time.Now().UTC(), MessageId: m.IDMensaje.String(),
		CorrelationId: m.IDCorrelacion.String(), Body: b,
	})
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
