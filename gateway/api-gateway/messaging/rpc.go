package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/google/uuid"
)

type RespuestaRPC struct {
	Estado int             `json:"estado"`
	Cuerpo json.RawMessage `json:"cuerpo"`
}

type Solicitante struct {
	publicador *Publicador
	respuestas *responses.Gestor
	tiempo     time.Duration
}

func NuevoSolicitante(p *Publicador, r *responses.Gestor, tiempo time.Duration) *Solicitante {
	return &Solicitante{publicador: p, respuestas: r, tiempo: tiempo}
}

func (s *Solicitante) Solicitar(ctx context.Context, tipo string, correlacion uuid.UUID, contenido any) (RespuestaRPC, error) {
	espera, liberar := s.respuestas.Registrar(correlacion)
	defer liberar()
	mensaje, err := events.Nuevo(tipo, correlacion, contenido)
	if err != nil {
		return RespuestaRPC{}, err
	}
	if err = s.publicador.Publicar(ctx, mensaje); err != nil {
		return RespuestaRPC{}, err
	}
	select {
	case respuesta := <-espera:
		var resultado RespuestaRPC
		if err = json.Unmarshal(respuesta.Contenido, &resultado); err != nil {
			return RespuestaRPC{}, fmt.Errorf("respuesta RPC invalida: %w", err)
		}
		return resultado, nil
	case <-time.After(s.tiempo):
		return RespuestaRPC{}, fmt.Errorf("tiempo de espera agotado para %s", tipo)
	case <-ctx.Done():
		return RespuestaRPC{}, ctx.Err()
	}
}
