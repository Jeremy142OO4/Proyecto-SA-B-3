package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/messaging"
	"github.com/Proyecto-SA-B-3/account-service/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ConsumidorValidacionCliente struct {
	canal    *amqp.Channel
	servicio services.ServicioCreacionCuentas
}

func NuevoConsumidorValidacionCliente(conexion *amqp.Connection, servicio services.ServicioCreacionCuentas) (*ConsumidorValidacionCliente, error) {
	canal, err := conexion.Channel()
	if err != nil {
		return nil, fmt.Errorf("crear canal de validacion de cliente: %w", err)
	}
	if err := canal.Qos(10, 0, false); err != nil {
		_ = canal.Close()
		return nil, err
	}
	return &ConsumidorValidacionCliente{canal: canal, servicio: servicio}, nil
}

func (c *ConsumidorValidacionCliente) Cerrar() error { return c.canal.Close() }

func (c *ConsumidorValidacionCliente) Ejecutar(ctx context.Context) error {
	entregas, err := c.canal.ConsumeWithContext(ctx, messaging.ColaEventosCliente, "account-service.validacion-cliente", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consumir validaciones de cliente: %w", err)
	}
	for entrega := range entregas {
		var mensaje events.SobreMensaje
		if err := json.Unmarshal(entrega.Body, &mensaje); err != nil {
			_ = entrega.Nack(false, false)
			continue
		}
		if mensaje.Tipo == "" {
			mensaje.Tipo = entrega.RoutingKey
		}
		var resultado events.ResultadoValidacionCliente
		if err := json.Unmarshal(mensaje.Contenido, &resultado); err != nil {
			_ = entrega.Nack(false, false)
			continue
		}
		if err := c.servicio.ProcesarValidacionCliente(ctx, mensaje, resultado); err != nil {
			slog.Error("fallo la validacion de cliente", "error", err, "idCorrelacion", mensaje.IDCorrelacion)
			_ = entrega.Nack(false, true)
			continue
		}
		_ = entrega.Ack(false)
	}
	return nil
}
