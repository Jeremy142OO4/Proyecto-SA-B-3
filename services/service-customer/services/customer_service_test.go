package services

import (
	"context"
	"encoding/json"
	"testing"

	"bank-usac/service-customer/config"
	"bank-usac/service-customer/events"
	"bank-usac/service-customer/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type customerRepositoryFake struct {
	customer      *models.Customer
	createdOutbox []*models.OutboxMessage
	updatedOutbox *models.OutboxMessage
}

func (r *customerRepositoryFake) CreateWithOutbox(_ context.Context, customer *models.Customer, _ *models.ActivationToken, outbox []*models.OutboxMessage) error {
	r.customer = customer
	r.createdOutbox = outbox
	return nil
}
func (r *customerRepositoryFake) GetByID(_ context.Context, id uuid.UUID) (*models.Customer, error) {
	if r.customer != nil && r.customer.CustomerID == id {
		return r.customer, nil
	}
	return nil, nil
}
func (r *customerRepositoryFake) GetByUsername(context.Context, string) (*models.Customer, error) { return nil, nil }
func (r *customerRepositoryFake) GetByEmail(context.Context, string) (*models.Customer, error)    { return nil, nil }
func (r *customerRepositoryFake) GetByDocumentID(context.Context, string) (*models.Customer, error) {
	return nil, nil
}
func (r *customerRepositoryFake) List(context.Context, int, int) ([]*models.Customer, error) { return nil, nil }
func (r *customerRepositoryFake) UpdateStatusWithOutbox(_ context.Context, id uuid.UUID, status models.CustomerStatus, outbox *models.OutboxMessage) (*models.Customer, error) {
	if r.customer == nil || r.customer.CustomerID != id {
		return nil, nil
	}
	r.customer.Status = status
	r.updatedOutbox = outbox
	return r.customer, nil
}
func (r *customerRepositoryFake) UpdateKYCStatusWithOutbox(_ context.Context, id uuid.UUID, status models.KYCStatus, outbox *models.OutboxMessage) (*models.Customer, error) {
	if r.customer == nil || r.customer.CustomerID != id {
		return nil, nil
	}
	r.customer.KYCStatus = status
	r.updatedOutbox = outbox
	return r.customer, nil
}
func (r *customerRepositoryFake) UpdateWithOutbox(context.Context, *models.Customer, *models.OutboxMessage) error { return nil }
func (r *customerRepositoryFake) ActivateCustomer(context.Context, uuid.UUID, uuid.UUID, *models.OutboxMessage) error {
	return nil
}
func (r *customerRepositoryFake) FindActivationToken(context.Context, string) (*models.ActivationToken, error) {
	return nil, nil
}
func (r *customerRepositoryFake) IsMessageProcessed(context.Context, uuid.UUID) (bool, error) { return false, nil }
func (r *customerRepositoryFake) MarkMessageProcessed(context.Context, *sqlx.Tx, uuid.UUID, string, string) error {
	return nil
}
func (r *customerRepositoryFake) GetProcessedMessageResult(context.Context, uuid.UUID) (bool, string, error) {
	return false, "", nil
}
func (r *customerRepositoryFake) RecordProcessedMessage(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (r *customerRepositoryFake) GetPendingOutbox(context.Context, int) ([]*models.OutboxMessage, error) {
	return nil, nil
}
func (r *customerRepositoryFake) MarkOutboxPublished(context.Context, uuid.UUID) error { return nil }
func (r *customerRepositoryFake) IncrementOutboxAttempt(context.Context, uuid.UUID, string) error {
	return nil
}
func (r *customerRepositoryFake) RegistrarValidacionCliente(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (r *customerRepositoryFake) RegistrarValidacionKYC(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

func TestCustomerUpdateStatusPreservesCorrelationAndState(t *testing.T) {
	idCliente, correlationID := uuid.New(), uuid.New()
	repo := &customerRepositoryFake{customer: &models.Customer{CustomerID: idCliente, Status: models.StatusPendingActivation}}
	service := NewCustomerService(repo, &config.Config{ActivationLinkBase: "http://test"})

	customer, err := service.UpdateCustomerStatus(context.Background(), idCliente, string(models.StatusActive), correlationID)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if customer.Status != models.StatusActive || repo.updatedOutbox == nil || repo.updatedOutbox.CorrelationID != correlationID {
		t.Fatalf("estado o correlacion inesperados: customer=%+v outbox=%+v", customer, repo.updatedOutbox)
	}
	var envelope events.EventEnvelope
	if err := json.Unmarshal(repo.updatedOutbox.Payload, &envelope); err != nil {
		t.Fatalf("evento de estado invalido: %v", err)
	}
	if envelope.CorrelationID != correlationID || envelope.Type != events.EventoClienteEstadoActualizado {
		t.Fatalf("sobre transversal inesperado: %+v", envelope)
	}
}

func TestCustomerRegisterCreatesPendingStateAndCorrelatedOutbox(t *testing.T) {
	correlationID := uuid.New()
	repo := &customerRepositoryFake{}
	service := NewCustomerService(repo, &config.Config{ActivationLinkBase: "http://test/activate"})
	customer, err := service.RegisterCustomer(context.Background(), RegisterRequest{
		FirstName: "Ana", LastName: "Prueba", DocumentID: "D-001", Email: "ANA@example.com",
		BirthDate: "1990-01-01", Address: "Zona 1", Password: "secreto123",
	}, correlationID)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if customer.Status != models.StatusPendingActivation || customer.KYCStatus != models.KYCPending {
		t.Fatalf("estado inicial inesperado: %+v", customer)
	}
	if len(repo.createdOutbox) != 2 {
		t.Fatalf("se esperaban dos eventos en Outbox, se obtuvieron %d", len(repo.createdOutbox))
	}
	for _, outbox := range repo.createdOutbox {
		if outbox.CorrelationID != correlationID {
			t.Fatalf("evento sin CorrelationId original: %+v", outbox)
		}
	}
}

func TestCustomerRegisterRechazaDatosInvalidos(t *testing.T) {
	service := NewCustomerService(&customerRepositoryFake{}, &config.Config{})
	_, err := service.RegisterCustomer(context.Background(), RegisterRequest{FirstName: "Ana"}, uuid.New())
	if err == nil {
		t.Fatal("se esperaba error para registro incompleto")
	}
}
