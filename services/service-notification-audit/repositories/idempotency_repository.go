package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type idempotencyRepo struct {
	db *sqlx.DB
}

func NewIdempotencyRepository(db *sqlx.DB) IdempotencyRepository {
	return &idempotencyRepo{db: db}
}

func (r *idempotencyRepo) IsMessageProcessed(ctx context.Context, messageID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(1) FROM processed_messages WHERE message_id = $1", messageID)
	return count > 0, err
}

func (r *idempotencyRepo) MarkMessageProcessed(ctx context.Context, messageID uuid.UUID, consumerName, ref string) error {
	query := "INSERT INTO processed_messages (message_id, consumer_name, result_reference) VALUES ($1, $2, $3)"
	_, err := r.db.ExecContext(ctx, query, messageID, consumerName, ref)
	return err
}
