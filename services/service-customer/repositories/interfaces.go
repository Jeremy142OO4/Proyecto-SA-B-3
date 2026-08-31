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
	Update(ctx context.Context, customer *models.Customer) error
	ActivateCustomer(ctx context.Context, customerID uuid.UUID, tokenID uuid.UUID, outboxEvent *models.OutboxMessage) error
	FindActivationToken(ctx context.Context, tokenHash string) (*models.ActivationToken, error)

	// Idempotencia y Outbox
	IsMessageProcessed(ctx context.Context, messageID uuid.UUID) (bool, error)
	MarkMessageProcessed(ctx context.Context, tx *sqlx.Tx, messageID uuid.UUID, consumerName, ref string) error
	GetPendingOutbox(ctx context.Context, limit int) ([]*models.OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, id uuid.UUID) error
	IncrementOutboxAttempt(ctx context.Context, id uuid.UUID, errStr string) error
}
