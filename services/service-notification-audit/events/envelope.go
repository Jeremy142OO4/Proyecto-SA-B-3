package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventEnvelope struct {
	MessageID     uuid.UUID       `json:"messageId"`
	CorrelationID uuid.UUID       `json:"correlationId"`
	CausationID   *uuid.UUID      `json:"causationId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Producer      string          `json:"producer"`
	Payload       json.RawMessage `json:"payload"`
}

type ActivationEmailPayload struct {
	CustomerID     uuid.UUID `json:"customerId"`
	Email          string    `json:"email"`
	FullName       string    `json:"fullName"`
	ActivationLink string    `json:"activationLink"`
	ExpiresAt      time.Time `json:"expiresAt"`
}
