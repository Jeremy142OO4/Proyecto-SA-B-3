package events

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
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
type sobreIngles struct {
	MessageID     uuid.UUID       `json:"messageId"`
	CorrelationID uuid.UUID       `json:"correlationId"`
	CausationID   *uuid.UUID      `json:"causationId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Producer      string          `json:"producer"`
	Payload       json.RawMessage `json:"payload"`
}

func Decodificar(c []byte) (SobreMensaje, error) {
	var s SobreMensaje
	if e := json.Unmarshal(c, &s); e != nil {
		return s, e
	}
	if s.IDMensaje != uuid.Nil {
		return s, nil
	}
	var i sobreIngles
	if e := json.Unmarshal(c, &i); e != nil {
		return s, e
	}
	return SobreMensaje{IDMensaje: i.MessageID, IDCorrelacion: i.CorrelationID, IDCausa: i.CausationID, Tipo: i.Type, Version: i.Version, OcurridoEn: i.OccurredAt, Productor: i.Producer, Contenido: i.Payload}, nil
}
func Nuevo(tipo string, id uuid.UUID, p any) (SobreMensaje, error) {
	b, e := json.Marshal(p)
	if e != nil {
		return SobreMensaje{}, e
	}
	return SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: id, Tipo: tipo, Version: 1, OcurridoEn: time.Now().UTC(), Productor: "api-gateway", Contenido: b}, nil
}
