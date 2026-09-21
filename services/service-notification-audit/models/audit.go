package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	EventID       uuid.UUID       `db:"event_id" json:"eventId"`
	CorrelationID uuid.UUID       `db:"correlation_id" json:"correlationId"`
	CausationID   *uuid.UUID      `db:"causation_id" json:"causationId,omitempty"`
	EventType     string          `db:"event_type" json:"eventType"`
	Severity      EventSeverity   `db:"severity" json:"severity"`
	Producer      string          `db:"producer" json:"producer"`
	Version       int             `db:"version" json:"version"`
	Payload       json.RawMessage `db:"payload" json:"payload"`
	OccurredAt    time.Time       `db:"occurred_at" json:"occurredAt"`
	RecordedAt    time.Time       `db:"recorded_at" json:"recordedAt"`
}

// EventSeverity classifies the operational importance of a domain event.
// INFO represents normal progress, WARNING represents a recoverable business
// outcome, and ERROR represents an operational failure or unrecoverable flow.
type EventSeverity string

const (
	EventInfo    EventSeverity = "INFO"
	EventWarning EventSeverity = "WARNING"
	EventError   EventSeverity = "ERROR"
)

type NotificationStatus string

const (
	NotificationPending NotificationStatus = "PENDING"
	NotificationSent    NotificationStatus = "SENT"
	NotificationFailed  NotificationStatus = "FAILED"
)

type NotificationLog struct {
	ID               uuid.UUID          `db:"id" json:"id"`
	CorrelationID    uuid.UUID          `db:"correlation_id" json:"correlationId"`
	Recipient        string             `db:"recipient" json:"recipient"`
	NotificationType string             `db:"notification_type" json:"notificationType"`
	Subject          string             `db:"subject" json:"subject"`
	BodySummary      string             `db:"body_summary" json:"bodySummary"`
	Status           NotificationStatus `db:"status" json:"status"`
	ErrorDetail      *string            `db:"error_detail" json:"errorDetail,omitempty"`
	SentAt           time.Time          `db:"sent_at" json:"sentAt"`
}

type NotificationFilter struct {
	Limit          int
	Recipient      string
	Status         NotificationStatus
	CorrelationID  *uuid.UUID
}
