package events

import "github.com/google/uuid"

const (
	ComandoCrearCuenta             = "cuenta.creacion.solicitada"
	ComandoListarCuentas           = "cuenta.historial.solicitado"
	ComandoConsultarCuenta         = "cuenta.consulta.solicitada"
	ComandoMovimientos             = "cuenta.movimientos.solicitados"
	ComandoProcesarPago            = "pago.procesamiento.solicitado"
	ComandoConsultarPago           = "pago.consulta.solicitada"
	ComandoHistorialPagos          = "pago.historial.solicitado"
	ComandoTransferir              = "transferencia.solicitada"
	ComandoConsultarTransferencia  = "transferencia.consulta.solicitada"
	ComandoHistorialTransferencias = "transferencia.historial.solicitado"
	EventoCuentasConsultadas       = "cuenta.historial.consultado"
	EventoCuentaConsultada         = "cuenta.consultada"
	EventoMovimientos              = "cuenta.movimientos.consultados"
	EventoPagoConsultado           = "pago.consultado"
	EventoHistorialPagos           = "pago.historial.consultado"
	EventoTransferenciaConsultada  = "transferencia.consultada"
	EventoHistorialTransferencias  = "transferencia.historial.consultado"
)

type SolicitudCrearCuenta struct {
	IDSolicitud uuid.UUID `json:"idSolicitud"`
	IDCliente   uuid.UUID `json:"idCliente"`
	TipoCuenta  string    `json:"tipoCuenta"`
}
type SolicitudConsultarCuenta struct {
	IDCuenta uuid.UUID `json:"idCuenta"`
}
type SolicitudMovimientos struct {
	IDCuenta       uuid.UUID `json:"idCuenta"`
	Limite         int       `json:"limite"`
	Desplazamiento int       `json:"desplazamiento"`
}
type SolicitudPago struct {
	IDPago         uuid.UUID `json:"idPago"`
	IDCliente      uuid.UUID `json:"idCliente"`
	IDCuentaOrigen uuid.UUID `json:"idCuentaOrigen"`
	Beneficiario   string    `json:"beneficiario"`
	Concepto       string    `json:"concepto"`
	MontoCentavos  int64     `json:"montoCentavos"`
	TipoPago       string    `json:"tipoPago"`
}
type SolicitudConsultarPago struct {
	IDPago uuid.UUID `json:"idPago"`
}
type SolicitudHistorial struct {
	IDCliente      uuid.UUID `json:"idCliente"`
	Limite         int       `json:"limite"`
	Desplazamiento int       `json:"desplazamiento"`
}
type SolicitudTransferencia struct {
	IDTransferencia uuid.UUID `json:"idTransferencia"`
	IDCliente       uuid.UUID `json:"idCliente"`
	IDCuentaOrigen  uuid.UUID `json:"idCuentaOrigen"`
	IDCuentaDestino uuid.UUID `json:"idCuentaDestino"`
	MontoCentavos   int64     `json:"montoCentavos"`
	Descripcion     string    `json:"descripcion,omitempty"`
}
type SolicitudConsultarTransferencia struct {
	IDTransferencia uuid.UUID `json:"idTransferencia"`
}
