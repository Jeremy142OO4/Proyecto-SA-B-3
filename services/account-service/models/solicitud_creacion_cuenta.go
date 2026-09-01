package models

import (
	"time"

	"github.com/google/uuid"
)

type EstadoSolicitudCuenta string

const (
	EstadoSolicitudPendiente  EstadoSolicitudCuenta = "PENDIENTE_VALIDACION"
	EstadoSolicitudCompletada EstadoSolicitudCuenta = "COMPLETADA"
	EstadoSolicitudRechazada  EstadoSolicitudCuenta = "RECHAZADA"
)

type SolicitudCreacionCuenta struct {
	IDSolicitud        uuid.UUID             `json:"idSolicitud"`
	IDCliente          uuid.UUID             `json:"idCliente"`
	TipoCuenta         TipoCuenta            `json:"tipoCuenta"`
	Estado             EstadoSolicitudCuenta `json:"estado"`
	IDCorrelacion      uuid.UUID             `json:"idCorrelacion"`
	IDCuenta           *uuid.UUID            `json:"idCuenta,omitempty"`
	MotivoRechazo      string                `json:"motivoRechazo,omitempty"`
	FechaCreacion      time.Time             `json:"fechaCreacion"`
	FechaActualizacion time.Time             `json:"fechaActualizacion"`
}
