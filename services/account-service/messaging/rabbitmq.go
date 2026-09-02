package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	IntercambioComandos = "banco.comandos"
	IntercambioEventos  = "banco.eventos"
	IntercambioFallidos = "banco.fallidos"
	ColaComandosCuenta  = "cuentas.comandos"
	ColaFallidosCuenta  = "cuentas.comandos.fallidos"
	ColaEventosCliente  = "cuentas.eventos-cliente"
)

func ConectarRabbitMQ(urlRabbitMQ string) (*amqp.Connection, error) {
	conexion, err := amqp.Dial(urlRabbitMQ)
	if err != nil {
		return nil, fmt.Errorf("conectar con RabbitMQ: %w", err)
	}
	return conexion, nil
}

func DeclararTopologia(canal *amqp.Channel) error {
	for _, intercambio := range []string{IntercambioComandos, IntercambioEventos, IntercambioFallidos} {
		if err := canal.ExchangeDeclare(intercambio, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("declarar intercambio %s: %w", intercambio, err)
		}
	}

	argumentos := amqp.Table{
		"x-dead-letter-exchange":    IntercambioFallidos,
		"x-dead-letter-routing-key": "cuenta.comando.fallido",
	}
	if _, err := canal.QueueDeclare(ColaComandosCuenta, true, false, false, false, argumentos); err != nil {
		return fmt.Errorf("declarar cola de comandos de cuenta: %w", err)
	}
	if _, err := canal.QueueDeclare(ColaFallidosCuenta, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declarar cola de comandos fallidos: %w", err)
	}
	if _, err := canal.QueueDeclare(ColaEventosCliente, true, false, false, false, argumentos); err != nil {
		return fmt.Errorf("declarar cola de eventos de cliente: %w", err)
	}

	comandos := []string{
		events.ComandoCrearCuenta,
		events.ComandoSolicitarDebito,
		events.ComandoSolicitarCredito,
		events.ComandoSolicitarCompensacion,
		events.ComandoConsultarCuenta,
		events.ComandoListarMovimientos,
		events.ComandoListarCuentas,
	}
	for _, evento := range []string{events.EventoClienteValidado, events.EventoClienteRechazado} {
		if err := canal.QueueBind(ColaEventosCliente, evento, IntercambioEventos, false, nil); err != nil {
			return fmt.Errorf("enlazar evento %s: %w", evento, err)
		}
	}
	for _, comando := range comandos {
		if err := canal.QueueBind(ColaComandosCuenta, comando, IntercambioComandos, false, nil); err != nil {
			return fmt.Errorf("enlazar comando %s: %w", comando, err)
		}
	}
	if err := canal.QueueBind(ColaFallidosCuenta, "cuenta.comando.fallido", IntercambioFallidos, false, nil); err != nil {
		return fmt.Errorf("enlazar cola de fallidos: %w", err)
	}
	return nil
}

type Publicador struct {
	canal *amqp.Channel
}

func NuevoPublicador(conexion *amqp.Connection) (*Publicador, error) {
	canal, err := conexion.Channel()
	if err != nil {
		return nil, fmt.Errorf("crear canal publicador: %w", err)
	}
	if err := canal.Confirm(false); err != nil {
		_ = canal.Close()
		return nil, fmt.Errorf("activar confirmaciones del publicador: %w", err)
	}
	return &Publicador{canal: canal}, nil
}

func (p *Publicador) Cerrar() error {
	return p.canal.Close()
}

func (p *Publicador) PublicarEvento(ctx context.Context, mensaje events.SobreMensaje) error {
	return p.publicar(ctx, IntercambioEventos, mensaje.Tipo, mensaje, nil, false)
}

func (p *Publicador) PublicarComando(ctx context.Context, mensaje events.SobreMensaje) error {
	return p.publicar(ctx, IntercambioComandos, mensaje.Tipo, mensaje, nil, true)
}

func (p *Publicador) ReintentarComando(ctx context.Context, clave string, cuerpo []byte, encabezados amqp.Table) error {
	return p.publicarCuerpo(ctx, IntercambioComandos, clave, cuerpo, encabezados, true)
}

func (p *Publicador) EnviarFallido(ctx context.Context, cuerpo []byte, encabezados amqp.Table) error {
	return p.publicarCuerpo(ctx, IntercambioFallidos, "cuenta.comando.fallido", cuerpo, encabezados, true)
}

func (p *Publicador) publicar(ctx context.Context, intercambio, clave string, mensaje any, encabezados amqp.Table, obligatorio bool) error {
	cuerpo, err := json.Marshal(mensaje)
	if err != nil {
		return fmt.Errorf("serializar mensaje: %w", err)
	}
	return p.publicarCuerpo(ctx, intercambio, clave, cuerpo, encabezados, obligatorio)
}

func (p *Publicador) publicarCuerpo(ctx context.Context, intercambio, clave string, cuerpo []byte, encabezados amqp.Table, obligatorio bool) error {
	confirmacion, err := p.canal.PublishWithDeferredConfirmWithContext(ctx, intercambio, clave, obligatorio, false, amqp.Publishing{
		Headers:      encabezados,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		MessageId:    uuid.NewString(),
		Body:         cuerpo,
	})
	if err != nil {
		return fmt.Errorf("publicar mensaje %s: %w", clave, err)
	}
	if confirmacion == nil || !confirmacion.Wait() {
		return fmt.Errorf("RabbitMQ no confirmo el mensaje %s", clave)
	}
	return nil
}
