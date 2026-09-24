package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventEnvelope struct {
	MessageID     uuid.UUID       `json:"idMensaje"`
	CorrelationID uuid.UUID       `json:"idCorrelacion"`
	CausationID   *uuid.UUID      `json:"idCausa,omitempty"`
	Type          string          `json:"tipo"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"ocurridoEn"`
	Producer      string          `json:"productor"`
	Payload       json.RawMessage `json:"contenido"`
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

const (
	ComandoValidarCliente    = "cliente.validacion.solicitada"
	EventoClienteValidado    = "cliente.validado"
	EventoClienteRechazado   = "cliente.rechazado"
	EventoClienteCreado      = "cliente.creado"
	EventoClienteActivado    = "cliente.activado"
	EventoClienteActualizado = "cliente.actualizado"
	EventoClienteEstadoActualizado = "cliente.estado.actualizado"
	EventoCorreoActivacion   = "notificacion.correo-activacion.solicitado"
	ComandoRegistrarCliente  = "cliente.registro.solicitado"
	ComandoActivarCliente    = "cliente.activacion.solicitada"
	ComandoLoginCliente      = "cliente.login.solicitado"
	ComandoPerfilCliente     = "cliente.perfil.solicitado"
	ComandoActualizarCliente = "cliente.actualizacion.solicitada"
	ComandoListarClientes    = "cliente.listado.solicitado"
	ComandoBuscarClienteDPI  = "cliente.dpi.consulta.solicitada"
	ComandoEstadoCliente     = "cliente.estado.solicitado"
	ComandoValidarKYC        = "cliente.kyc.validacion.solicitada"
	EventoKYCVerificado      = "cliente.kyc.verificado"
	EventoKYCRechazado       = "cliente.kyc.rechazado"
	ComandoEstadoKYC         = "cliente.kyc.estado.solicitado"
	EventoKYCActualizado     = "cliente.kyc.estado.actualizado"
)

type SolicitudBuscarClienteDPI struct {
	Documento string `json:"documentId"`
}

type SolicitudValidacionCliente struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
}

type ResultadoValidacionCliente struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
	Activo      bool      `json:"activo"`
	Motivo      string    `json:"motivo,omitempty"`
}

type SolicitudValidacionKYC struct {
	IDOperacion uuid.UUID `json:"idOperacion"`
	IDCliente   uuid.UUID `json:"idCliente"`
}

type ResultadoValidacionKYC struct {
	IDOperacion uuid.UUID `json:"idOperacion"`
	IDCliente   uuid.UUID `json:"idCliente"`
	EstadoKYC   string    `json:"estadoKyc"`
	Valido      bool      `json:"valido"`
	Motivo      string    `json:"motivo,omitempty"`
}

type CustomerKYCUpdatedPayload struct {
	CustomerID uuid.UUID `json:"idCliente"`
	EstadoKYC  string    `json:"estadoKyc"`
}

type CustomerCreatedPayload struct {
	CustomerID uuid.UUID `json:"idCliente"`
	FullName   string    `json:"nombreCompleto"`
	Email      string    `json:"correo"`
	Username   string    `json:"usuario"`
	DocumentID string    `json:"documento"`
	Role       string    `json:"rol"`
	Status     string    `json:"estado"`
}

type ActivationEmailRequestedPayload struct {
	CustomerID     uuid.UUID `json:"idCliente"`
	Email          string    `json:"correo"`
	FullName       string    `json:"nombreCompleto"`
	ActivationLink string    `json:"enlaceActivacion"`
	ExpiresAt      time.Time `json:"expiraEn"`
}

type CustomerActivatedPayload struct {
	CustomerID  uuid.UUID `json:"idCliente"`
	ActivatedAt time.Time `json:"activadoEn"`
}

type CustomerUpdatedPayload struct {
	CustomerID uuid.UUID `json:"idCliente"`
	Address    string    `json:"direccion"`
	Email      string    `json:"correo"`
}

type CustomerStatusUpdatedPayload struct {
	CustomerID uuid.UUID `json:"idCliente"`
	Status     string    `json:"estado"`
}
