package models

import (
	"github.com/google/uuid"
	"time"
)

type TipoPago string

const (
	TipoPagoInterno TipoPago = "INTERNO"
	TipoPagoExterno TipoPago = "EXTERNO"
)

type EstadoPago string

const (
	EstadoPagoPendiente   EstadoPago = "PENDIENTE"
	EstadoPagoProcesando  EstadoPago = "PROCESANDO"
	EstadoPagoCompensando EstadoPago = "COMPENSANDO"
	EstadoPagoCompletado  EstadoPago = "COMPLETADO"
	EstadoPagoRechazado   EstadoPago = "RECHAZADO"
)

type Pago struct {
	IDPago             uuid.UUID  `json:"idPago"`
	IDCliente          uuid.UUID  `json:"idCliente"`
	IDCuentaOrigen     uuid.UUID  `json:"idCuentaOrigen"`
	Beneficiario       string     `json:"beneficiario"`
	Concepto           string     `json:"concepto"`
	MontoCentavos      int64      `json:"montoCentavos"`
	Moneda             string     `json:"moneda"`
	TipoPago           TipoPago   `json:"tipoPago"`
	Estado             EstadoPago `json:"estado"`
	ReferenciaExterna  string     `json:"referenciaExterna,omitempty"`
	IDCorrelacion      uuid.UUID  `json:"idCorrelacion"`
	MotivoRechazo      string     `json:"motivoRechazo,omitempty"`
	FechaCreacion      time.Time  `json:"fechaCreacion"`
	FechaActualizacion time.Time  `json:"fechaActualizacion"`
}
