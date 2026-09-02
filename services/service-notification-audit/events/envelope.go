package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventEnvelope struct {
	MessageID     uuid.UUID       `json:"idMensaje"`
	CorrelationID uuid.UUID       `json:"idCorrelacion"`
	CausationID   *uuid.UUID      `json:"idCausa"`
	Type          string          `json:"tipo"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"ocurridoEn"`
	Producer      string          `json:"productor"`
	Payload       json.RawMessage `json:"contenido"`
}

type ActivationEmailPayload struct {
	CustomerID     uuid.UUID `json:"idCliente"`
	Email          string    `json:"correo"`
	FullName       string    `json:"nombreCompleto"`
	ActivationLink string    `json:"enlaceActivacion"`
	ExpiresAt      time.Time `json:"expiraEn"`
}

const (
	ComandoRegistros      = "auditoria.registros.solicitados"
	ComandoTraza          = "auditoria.traza.solicitada"
	ComandoNotificaciones = "auditoria.notificaciones.solicitadas"
)
