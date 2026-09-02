package models

import (
	"github.com/google/uuid"
	"time"
)

type Estado string

const (
	Pendiente           Estado = "PENDIENTE"
	Procesando          Estado = "PROCESANDO"
	Completada          Estado = "COMPLETADA"
	Rechazada           Estado = "RECHAZADA"
	Compensando         Estado = "COMPENSANDO"
	Compensada          Estado = "COMPENSADA"
	CompensacionFallida Estado = "COMPENSACION_FALLIDA"
)

type Transferencia struct {
	IDTransferencia    uuid.UUID `json:"idTransferencia"`
	IDCliente          uuid.UUID `json:"idCliente"`
	IDCuentaOrigen     uuid.UUID `json:"idCuentaOrigen"`
	IDCuentaDestino    uuid.UUID `json:"idCuentaDestino"`
	IDCorrelacion      uuid.UUID `json:"idCorrelacion"`
	MontoCentavos      int64     `json:"montoCentavos"`
	Moneda             string    `json:"moneda"`
	Descripcion        string    `json:"descripcion,omitempty"`
	Estado             Estado    `json:"estado"`
	CodigoError        string    `json:"codigoError,omitempty"`
	FechaCreacion      time.Time `json:"fechaCreacion"`
	FechaActualizacion time.Time `json:"fechaActualizacion"`
}
