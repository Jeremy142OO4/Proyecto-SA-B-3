package events

import "github.com/google/uuid"

const (
	ComandoProcesarPago          = "pago.procesamiento.solicitado"
	ComandoConsultarPago         = "pago.consulta.solicitada"
	ComandoListarPagos           = "pago.historial.solicitado"
	ComandoSolicitarDebito       = "cuenta.debito.solicitado"
	ComandoSolicitarCompensacion = "cuenta.compensacion.solicitada"
	EventoCuentaDebitada         = "cuenta.debitada"
	EventoDebitoRechazado        = "cuenta.debito.rechazado"
	EventoCuentaCompensada       = "cuenta.compensada"
	EventoPagoCompletado         = "pago.completado"
	EventoPagoRechazado          = "pago.rechazado"
	EventoPagoConsultado         = "pago.consultado"
	EventoHistorialConsultado    = "pago.historial.consultado"
	ComandoValidarKYC            = "cliente.kyc.validacion.solicitada"
	EventoKYCVerificado          = "cliente.kyc.verificado"
	EventoKYCRechazado           = "cliente.kyc.rechazado"
	ComandoValidarCuenta         = "cuenta.transferencia.validacion.solicitada"
	EventoCuentaValidada         = "cuenta.transferencia.validada"
	EventoCuentaRechazada        = "cuenta.transferencia.rechazada"
)

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
type SolicitudMovimiento struct {
	IDCuenta      uuid.UUID `json:"idCuenta"`
	IDOperacion   uuid.UUID `json:"idOperacion"`
	MontoCentavos int64     `json:"montoCentavos"`
}
type SolicitudValidacionKYC struct {
	IDOperacion uuid.UUID `json:"idOperacion"`
	IDCliente   uuid.UUID `json:"idCliente"`
}
type ResultadoValidacionKYC struct {
	IDOperacion uuid.UUID `json:"idOperacion"`
	IDCliente   uuid.UUID `json:"idCliente"`
	EstadoKYC   string    `json:"estadoKyc"`
	Valido      bool      `json:"valido"`
	Motivo      string    `json:"motivo,omitempty"`
}
type SolicitudValidacionCuenta struct {
	IDOperacion     uuid.UUID `json:"idOperacion"`
	IDCliente       uuid.UUID `json:"idCliente"`
	IDCuentaOrigen  uuid.UUID `json:"idCuentaOrigen"`
	IDCuentaDestino uuid.UUID `json:"idCuentaDestino"`
	MontoCentavos   int64     `json:"montoCentavos"`
}
type ResultadoValidacionCuenta struct {
	IDOperacion       uuid.UUID `json:"idOperacion"`
	IDCliente         uuid.UUID `json:"idCliente"`
	Valida            bool      `json:"valida"`
	TipoCuentaOrigen  string    `json:"tipoCuentaOrigen,omitempty"`
	TipoCuentaDestino string    `json:"tipoCuentaDestino,omitempty"`
	Codigo            string    `json:"codigo,omitempty"`
	Motivo            string    `json:"motivo,omitempty"`
}
type ResultadoMovimiento struct {
	IDOperacion uuid.UUID `json:"idOperacion"`
	IDCuenta    uuid.UUID `json:"idCuenta"`
	Codigo      string    `json:"codigo,omitempty"`
	Mensaje     string    `json:"mensaje,omitempty"`
}
type SolicitudConsultarPago struct {
	IDPago uuid.UUID `json:"idPago"`
}
type SolicitudListarPagos struct {
	IDCliente      uuid.UUID `json:"idCliente"`
	Limite         int       `json:"limite"`
	Desplazamiento int       `json:"desplazamiento"`
}
