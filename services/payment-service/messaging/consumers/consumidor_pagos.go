package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/messaging"
	"github.com/Proyecto-SA-B-3/payment-service/services"
	amqp "github.com/rabbitmq/amqp091-go"
	"log/slog"
	"strings"
)

type ConsumidorPagos struct {
	canal            *amqp.Channel
	servicio         services.ServicioPagos
	publicador       *messaging.Publicador
	maximoReintentos int
}

func NuevoConsumidorPagos(c *amqp.Connection, s services.ServicioPagos, publicador *messaging.Publicador, maximo int) (*ConsumidorPagos, error) {
	ch, e := c.Channel()
	if e != nil {
		return nil, e
	}
	_ = ch.Qos(10, 0, false)
	return &ConsumidorPagos{canal: ch, servicio: s, publicador: publicador, maximoReintentos: maximo}, nil
}
func (c *ConsumidorPagos) Cerrar() error { return c.canal.Close() }
func (c *ConsumidorPagos) Ejecutar(ctx context.Context, cola, nombre string) error {
	ds, e := c.canal.ConsumeWithContext(ctx, cola, nombre, false, false, false, false, nil)
	if e != nil {
		return e
	}
	for d := range ds {
		if e = c.procesar(ctx, d); e != nil {
			slog.Error("fallo mensaje de pago", "error", e, "tipo", d.RoutingKey)
			c.resolverFallo(ctx, d, e)
		} else {
			_ = d.Ack(false)
		}
	}
	return nil
}

func (c *ConsumidorPagos) resolverFallo(ctx context.Context, entrega amqp.Delivery, causa error) {
	intentos := 0
	if valor, ok := entrega.Headers["x-intentos"].(int32); ok {
		intentos = int(valor)
	}
	encabezados := amqp.Table{}
	for clave, valor := range entrega.Headers {
		encabezados[clave] = valor
	}
	encabezados["x-ultimo-error"] = causa.Error()
	permanente := errors.Is(causa, services.ErrSolicitudInvalida) || errors.Is(causa, services.ErrTipoPagoInvalido) || strings.Contains(causa.Error(), "invalid character")
	var err error
	if permanente || intentos >= c.maximoReintentos {
		err = c.publicador.EnviarFallido(ctx, entrega.Body, encabezados)
	} else {
		encabezados["x-intentos"] = int32(intentos + 1)
		intercambio := entrega.Exchange
		if intercambio == "" {
			intercambio = messaging.IntercambioComandos
		}
		err = c.publicador.Republicar(ctx, intercambio, entrega.RoutingKey, entrega.Body, encabezados)
	}
	if err != nil {
		_ = entrega.Nack(false, true)
	} else {
		_ = entrega.Ack(false)
	}
}
func (c *ConsumidorPagos) procesar(ctx context.Context, d amqp.Delivery) error {
	var m events.SobreMensaje
	if e := json.Unmarshal(d.Body, &m); e != nil {
		return e
	}
	if m.Tipo == "" {
		m.Tipo = d.RoutingKey
	}
	if d.RoutingKey == events.ComandoProcesarPago {
		var s events.SolicitudPago
		if e := json.Unmarshal(m.Contenido, &s); e != nil {
			return e
		}
		return c.servicio.Procesar(ctx, m, s)
	}
	if d.RoutingKey == events.ComandoConsultarPago {
		var s events.SolicitudConsultarPago
		if e := json.Unmarshal(m.Contenido, &s); e != nil {
			return e
		}
		pago, e := c.servicio.Consultar(ctx, s.IDPago)
		if e != nil {
			return e
		}
		return c.servicio.RegistrarRespuesta(ctx, m, events.EventoPagoConsultado, pago)
	}
	if d.RoutingKey == events.ComandoListarPagos {
		var s events.SolicitudListarPagos
		if e := json.Unmarshal(m.Contenido, &s); e != nil {
			return e
		}
		lista, e := c.servicio.ListarCliente(ctx, s.IDCliente, s.Limite, s.Desplazamiento)
		if e != nil {
			return e
		}
		return c.servicio.RegistrarRespuesta(ctx, m, events.EventoHistorialConsultado, map[string]any{"idCliente": s.IDCliente, "pagos": lista})
	}
	for _, tipo := range []string{events.EventoCuentaDebitada, events.EventoDebitoRechazado, events.EventoCuentaCompensada} {
		if d.RoutingKey == tipo {
			var r events.ResultadoMovimiento
			if e := json.Unmarshal(m.Contenido, &r); e != nil {
				return e
			}
			return c.servicio.ProcesarResultadoCuenta(ctx, m, r)
		}
	}
	return fmt.Errorf("mensaje no soportado: %s", d.RoutingKey)
}
