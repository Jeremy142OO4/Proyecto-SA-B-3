package repositories

import (
	"context"

	"bank-usac/service-customer/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CustomerRepository interface {
	CreateWithOutbox(ctx context.Context, customer *models.Customer, token *models.ActivationToken, outboxEvents []*models.OutboxMessage) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Customer, error)
	GetByUsername(ctx context.Context, username string) (*models.Customer, error)
	GetByEmail(ctx context.Context, email string) (*models.Customer, error)
	GetByDocumentID(ctx context.Context, docID string) (*models.Customer, error)
	List(ctx context.Context, limit, offset int) ([]*models.Customer, error)
	UpdateStatusWithOutbox(ctx context.Context, id uuid.UUID, status models.CustomerStatus, outboxEvent *models.OutboxMessage) (*models.Customer, error)
	UpdateKYCStatusWithOutbox(ctx context.Context, id uuid.UUID, status models.KYCStatus, outboxEvent *models.OutboxMessage) (*models.Customer, error)
	UpdateWithOutbox(ctx context.Context, customer *models.Customer, outboxEvent *models.OutboxMessage) error
	ActivateCustomer(ctx context.Context, customerID uuid.UUID, tokenID uuid.UUID, outboxEvent *models.OutboxMessage) error
	FindActivationToken(ctx context.Context, tokenHash string) (*models.ActivationToken, error)

	// Idempotencia y Outbox
	IsMessageProcessed(ctx context.Context, messageID uuid.UUID) (bool, error)
	MarkMessageProcessed(ctx context.Context, tx *sqlx.Tx, messageID uuid.UUID, consumerName, ref string) error
	GetProcessedMessageResult(ctx context.Context, messageID uuid.UUID) (bool, string, error)
	RecordProcessedMessage(ctx context.Context, messageID uuid.UUID, consumerName, ref string) error
	GetPendingOutbox(ctx context.Context, limit int) ([]*models.OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, id uuid.UUID) error
	IncrementOutboxAttempt(ctx context.Context, id uuid.UUID, errStr string) error
	RegistrarValidacionCliente(ctx context.Context, mensajeID, correlacionID uuid.UUID, solicitudID, clienteID uuid.UUID) error
	RegistrarValidacionKYC(ctx context.Context, mensajeID, correlacionID, operacionID, clienteID uuid.UUID) error
}
