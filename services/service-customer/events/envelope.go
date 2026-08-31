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

func NewEnvelope(eventType string, correlationID uuid.UUID, causationID *uuid.UUID, payload interface{}) (*EventEnvelope, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &EventEnvelope{
		MessageID:     uuid.New(),
		CorrelationID: correlationID,
		CausationID:   causationID,
		Type:          eventType,
		Version:       1,
		OccurredAt:    time.Now().UTC(),
		Producer:      "customer-service",
		Payload:       bytes,
	}, nil
}

type CustomerCreatedPayload struct {
	CustomerID uuid.UUID `json:"customerId"`
	FullName   string    `json:"fullName"`
	Email      string    `json:"email"`
	Username   string    `json:"username"`
	DocumentID string    `json:"documentId"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
}

type ActivationEmailRequestedPayload struct {
	CustomerID     uuid.UUID `json:"customerId"`
	Email          string    `json:"email"`
	FullName       string    `json:"fullName"`
	ActivationLink string    `json:"activationLink"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

type CustomerActivatedPayload struct {
	CustomerID  uuid.UUID `json:"customerId"`
	ActivatedAt time.Time `json:"activatedAt"`
}

type CustomerUpdatedPayload struct {
	CustomerID uuid.UUID `json:"customerId"`
	Address    string    `json:"address"`
	Email      string    `json:"email"`
}
