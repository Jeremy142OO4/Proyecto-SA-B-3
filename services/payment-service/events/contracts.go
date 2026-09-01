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
)

type SolicitudPago struct {
	IDPago         uuid.UUID `json:"idPago"`
	IDCliente      uuid.UUID `json:"idCliente"`
	IDCuentaOrigen uuid.UUID `json:"idCuentaOrigen"`
	Beneficiario   string    `json:"beneficiario"`
	Concepto       string    `json:"concepto"`
	MontoCentavos  int64     `json:"montoCentavos"`
	TipoPago       string    `json:"tipoPago"`
}
type SolicitudMovimiento struct {
	IDCuenta      uuid.UUID `json:"idCuenta"`
	IDOperacion   uuid.UUID `json:"idOperacion"`
	MontoCentavos int64     `json:"montoCentavos"`
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
