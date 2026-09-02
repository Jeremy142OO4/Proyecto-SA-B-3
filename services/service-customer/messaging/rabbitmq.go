package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"bank-usac/service-customer/events"
	"bank-usac/service-customer/repositories"
	"bank-usac/service-customer/services"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	intercambioComandos   = "banco.comandos"
	intercambioEventos    = "banco.eventos"
	intercambioRespuestas = "banco.respuestas"
	intercambioFallidos   = "banco.fallidos"
	colaComandosCliente   = "clientes.comandos"
	colaFallidosCliente   = "clientes.comandos.fallidos"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	repo    repositories.CustomerRepository
	svc     services.CustomerService
}

func NewRabbitMQClient(url string, repo repositories.CustomerRepository, svc services.CustomerService) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declarar Exchanges estándar
	exchanges := []string{intercambioComandos, intercambioEventos, intercambioRespuestas, intercambioFallidos}
	for _, ex := range exchanges {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return nil, err
		}
	}
	if _, err := ch.QueueDeclare(colaFallidosCliente, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.QueueBind(colaFallidosCliente, "cliente.#", intercambioFallidos, false, nil); err != nil {
		return nil, err
	}
	argumentos := amqp.Table{
		"x-dead-letter-exchange":    intercambioFallidos,
		"x-dead-letter-routing-key": "cliente.validacion.fallida",
	}
	if _, err := ch.QueueDeclare(colaComandosCliente, true, false, false, false, argumentos); err != nil {
		return nil, err
	}
	for _, tipo := range []string{events.ComandoValidarCliente, events.ComandoRegistrarCliente, events.ComandoActivarCliente, events.ComandoLoginCliente, events.ComandoPerfilCliente, events.ComandoActualizarCliente, events.ComandoListarClientes, events.ComandoEstadoCliente} {
		if err := ch.QueueBind(colaComandosCliente, tipo, intercambioComandos, false, nil); err != nil {
			return nil, err
		}
	}

	return &RabbitMQClient{conn: conn, channel: ch, repo: repo, svc: svc}, nil
}

func (r *RabbitMQClient) StartValidationConsumer(ctx context.Context) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	if err = ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		return err
	}
	entregas, err := ch.ConsumeWithContext(ctx, colaComandosCliente, "customer-service.validacion-cliente", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return err
	}
	go func() {
		defer ch.Close()
		for entrega := range entregas {
			var sobre events.EventEnvelope
			if json.Unmarshal(entrega.Body, &sobre) != nil || sobre.MessageID == uuid.Nil || sobre.CorrelationID == uuid.Nil {
				_ = entrega.Nack(false, false)
				continue
			}
			if err := r.procesarComando(ctx, sobre); err != nil {
				log.Printf("[ComandoCliente] Error: %v", err)
				_ = entrega.Nack(false, false)
				continue
			}
			_ = entrega.Ack(false)
		}
	}()
	return nil
}

type respuestaRPC struct {
	Estado int             `json:"estado"`
	Cuerpo json.RawMessage `json:"cuerpo"`
}

func (r *RabbitMQClient) procesarComando(ctx context.Context, sobre events.EventEnvelope) error {
	if sobre.Type == events.ComandoValidarCliente {
		var solicitud events.SolicitudValidacionCliente
		if json.Unmarshal(sobre.Payload, &solicitud) != nil || solicitud.IDSolicitud == uuid.Nil || solicitud.IDCliente == uuid.Nil {
			return errors.New("solicitud de validacion invalida")
		}
		return r.repo.RegistrarValidacionCliente(ctx, sobre.MessageID, sobre.CorrelationID, solicitud.IDSolicitud, solicitud.IDCliente)
	}
	estado, cuerpo := r.ejecutarRPC(ctx, sobre)
	contenido, _ := json.Marshal(respuestaRPC{Estado: estado, Cuerpo: cuerpo})
	respuesta, err := events.NewEnvelope(strings.TrimSuffix(sobre.Type, ".solicitado")+".respondido", sobre.CorrelationID, &sobre.MessageID, json.RawMessage(contenido))
	if err != nil {
		return err
	}
	bytes, _ := json.Marshal(respuesta)
	return r.channel.PublishWithContext(ctx, intercambioRespuestas, respuesta.Type, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", CorrelationId: sobre.CorrelationID.String(), Body: bytes})
}

func (r *RabbitMQClient) ejecutarRPC(ctx context.Context, sobre events.EventEnvelope) (int, json.RawMessage) {
	errorRespuesta := func(estado int, err error) (int, json.RawMessage) {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return estado, b
	}
	respuesta := func(estado int, valor any) (int, json.RawMessage) {
		b, _ := json.Marshal(valor)
		return estado, b
	}
	switch sobre.Type {
	case events.ComandoRegistrarCliente:
		var req services.RegisterRequest
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		cliente, err := r.svc.RegisterCustomer(ctx, req, sobre.CorrelationID)
		if err != nil {
			return errorRespuesta(400, err)
		}
		return respuesta(201, map[string]any{"message": "Cliente registrado exitosamente. Se ha enviado un enlace de activacion al correo.", "customer": cliente})
	case events.ComandoActivarCliente:
		var req struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil || req.Token == "" {
			return errorRespuesta(400, errors.New("token requerido"))
		}
		if err := r.svc.ActivateCustomer(ctx, req.Token, sobre.CorrelationID); err != nil {
			return errorRespuesta(400, err)
		}
		return respuesta(200, map[string]string{"message": "Cuenta activada exitosamente. Ahora puede iniciar sesion."})
	case events.ComandoLoginCliente:
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		login, err := r.svc.Login(ctx, req.Username, req.Password)
		if err != nil {
			return errorRespuesta(401, err)
		}
		return respuesta(200, login)
	case events.ComandoPerfilCliente:
		var req struct {
			IDCliente uuid.UUID `json:"idCliente"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		cliente, err := r.svc.GetProfile(ctx, req.IDCliente)
		if err != nil || cliente == nil {
			return errorRespuesta(404, errors.New("perfil no encontrado"))
		}
		return respuesta(200, cliente)
	case events.ComandoActualizarCliente:
		var req struct {
			IDCliente uuid.UUID `json:"idCliente"`
			services.UpdateRequest
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		cliente, err := r.svc.UpdateCustomer(ctx, req.IDCliente, req.UpdateRequest, sobre.CorrelationID)
		if err != nil {
			return errorRespuesta(400, err)
		}
		return respuesta(200, map[string]any{"message": "Perfil actualizado correctamente", "customer": cliente})
	case events.ComandoListarClientes:
		var req struct {
			Limite         int `json:"limite"`
			Desplazamiento int `json:"desplazamiento"`
		}
		_ = json.Unmarshal(sobre.Payload, &req)
		clientes, err := r.svc.ListCustomers(ctx, req.Limite, req.Desplazamiento)
		if err != nil {
			return errorRespuesta(500, err)
		}
		return respuesta(200, clientes)
	case events.ComandoEstadoCliente:
		var req struct {
			IDCliente uuid.UUID `json:"idCliente"`
			Estado    string    `json:"estado"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		cliente, err := r.svc.UpdateCustomerStatus(ctx, req.IDCliente, req.Estado)
		if err != nil {
			return errorRespuesta(400, err)
		}
		return respuesta(200, cliente)
	default:
		return errorRespuesta(400, errors.New("comando de cliente no soportado"))
	}
}

func (r *RabbitMQClient) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// StartOutboxWorker envía periódicamente los eventos pendientes a RabbitMQ
func (r *RabbitMQClient) StartOutboxWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.processPendingOutbox(ctx)
			}
		}
	}()
}

func (r *RabbitMQClient) processPendingOutbox(ctx context.Context) {
	msgs, err := r.repo.GetPendingOutbox(ctx, 20)
	if err != nil || len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		exchange := intercambioEventos
		routingKey := msg.EventType

		err := r.channel.PublishWithContext(
			ctx,
			exchange,
			routingKey,
			false,
			false,
			amqp.Publishing{
				DeliveryMode:  amqp.Persistent,
				ContentType:   "application/json",
				CorrelationId: msg.CorrelationID.String(),
				Timestamp:     msg.CreatedAt,
				Body:          msg.Payload,
			},
		)

		if err != nil {
			log.Printf("[Outbox] Error publicando evento %s: %v", msg.EventType, err)
			_ = r.repo.IncrementOutboxAttempt(ctx, msg.ID, err.Error())
		} else {
			_ = r.repo.MarkOutboxPublished(ctx, msg.ID)
		}
	}
}
