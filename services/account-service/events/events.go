package events

import "github.com/google/uuid"

const (
	EventoCuentaCreada            = "cuenta.creada"
	EventoCreacionCuentaRechazada = "cuenta.creacion.rechazada"
	EventoCuentaDebitada          = "cuenta.debitada"
	EventoDebitoRechazado         = "cuenta.debito.rechazado"
	EventoCuentaAcreditada        = "cuenta.acreditada"
	EventoCreditoRechazado        = "cuenta.credito.rechazado"
	EventoCuentaCompensada        = "cuenta.compensada"
	EventoCompensacionRechazada   = "cuenta.compensacion.rechazada"
	EventoClienteValidado         = "cliente.validado"
	EventoClienteRechazado        = "cliente.rechazado"
	EventoCuentaDesactivada       = "cuenta.desactivada"
	EventoCuentaConsultada        = "cuenta.consultada"
	EventoMovimientosConsultados  = "cuenta.movimientos.consultados"
)

type ResultadoValidacionCliente struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
	Activo      bool      `json:"activo"`
	Motivo      string    `json:"motivo,omitempty"`
}
