package events

import "github.com/google/uuid"

const (
	ComandoCrearCuenta             = "cuenta.creacion.solicitada"
	ComandoListarCuentas           = "cuenta.historial.solicitado"
	ComandoConsultarCuenta         = "cuenta.consulta.solicitada"
	ComandoMovimientos             = "cuenta.movimientos.solicitados"
	ComandoDepositar               = "cuenta.credito.solicitado"
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
	ComandoRegistrarCliente        = "cliente.registro.solicitado"
	ComandoActivarCliente          = "cliente.activacion.solicitada"
	ComandoLoginCliente            = "cliente.login.solicitado"
	ComandoPerfilCliente           = "cliente.perfil.solicitado"
	ComandoActualizarCliente       = "cliente.actualizacion.solicitada"
	ComandoListarClientes          = "cliente.listado.solicitado"
	ComandoBuscarClienteDPI        = "cliente.dpi.consulta.solicitada"
	ComandoEstadoCliente           = "cliente.estado.solicitado"
	ComandoEstadoKYC               = "cliente.kyc.estado.solicitado"
	ComandoAuditoriaRegistros      = "auditoria.registros.solicitados"
	ComandoAuditoriaTraza          = "auditoria.traza.solicitada"
	ComandoAuditoriaNotificaciones = "auditoria.notificaciones.solicitadas"
)

type SolicitudBuscarClienteDPI struct {
	Documento string `json:"documentId"`
}

type SolicitudCrearCuenta struct {
	IDSolicitud                 uuid.UUID `json:"idSolicitud"`
	IDCliente                   uuid.UUID `json:"idCliente"`
	TipoCuenta                  string    `json:"tipoCuenta"`
	SaldoMinimoCentavos         int64     `json:"saldoMinimoCentavos"`
	ComisionTransaccionCentavos int64     `json:"comisionTransaccionCentavos"`
}
type SolicitudConsultarCuenta struct {
	IDCuenta uuid.UUID `json:"idCuenta"`
}
type SolicitudMovimientos struct {
	IDCuenta       uuid.UUID `json:"idCuenta"`
	Limite         int       `json:"limite"`
	Desplazamiento int       `json:"desplazamiento"`
}
type SolicitudDeposito struct {
	IDCuenta      uuid.UUID `json:"idCuenta"`
	IDOperacion   uuid.UUID `json:"idOperacion"`
	MontoCentavos int64     `json:"montoCentavos"`
}
type SolicitudPago struct {
	IDPago            uuid.UUID `json:"idPago"`
	IDCliente         uuid.UUID `json:"idCliente"`
	IDCuentaOrigen    uuid.UUID `json:"idCuentaOrigen"`
	Beneficiario      string    `json:"beneficiario"`
	Concepto          string    `json:"concepto"`
	MontoCentavos     int64     `json:"montoCentavos"`
	TipoPago          string    `json:"tipoPago"`
	ResultadoSimulado string    `json:"resultadoSimulado,omitempty"`
}
type SolicitudConsultarPago struct {
	IDPago uuid.UUID `json:"idPago"`
}
type SolicitudHistorial struct {
	IDCliente      uuid.UUID  `json:"idCliente"`
	IDCuenta       *uuid.UUID `json:"idCuenta,omitempty"`
	FechaDesde     string     `json:"fechaDesde,omitempty"`
	FechaHasta     string     `json:"fechaHasta,omitempty"`
	Estado         string     `json:"estado,omitempty"`
	Limite         int        `json:"limite"`
	Desplazamiento int        `json:"desplazamiento"`
}
type SolicitudTransferencia struct {
	IDTransferencia          uuid.UUID `json:"idTransferencia"`
	IDCliente                uuid.UUID `json:"idCliente"`
	IDCuentaOrigen           uuid.UUID `json:"idCuentaOrigen"`
	IDCuentaDestino          uuid.UUID `json:"idCuentaDestino"`
	MontoCentavos            int64     `json:"montoCentavos"`
	Descripcion              string    `json:"descripcion,omitempty"`
	ResultadoExternoSimulado string    `json:"resultadoExternoSimulado,omitempty"`
}
type SolicitudConsultarTransferencia struct {
	IDTransferencia uuid.UUID `json:"idTransferencia"`
}
