package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"time"
)

const ConsultarSimulacion = "pago.simulacion.consulta.solicitada"
const ActualizarSimulacion = "pago.simulacion.actualizacion.solicitada"
const RespuestaSimulacion = "pago.simulacion.respondida"

func configuracionSimulacion(ctx context.Context, tx pgx.Tx) (models.ConfiguracionSimulacion, error) {
	var c models.ConfiguracionSimulacion
	err := tx.QueryRow(ctx, `SELECT habilitada,tasa_fallos,latencia_ms,timeout_ms,forzar_timeout,version FROM configuracion_simulacion_pagos WHERE id=TRUE`).Scan(&c.Habilitada, &c.TasaFallosPorcentaje, &c.LatenciaMs, &c.TimeoutMs, &c.ForzarTimeout, &c.Version)
	return c, err
}

// Commands originate from the ADMIN-only Gateway route. Persist response,
// configuration and audit event atomically; broker retries do not bump versions.
func (r *RepositorioPagosPostgres) AdministrarSimulacion(ctx context.Context, m events.SobreMensaje) error {
	ctx = telemetry.WithCorrelation(ctx, m.IDCorrelacion.String())
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado) VALUES($1,'payment-service.administracion',$2,$3,'{}') ON CONFLICT DO NOTHING`, m.IDMensaje, m.Tipo, m.IDCorrelacion)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	status := 200
	var body any
	switch m.Tipo {
	case ConsultarSimulacion:
		body, err = configuracionSimulacion(ctx, tx)
	case ActualizarSimulacion:
		var p struct {
			models.ConfiguracionSimulacion
			IDAdministrador uuid.UUID `json:"idAdministrador"`
		}
		if json.Unmarshal(m.Contenido, &p) != nil || !p.Valida() || p.IDAdministrador == uuid.Nil {
			status = 422
			body = map[string]string{"error": "Tasa de 0 a 100, latencia de 0 a 30000 ms y timeout de 1 a 30000 ms"}
			break
		}
		err = tx.QueryRow(ctx, `UPDATE configuracion_simulacion_pagos SET habilitada=$1,tasa_fallos=$2,latencia_ms=$3,timeout_ms=$4,forzar_timeout=$5,version=version+1,id_administrador=$6,actualizada_en=clock_timestamp() WHERE id=TRUE RETURNING version`, p.Habilitada, p.TasaFallosPorcentaje, p.LatenciaMs, p.TimeoutMs, p.ForzarTimeout, p.IDAdministrador).Scan(&p.Version)
		body = p.ConfiguracionSimulacion
		if err == nil {
			b, _ := json.Marshal(p)
			err = insertarSalida(ctx, tx, "pago.simulacion.configuracion.actualizada", b, m.IDCorrelacion)
		}
	default:
		return errors.New("comando de simulacion no soportado")
	}
	if err != nil {
		return err
	}
	b, err := json.Marshal(map[string]any{"estado": status, "cuerpo": body})
	if err != nil {
		return err
	}
	if err = insertarSalida(ctx, tx, RespuestaSimulacion, b, m.IDCorrelacion); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type planProveedor struct {
	CorrelationID    uuid.UUID
	Resultado        models.ResultadoSimulado
	EsperaMs         int
	Inicio           time.Time
	Version          int64
	TraceContext     string
	RemainingSeconds float64
}

// Uniform deterministic sampling makes retries and worker changes keep the same
// outcome. Its decision and the exact configuration snapshot are also durable.
func decidirSimulacion(c models.ConfiguracionSimulacion, id uuid.UUID, legacy models.ResultadoSimulado) (models.ResultadoSimulado, int) {
	if !c.Habilitada {
		return legacy, 0
	}
	if c.ForzarTimeout || c.LatenciaMs >= c.TimeoutMs {
		return models.ResultadoTimeout, c.TimeoutMs
	}
	digest := sha256.Sum256(id[:])
	sample := float64(binary.BigEndian.Uint64(digest[:8])>>11) / float64(uint64(1)<<53) * 100
	if sample < c.TasaFallosPorcentaje {
		return models.ResultadoFallo, c.LatenciaMs
	}
	return models.ResultadoExito, c.LatenciaMs
}

func (r *RepositorioPagosPostgres) prepararProveedor(ctx context.Context, id uuid.UUID) (*planProveedor, error) {
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	p, err := buscarPagoTxBloqueado(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if p.TipoPago != models.TipoPagoExterno || p.Estado != models.EstadoPagoProcesando {
		return nil, tx.Commit(ctx)
	}
	plan := &planProveedor{CorrelationID: p.IDCorrelacion}
	err = tx.QueryRow(ctx, `SELECT resultado,espera_ms,inicio,version_configuracion FROM simulaciones_pago WHERE id_pago=$1`, id).Scan(&plan.Resultado, &plan.EsperaMs, &plan.Inicio, &plan.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		c, e := configuracionSimulacion(ctx, tx)
		if e != nil {
			return nil, e
		}
		plan.Resultado, plan.EsperaMs = decidirSimulacion(c, id, p.ResultadoSimulado)
		plan.Version = c.Version
		b, _ := json.Marshal(c)
		err = tx.QueryRow(ctx, `INSERT INTO simulaciones_pago(id_pago,version_configuracion,configuracion,resultado,espera_ms) VALUES($1,$2,$3,$4,$5) RETURNING inicio`, id, c.Version, b, plan.Resultado, plan.EsperaMs).Scan(&plan.Inicio)
	}
	if err != nil {
		return nil, err
	}
	if err = tx.QueryRow(ctx, `SELECT p.trace_context,GREATEST(0,EXTRACT(EPOCH FROM(s.inicio+s.espera_ms*INTERVAL '1 millisecond'-clock_timestamp())))::float8 FROM pagos p JOIN simulaciones_pago s USING(id_pago) WHERE p.id_pago=$1`, id).Scan(&plan.TraceContext, &plan.RemainingSeconds); err != nil {
		return nil, err
	}
	return plan, tx.Commit(ctx)
}

func (r *RepositorioPagosPostgres) esperarProveedor(ctx context.Context, id uuid.UUID) (context.Context, *planProveedor, func(), error) {
	plan, err := r.prepararProveedor(ctx, id)
	if err != nil || plan == nil {
		return ctx, plan, func() {}, err
	}
	ctx = restaurarPago(ctx, plan.TraceContext)
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "pago.proveedor.simulado", trace.WithSpanKind(trace.SpanKindClient))
	ctx = telemetry.WithCorrelation(ctx, plan.CorrelationID.String())
	remaining := time.Duration(plan.RemainingSeconds * float64(time.Second))
	if remaining > 0 {
		timer := time.NewTimer(remaining)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			span.End()
			return ctx, nil, func() {}, ctx.Err()
		}
	}
	telemetry.ProveedorSimulado(ctx, id.String(), string(plan.Resultado), plan.Version, plan.EsperaMs)
	return ctx, plan, func() { span.End() }, nil
}

func restaurarPago(ctx context.Context, saved string) context.Context {
	restored := telemetry.Restore(ctx, saved)
	if sc := trace.SpanContextFromContext(restored); sc.IsValid() && sc.TraceID() != trace.SpanContextFromContext(ctx).TraceID() {
		return restored
	}
	return ctx
}

func (r *RepositorioPagosPostgres) contextoPago(ctx context.Context, tx pgx.Tx, p *models.Pago) (context.Context, error) {
	var saved string
	err := tx.QueryRow(ctx, `SELECT trace_context FROM pagos WHERE id_pago=$1`, p.IDPago).Scan(&saved)
	return telemetry.WithCorrelation(restaurarPago(ctx, saved), p.IDCorrelacion.String()), err
}
