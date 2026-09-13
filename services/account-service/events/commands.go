package events

import "github.com/google/uuid"

const (
	ComandoCrearCuenta           = "cuenta.creacion.solicitada"
	ComandoSolicitarDebito       = "cuenta.debito.solicitado"
	ComandoSolicitarCredito      = "cuenta.credito.solicitado"
	ComandoSolicitarCompensacion = "cuenta.compensacion.solicitada"
	ComandoConsultarCuenta       = "cuenta.consulta.solicitada"
	ComandoListarMovimientos     = "cuenta.movimientos.solicitados"
	ComandoListarCuentas         = "cuenta.historial.solicitado"
	ComandoValidarTransferencia  = "cuenta.transferencia.validacion.solicitada"
)

type SolicitudCrearCuenta struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
	TipoCuenta  string    `json:"tipoCuenta"`
}

type SolicitudConsultarCuenta struct {
	IDCuenta uuid.UUID `json:"idCuenta"`
}

type SolicitudListarMovimientos struct {
	IDCuenta       uuid.UUID `json:"idCuenta"`
	Limite         int       `json:"limite"`
	Desplazamiento int       `json:"desplazamiento"`
}
type SolicitudListarCuentas struct {
	IDCliente uuid.UUID `json:"idCliente"`
}

type SolicitudMovimiento struct {
	IDCuenta      uuid.UUID `json:"idCuenta"`
	IDOperacion   uuid.UUID `json:"idOperacion"`
	MontoCentavos int64     `json:"montoCentavos"`
}

type SolicitudValidacionTransferencia struct {
	IDOperacion     uuid.UUID `json:"idOperacion"`
	IDCliente       uuid.UUID `json:"idCliente"`
	IDCuentaOrigen  uuid.UUID `json:"idCuentaOrigen"`
	IDCuentaDestino uuid.UUID `json:"idCuentaDestino"`
	MontoCentavos   int64     `json:"montoCentavos"`
}

const ComandoValidarCliente = "cliente.validacion.solicitada"

type SolicitudValidarCliente struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
}
