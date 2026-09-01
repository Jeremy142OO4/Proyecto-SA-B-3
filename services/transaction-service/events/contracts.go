package events

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

const (
	ComandoTransferenciaPlan    = "transfer.requested"
	ComandoTransferencia        = "transferencia.solicitada"
	ComandoConsultar            = "transferencia.consulta.solicitada"
	ComandoHistorial            = "transferencia.historial.solicitado"
	ComandoDebito               = "cuenta.debito.solicitado"
	ComandoCredito              = "cuenta.credito.solicitado"
	ComandoCompensacion         = "cuenta.compensacion.solicitada"
	EventoDebitada              = "cuenta.debitada"
	EventoDebitoRechazado       = "cuenta.debito.rechazado"
	EventoAcreditada            = "cuenta.acreditada"
	EventoCreditoRechazado      = "cuenta.credito.rechazado"
	EventoCuentaCompensada      = "cuenta.compensada"
	EventoCompensacionRechazada = "cuenta.compensacion.rechazada"
	EventoProcesando            = "transferencia.procesando"
	EventoCompletada            = "transferencia.completada"
	EventoRechazada             = "transferencia.rechazada"
	EventoCompensando           = "transferencia.compensando"
	EventoCompensada            = "transferencia.compensada"
	EventoCompensacionFallida   = "transferencia.compensacion.fallida"
	EventoConsultada            = "transferencia.consultada"
	EventoHistorial             = "transferencia.historial.consultado"
)

type SobreMensaje struct {
	IDMensaje     uuid.UUID       `json:"idMensaje"`
	IDCorrelacion uuid.UUID       `json:"idCorrelacion"`
	IDCausa       *uuid.UUID      `json:"idCausa,omitempty"`
	Tipo          string          `json:"tipo"`
	Version       int             `json:"version"`
	OcurridoEn    time.Time       `json:"ocurridoEn"`
	Productor     string          `json:"productor"`
	Contenido     json.RawMessage `json:"contenido"`
}
type sobrePlan struct {
	MessageID     uuid.UUID       `json:"messageId"`
	CorrelationID uuid.UUID       `json:"correlationId"`
	CausationID   *uuid.UUID      `json:"causationId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Producer      string          `json:"producer"`
	Payload       json.RawMessage `json:"payload"`
}

func DecodificarSobre(cuerpo []byte) (SobreMensaje, error) {
	var s SobreMensaje
	if e := json.Unmarshal(cuerpo, &s); e != nil {
		return s, e
	}
	if s.IDMensaje != uuid.Nil {
		return s, nil
	}
	var p sobrePlan
	if e := json.Unmarshal(cuerpo, &p); e != nil {
		return s, e
	}
	return SobreMensaje{IDMensaje: p.MessageID, IDCorrelacion: p.CorrelationID, IDCausa: p.CausationID, Tipo: p.Type, Version: p.Version, OcurridoEn: p.OccurredAt, Productor: p.Producer, Contenido: p.Payload}, nil
}

type SolicitudTransferencia struct {
	IDTransferencia uuid.UUID `json:"idTransferencia"`
	OperationID     uuid.UUID `json:"operationId"`
	IDCliente       uuid.UUID `json:"idCliente"`
	CustomerID      uuid.UUID `json:"customerId"`
	IDCuentaOrigen  uuid.UUID `json:"idCuentaOrigen"`
	SourceAccount   uuid.UUID `json:"sourceAccount"`
	IDCuentaDestino uuid.UUID `json:"idCuentaDestino"`
	TargetAccount   uuid.UUID `json:"targetAccount"`
	MontoCentavos   int64     `json:"montoCentavos"`
	AmountCents     int64     `json:"amountCents"`
	Descripcion     string    `json:"descripcion"`
	Description     string    `json:"description"`
}

func (s SolicitudTransferencia) Normalizar() (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) {
	id := s.IDTransferencia
	if id == uuid.Nil {
		id = s.OperationID
	}
	cl := s.IDCliente
	if cl == uuid.Nil {
		cl = s.CustomerID
	}
	o := s.IDCuentaOrigen
	if o == uuid.Nil {
		o = s.SourceAccount
	}
	d := s.IDCuentaDestino
	if d == uuid.Nil {
		d = s.TargetAccount
	}
	m := s.MontoCentavos
	if m == 0 {
		m = s.AmountCents
	}
	desc := s.Descripcion
	if desc == "" {
		desc = s.Description
	}
	return id, cl, o, d, m, desc
}

type SolicitudMovimiento struct {
	IDCuenta      uuid.UUID `json:"idCuenta"`
	IDOperacion   uuid.UUID `json:"idOperacion"`
	MontoCentavos int64     `json:"montoCentavos"`
}
type ResultadoMovimiento struct {
	IDOperacion   uuid.UUID `json:"idOperacion"`
	IDCuenta      uuid.UUID `json:"idCuenta"`
	MontoCentavos int64     `json:"montoCentavos"`
	Codigo        string    `json:"codigo,omitempty"`
	Mensaje       string    `json:"mensaje,omitempty"`
}
type SolicitudConsulta struct {
	IDTransferencia uuid.UUID `json:"idTransferencia"`
}
type SolicitudHistorial struct {
	IDCliente      uuid.UUID `json:"idCliente"`
	Limite         int       `json:"limite"`
	Desplazamiento int       `json:"desplazamiento"`
}
