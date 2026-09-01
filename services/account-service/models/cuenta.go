package models

import (
	"time"

	"github.com/google/uuid"
)

type TipoCuenta string

const (
	TipoCuentaMonetaria TipoCuenta = "MONETARIA"
	TipoCuentaAhorro    TipoCuenta = "AHORRO"
)

type EstadoCuenta string

const (
	EstadoCuentaActiva    EstadoCuenta = "ACTIVA"
	EstadoCuentaInactiva  EstadoCuenta = "INACTIVA"
	EstadoCuentaBloqueada EstadoCuenta = "BLOQUEADA"
	EstadoCuentaCerrada   EstadoCuenta = "CERRADA"
)

type Cuenta struct {
	IDCuenta           uuid.UUID    `json:"idCuenta"`
	IDCliente          uuid.UUID    `json:"idCliente"`
	NumeroCuenta       string       `json:"numeroCuenta"`
	TipoCuenta         TipoCuenta   `json:"tipoCuenta"`
	SaldoCentavos      int64        `json:"saldoCentavos"`
	Moneda             string       `json:"moneda"`
	Estado             EstadoCuenta `json:"estado"`
	UltimaActividad    *time.Time   `json:"ultimaActividad,omitempty"`
	FechaCreacion      time.Time    `json:"fechaCreacion"`
	FechaActualizacion time.Time    `json:"fechaActualizacion"`
	Version            int64        `json:"version"`
}
