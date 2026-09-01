package models

import (
	"time"

	"github.com/google/uuid"
)

type TipoMovimiento string

const (
	TipoMovimientoDebito       TipoMovimiento = "DEBITO"
	TipoMovimientoCredito      TipoMovimiento = "CREDITO"
	TipoMovimientoCompensacion TipoMovimiento = "COMPENSACION"
)

type MovimientoCuenta struct {
	IDMovimiento          uuid.UUID      `json:"idMovimiento"`
	IDCuenta              uuid.UUID      `json:"idCuenta"`
	IDOperacion           uuid.UUID      `json:"idOperacion"`
	IDCorrelacion         uuid.UUID      `json:"idCorrelacion"`
	TipoMovimiento        TipoMovimiento `json:"tipoMovimiento"`
	MontoCentavos         int64          `json:"montoCentavos"`
	SaldoAnteriorCentavos int64          `json:"saldoAnteriorCentavos"`
	SaldoNuevoCentavos    int64          `json:"saldoNuevoCentavos"`
	Descripcion           string         `json:"descripcion,omitempty"`
	FechaCreacion         time.Time      `json:"fechaCreacion"`
}
