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

const ComandoValidarCliente = "cliente.validacion.solicitada"

type SolicitudValidarCliente struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
}
