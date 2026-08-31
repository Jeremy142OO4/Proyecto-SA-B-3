package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin    Role = "ADMIN"
	RoleTeller   Role = "TELLER"
	RoleCustomer Role = "CUSTOMER"
)

type CustomerStatus string

const (
	StatusPendingActivation CustomerStatus = "PENDING_ACTIVATION"
	StatusActive            CustomerStatus = "ACTIVE"
	StatusBlocked           CustomerStatus = "BLOCKED"
)

type Customer struct {
	CustomerID       uuid.UUID      `db:"customer_id" json:"customerId"`
	FirstName        string         `db:"first_name" json:"firstName"`
	LastName         string         `db:"last_name" json:"lastName"`
	FullName         string         `db:"full_name" json:"fullName"`
	DocumentID       string         `db:"document_id" json:"documentId"`
	DocumentPhotoURL string         `db:"document_photo_url" json:"documentPhotoUrl,omitempty"`
	Email            string         `db:"email" json:"email"`
	BirthDate        time.Time      `db:"birth_date" json:"birthDate"`
	Address          string         `db:"address" json:"address"`
	Username         string         `db:"username" json:"username"`
	PasswordHash     string         `db:"password_hash" json:"-"`
	Role             Role           `db:"role" json:"role"`
	Status           CustomerStatus `db:"status" json:"status"`
	CreatedAt        time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updatedAt"`
}

type ActivationToken struct {
	ID         uuid.UUID  `db:"id"`
	CustomerID uuid.UUID  `db:"customer_id"`
	TokenHash  string     `db:"token_hash"`
	ExpiresAt  time.Time  `db:"expires_at"`
	IsUsed     bool       `db:"is_used"`
	UsedAt     *time.Time `db:"used_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type OutboxMessage struct {
	ID            uuid.UUID  `db:"id"`
	EventType     string     `db:"event_type"`
	Payload       []byte     `db:"payload"`
	CorrelationID uuid.UUID  `db:"correlation_id"`
	CausationID   *uuid.UUID `db:"causation_id"`
	CreatedAt     time.Time  `db:"created_at"`
	PublishedAt   *time.Time `db:"published_at"`
	Attempts      int        `db:"attempts"`
	LastError     *string    `db:"last_error"`
}
