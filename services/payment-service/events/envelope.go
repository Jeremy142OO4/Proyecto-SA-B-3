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
