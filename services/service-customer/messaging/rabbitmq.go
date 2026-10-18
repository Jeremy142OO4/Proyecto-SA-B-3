package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	sesion *sesionAMQP
	repo   repositories.CustomerRepository
	svc    services.CustomerService
}

func NewRabbitMQClient(url string, repo repositories.CustomerRepository, svc services.CustomerService) (*RabbitMQClient, error) {
	r := &RabbitMQClient{repo: repo, svc: svc}
	s, err := nuevaSesion(url, colaComandosCliente, 10, declararTopologia, func(ctx context.Context, d amqp.Delivery) error {
		var sobre events.EventEnvelope
		if json.Unmarshal(d.Body, &sobre) != nil || sobre.MessageID == uuid.Nil || sobre.CorrelationID == uuid.Nil {
			return errors.New("comando de cliente invalido")
		}
		return r.procesarComando(ctx, sobre)
	})
	if err != nil {
		return nil, err
	}
	r.sesion = s
	s.iniciar()
	return r, nil
}

func declararTopologia(ch *amqp.Channel) error {
	// Declarar Exchanges estándar
	exchanges := []string{intercambioComandos, intercambioEventos, intercambioRespuestas, intercambioFallidos}
	for _, ex := range exchanges {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}
	if _, err := ch.QueueDeclare(colaFallidosCliente, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(colaFallidosCliente, "cliente.#", intercambioFallidos, false, nil); err != nil {
		return err
	}
	argumentos := amqp.Table{
		"x-dead-letter-exchange":    intercambioFallidos,
		"x-dead-letter-routing-key": "cliente.validacion.fallida",
	}
	if _, err := ch.QueueDeclare(colaComandosCliente, true, false, false, false, argumentos); err != nil {
		return err
	}
	for _, tipo := range []string{events.ComandoValidarCliente, events.ComandoValidarKYC, events.ComandoRegistrarCliente, events.ComandoActivarCliente, events.ComandoLoginCliente, events.ComandoPerfilCliente, events.ComandoActualizarCliente, events.ComandoListarClientes, events.ComandoBuscarClienteDPI, events.ComandoEstadoCliente, events.ComandoEstadoKYC} {
		if err := ch.QueueBind(colaComandosCliente, tipo, intercambioComandos, false, nil); err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMQClient) Listo() bool { return r.sesion.listo() }

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
	if sobre.Type == events.ComandoValidarKYC {
		var solicitud events.SolicitudValidacionKYC
		if json.Unmarshal(sobre.Payload, &solicitud) != nil || solicitud.IDOperacion == uuid.Nil || solicitud.IDCliente == uuid.Nil {
			return errors.New("solicitud de validacion KYC invalida")
		}
		return r.repo.RegistrarValidacionKYC(ctx, sobre.MessageID, sobre.CorrelationID, solicitud.IDOperacion, solicitud.IDCliente)
	}

	procesado, resultado, err := r.repo.GetProcessedMessageResult(ctx, sobre.MessageID)
	if err != nil {
		return fmt.Errorf("error verificando idempotencia del comando: %w", err)
	}
	if procesado {
		var cache respuestaRPC
		if err := json.Unmarshal([]byte(resultado), &cache); err != nil {
			return fmt.Errorf("resultado idempotente inválido: %w", err)
		}
		return r.publicarRespuesta(ctx, sobre, cache.Estado, cache.Cuerpo)
	}

	estado, cuerpo := r.ejecutarRPC(ctx, sobre)
	if estado < 500 {
		cache, err := json.Marshal(respuestaRPC{Estado: estado, Cuerpo: cuerpo})
		if err != nil {
			return fmt.Errorf("error serializando resultado idempotente: %w", err)
		}
		if err := r.repo.RecordProcessedMessage(ctx, sobre.MessageID, "customer-service.commands", string(cache)); err != nil {
			return fmt.Errorf("error registrando comando procesado: %w", err)
		}
	}
	return r.publicarRespuesta(ctx, sobre, estado, cuerpo)
}

func (r *RabbitMQClient) publicarRespuesta(ctx context.Context, sobre events.EventEnvelope, estado int, cuerpo json.RawMessage) error {
	contenido, err := json.Marshal(respuestaRPC{Estado: estado, Cuerpo: cuerpo})
	if err != nil {
		return err
	}
	respuesta, err := events.NewEnvelope(strings.TrimSuffix(sobre.Type, ".solicitado")+".respondido", sobre.CorrelationID, &sobre.MessageID, json.RawMessage(contenido))
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(respuesta)
	if err != nil {
		return err
	}
	return r.sesion.publicar(ctx, intercambioRespuestas, respuesta.Type, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", CorrelationId: sobre.CorrelationID.String(), Body: bytes})
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
	case events.ComandoBuscarClienteDPI:
		var req events.SolicitudBuscarClienteDPI
		if err := json.Unmarshal(sobre.Payload, &req); err != nil || strings.TrimSpace(req.Documento) == "" {
			return errorRespuesta(400, errors.New("documentId requerido"))
		}
		cliente, err := r.svc.GetCustomerByDocumentID(ctx, req.Documento)
		if err != nil {
			return errorRespuesta(404, err)
		}
		return respuesta(200, cliente)
	case events.ComandoEstadoCliente:
		var req struct {
			IDCliente uuid.UUID `json:"idCliente"`
			Estado    string    `json:"estado"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		cliente, err := r.svc.UpdateCustomerStatus(ctx, req.IDCliente, req.Estado, sobre.CorrelationID)
		if err != nil {
			return errorRespuesta(400, err)
		}
		return respuesta(200, cliente)
	case events.ComandoEstadoKYC:
		var req struct {
			IDCliente uuid.UUID `json:"idCliente"`
			EstadoKYC string    `json:"estadoKyc"`
		}
		if err := json.Unmarshal(sobre.Payload, &req); err != nil {
			return errorRespuesta(400, err)
		}
		cliente, err := r.svc.UpdateCustomerKYCStatus(ctx, req.IDCliente, req.EstadoKYC, sobre.CorrelationID)
		if err != nil {
			return errorRespuesta(400, err)
		}
		return respuesta(200, cliente)
	default:
		return errorRespuesta(400, errors.New("comando de cliente no soportado"))
	}
}

func (r *RabbitMQClient) Close() { r.sesion.cerrar() }

// StartOutboxWorker envía periódicamente los eventos pendientes a RabbitMQ
func (r *RabbitMQClient) StartOutboxWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
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
	if !r.Listo() {
		return
	}
	msgs, err := r.repo.GetPendingOutbox(ctx, 20)
	if err != nil || len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		exchange := intercambioEventos
		routingKey := msg.EventType

		err := r.sesion.publicar(
			ctx,
			exchange,
			routingKey,
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
