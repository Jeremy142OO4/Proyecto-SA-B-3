package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"bank-usac/service-notification-audit/events"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const customerProfileCommand = "cliente.perfil.solicitado"

type customerResolver struct {
	publish  *amqp.Channel
	consume  *amqp.Channel
	pending  map[uuid.UUID]chan customerProfileResponse
	mutex    sync.Mutex
}

type customerProfileResponse struct {
	status int
	email  string
	name   string
	err    error
}

func newCustomerResolver(conn *amqp.Connection) (*customerResolver, error) {
	publish, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	consume, err := conn.Channel()
	if err != nil {
		_ = publish.Close()
		return nil, err
	}
	if _, err = consume.QueueDeclare("notification-audit.customer-responses.q", true, false, false, false, nil); err != nil {
		_ = publish.Close()
		_ = consume.Close()
		return nil, err
	}
	if err = consume.QueueBind("notification-audit.customer-responses.q", "cliente.perfil.respondido", "banco.respuestas", false, nil); err != nil {
		_ = publish.Close()
		_ = consume.Close()
		return nil, err
	}
	return &customerResolver{publish: publish, consume: consume, pending: make(map[uuid.UUID]chan customerProfileResponse)}, nil
}

func (r *customerResolver) Start(ctx context.Context) error {
	deliveries, err := r.consume.Consume("notification-audit.customer-responses.q", "notification-audit-customer-resolver", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}
				r.handleResponse(delivery)
			}
		}
	}()
	return nil
}

func (r *customerResolver) handleResponse(delivery amqp.Delivery) {
	defer delivery.Ack(false)
	var envelope events.EventEnvelope
	if json.Unmarshal(delivery.Body, &envelope) != nil || envelope.CorrelationID == uuid.Nil {
		return
	}
	var rpc respuestaRPC
	if json.Unmarshal(envelope.Payload, &rpc) != nil {
		return
	}
	response := customerProfileResponse{status: rpc.Estado}
	if rpc.Estado < 200 || rpc.Estado >= 300 {
		var detail struct{ Error string `json:"error"` }
		_ = json.Unmarshal(rpc.Cuerpo, &detail)
		response.err = fmt.Errorf("customer-service respondió %d: %s", rpc.Estado, detail.Error)
	} else {
		var customer struct {
			Email    string `json:"email"`
			Correo   string `json:"correo"`
			FullName string `json:"fullName"`
		}
		if err := json.Unmarshal(rpc.Cuerpo, &customer); err != nil {
			response.err = err
		} else {
			response.email = strings.TrimSpace(customer.Email)
			if response.email == "" {
				response.email = strings.TrimSpace(customer.Correo)
			}
			response.name = strings.TrimSpace(customer.FullName)
			if response.email == "" {
				response.err = fmt.Errorf("customer-service no devolvió correo")
			}
		}
	}

	r.mutex.Lock()
	responseChannel, ok := r.pending[envelope.CorrelationID]
	if ok {
		delete(r.pending, envelope.CorrelationID)
	}
	r.mutex.Unlock()
	if ok {
		responseChannel <- response
	}
}

func (r *customerResolver) ResolveCustomer(ctx context.Context, customerID uuid.UUID) (string, string, error) {
	if customerID == uuid.Nil {
		return "", "", fmt.Errorf("idCliente inválido")
	}
	correlationID := uuid.New()
	responseChannel := make(chan customerProfileResponse, 1)
	r.mutex.Lock()
	r.pending[correlationID] = responseChannel
	r.mutex.Unlock()
	defer func() {
		r.mutex.Lock()
		delete(r.pending, correlationID)
		r.mutex.Unlock()
	}()

	payload, _ := json.Marshal(map[string]uuid.UUID{"idCliente": customerID})
	envelope := events.EventEnvelope{MessageID: uuid.New(), CorrelationID: correlationID, Type: customerProfileCommand, Version: 1, OccurredAt: time.Now().UTC(), Producer: "notification-audit-service", Payload: payload}
	body, err := json.Marshal(envelope)
	if err != nil {
		return "", "", err
	}
	if err = r.publish.PublishWithContext(ctx, "banco.comandos", customerProfileCommand, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", CorrelationId: correlationID.String(), Body: body}); err != nil {
		return "", "", err
	}
	select {
	case response := <-responseChannel:
		return response.email, response.name, response.err
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

func (r *customerResolver) Close() {
	if r.publish != nil {
		_ = r.publish.Close()
	}
	if r.consume != nil {
		_ = r.consume.Close()
	}
}
