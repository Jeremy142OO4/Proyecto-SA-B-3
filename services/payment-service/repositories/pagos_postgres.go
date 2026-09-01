package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

var ErrPagoNoEncontrado = errors.New("pago no encontrado")

type RepositorioPagosPostgres struct{ conexion *pgxpool.Pool }

func NuevoRepositorioPagosPostgres(c *pgxpool.Pool) *RepositorioPagosPostgres {
	return &RepositorioPagosPostgres{conexion: c}
}

func (r *RepositorioPagosPostgres) Iniciar(ctx context.Context, m events.SobreMensaje, s events.SolicitudPago) (*models.Pago, bool, error) {
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	res, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado) VALUES($1,'payment-service.comandos',$2,$3,'{}') ON CONFLICT DO NOTHING`, m.IDMensaje, m.Tipo, m.IDCorrelacion)
	if err != nil {
		return nil, false, err
	}
	if res.RowsAffected() == 0 {
		p, e := buscarPagoTx(ctx, tx, s.IDPago)
		if e != nil {
			return nil, false, e
		}
		return p, false, tx.Commit(ctx)
	}
	ahora := time.Now().UTC()
	p := &models.Pago{IDPago: s.IDPago, IDCliente: s.IDCliente, IDCuentaOrigen: s.IDCuentaOrigen, Beneficiario: s.Beneficiario, Concepto: s.Concepto, MontoCentavos: s.MontoCentavos, Moneda: "GTQ", TipoPago: models.TipoPago(s.TipoPago), Estado: models.EstadoPagoProcesando, IDCorrelacion: m.IDCorrelacion, FechaCreacion: ahora, FechaActualizacion: ahora}
	_, err = tx.Exec(ctx, `INSERT INTO pagos(id_pago,id_cliente,id_cuenta_origen,beneficiario,concepto,monto_centavos,moneda,tipo_pago,estado,id_correlacion,fecha_creacion,fecha_actualizacion)VALUES($1,$2,$3,$4,$5,$6,'GTQ',$7,'PROCESANDO',$8,$9,$9)`, p.IDPago, p.IDCliente, p.IDCuentaOrigen, p.Beneficiario, p.Concepto, p.MontoCentavos, p.TipoPago, p.IDCorrelacion, ahora)
	if err != nil {
		return nil, false, fmt.Errorf("guardar pago: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO intentos_pago(id_pago,numero_intento,estado,fecha_inicio) VALUES($1,1,'PROCESANDO',$2)`, p.IDPago, ahora)
	if err != nil {
		return nil, false, fmt.Errorf("guardar intento de pago: %w", err)
	}
	contenido, _ := json.Marshal(events.SolicitudMovimiento{IDCuenta: p.IDCuentaOrigen, IDOperacion: p.IDPago, MontoCentavos: p.MontoCentavos})
	if err = insertarSalida(ctx, tx, events.ComandoSolicitarDebito, contenido, m.IDCorrelacion); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

func (r *RepositorioPagosPostgres) ProcesarResultadoCuenta(ctx context.Context, m events.SobreMensaje, respuesta events.ResultadoMovimiento) (bool, error) {
	tx, err := r.conexion.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	res, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado)VALUES($1,'payment-service.eventos-cuenta',$2,$3,'{}')ON CONFLICT DO NOTHING`, m.IDMensaje, m.Tipo, m.IDCorrelacion)
	if err != nil {
		return false, err
	}
	if res.RowsAffected() == 0 {
		return false, tx.Commit(ctx)
	}
	pago, err := buscarPagoTxBloqueado(ctx, tx, respuesta.IDOperacion)
	if err != nil {
		return false, err
	}
	switch m.Tipo {
	case events.EventoDebitoRechazado:
		pago.Estado = models.EstadoPagoRechazado
		pago.MotivoRechazo = respuesta.Mensaje
		if pago.MotivoRechazo == "" {
			pago.MotivoRechazo = respuesta.Codigo
		}
		if err = actualizarPago(ctx, tx, pago); err != nil {
			return false, err
		}
		if err = finalizarIntento(ctx, tx, pago.IDPago, "RECHAZADO", respuesta.Codigo, pago.MotivoRechazo); err != nil {
			return false, err
		}
		contenido, _ := json.Marshal(pago)
		err = insertarSalida(ctx, tx, events.EventoPagoRechazado, contenido, pago.IDCorrelacion)
	case events.EventoCuentaDebitada:
		if pago.TipoPago == models.TipoPagoExterno && strings.Contains(strings.ToUpper(pago.Beneficiario), "FALLO") {
			pago.Estado = models.EstadoPagoCompensando
			pago.MotivoRechazo = "fallo simulado del proveedor externo"
			if err = actualizarPago(ctx, tx, pago); err != nil {
				return false, err
			}
			if err = finalizarIntento(ctx, tx, pago.IDPago, "FALLIDO", "PROVEEDOR_EXTERNO", pago.MotivoRechazo); err != nil {
				return false, err
			}
			contenido, _ := json.Marshal(events.SolicitudMovimiento{IDCuenta: pago.IDCuentaOrigen, IDOperacion: pago.IDPago, MontoCentavos: pago.MontoCentavos})
			err = insertarSalida(ctx, tx, events.ComandoSolicitarCompensacion, contenido, pago.IDCorrelacion)
		} else {
			pago.Estado = models.EstadoPagoCompletado
			if pago.TipoPago == models.TipoPagoExterno {
				pago.ReferenciaExterna = "EXT-" + pago.IDPago.String()[:8]
			}
			if err = actualizarPago(ctx, tx, pago); err != nil {
				return false, err
			}
			if err = finalizarIntento(ctx, tx, pago.IDPago, "COMPLETADO", "OK", ""); err != nil {
				return false, err
			}
			contenido, _ := json.Marshal(pago)
			err = insertarSalida(ctx, tx, events.EventoPagoCompletado, contenido, pago.IDCorrelacion)
		}
	case events.EventoCuentaCompensada:
		pago.Estado = models.EstadoPagoRechazado
		if err = actualizarPago(ctx, tx, pago); err != nil {
			return false, err
		}
		contenido, _ := json.Marshal(pago)
		err = insertarSalida(ctx, tx, events.EventoPagoRechazado, contenido, pago.IDCorrelacion)
	default:
		return false, fmt.Errorf("evento de cuenta no soportado: %s", m.Tipo)
	}
	if err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func finalizarIntento(ctx context.Context, tx pgx.Tx, idPago uuid.UUID, estado, codigo, detalle string) error {
	_, err := tx.Exec(ctx, `UPDATE intentos_pago
		SET estado=$1,codigo_respuesta=NULLIF($2,''),detalle_error=NULLIF($3,''),fecha_finalizacion=NOW()
		WHERE id_pago=$4 AND numero_intento=1 AND fecha_finalizacion IS NULL`, estado, codigo, detalle, idPago)
	return err
}

const columnasPago = `id_pago,id_cliente,id_cuenta_origen,beneficiario,concepto,monto_centavos,moneda,tipo_pago,estado,COALESCE(referencia_externa,''),id_correlacion,COALESCE(motivo_rechazo,''),fecha_creacion,fecha_actualizacion`

type scanner interface{ Scan(...any) error }

func escanearPago(s scanner) (*models.Pago, error) {
	var p models.Pago
	err := s.Scan(&p.IDPago, &p.IDCliente, &p.IDCuentaOrigen, &p.Beneficiario, &p.Concepto, &p.MontoCentavos, &p.Moneda, &p.TipoPago, &p.Estado, &p.ReferenciaExterna, &p.IDCorrelacion, &p.MotivoRechazo, &p.FechaCreacion, &p.FechaActualizacion)
	return &p, err
}
func buscarPagoTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.Pago, error) {
	p, err := escanearPago(tx.QueryRow(ctx, `SELECT `+columnasPago+` FROM pagos WHERE id_pago=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPagoNoEncontrado
	}
	return p, err
}
func buscarPagoTxBloqueado(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.Pago, error) {
	p, err := escanearPago(tx.QueryRow(ctx, `SELECT `+columnasPago+` FROM pagos WHERE id_pago=$1 FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPagoNoEncontrado
	}
	return p, err
}
func (r *RepositorioPagosPostgres) BuscarPorID(ctx context.Context, id uuid.UUID) (*models.Pago, error) {
	p, err := escanearPago(r.conexion.QueryRow(ctx, `SELECT `+columnasPago+` FROM pagos WHERE id_pago=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPagoNoEncontrado
	}
	return p, err
}
func (r *RepositorioPagosPostgres) ListarPorCliente(ctx context.Context, id uuid.UUID, limite, offset int) ([]models.Pago, error) {
	filas, err := r.conexion.Query(ctx, `SELECT `+columnasPago+` FROM pagos WHERE id_cliente=$1 ORDER BY fecha_creacion DESC LIMIT $2 OFFSET $3`, id, limite, offset)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	lista := []models.Pago{}
	for filas.Next() {
		p, e := escanearPago(filas)
		if e != nil {
			return nil, e
		}
		lista = append(lista, *p)
	}
	return lista, filas.Err()
}
func actualizarPago(ctx context.Context, tx pgx.Tx, p *models.Pago) error {
	_, err := tx.Exec(ctx, `UPDATE pagos SET estado=$1,referencia_externa=NULLIF($2,''),motivo_rechazo=NULLIF($3,''),fecha_actualizacion=NOW() WHERE id_pago=$4`, p.Estado, p.ReferenciaExterna, p.MotivoRechazo, p.IDPago)
	return err
}
func insertarSalida(ctx context.Context, tx pgx.Tx, tipo string, contenido []byte, correlacion uuid.UUID) error {
	_, err := tx.Exec(ctx, `INSERT INTO mensajes_salida(id_mensaje,tipo_evento,version_evento,contenido,id_correlacion)VALUES($1,$2,1,$3,$4)`, uuid.New(), tipo, contenido, correlacion)
	return err
}
func (r *RepositorioPagosPostgres) ListarSalida(ctx context.Context, limite int) ([]MensajeSalida, error) {
	filas, err := r.conexion.Query(ctx, `SELECT id_mensaje,tipo_evento,version_evento,contenido,id_correlacion,fecha_creacion FROM mensajes_salida WHERE estado='PENDIENTE' ORDER BY fecha_creacion LIMIT $1`, limite)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	lista := []MensajeSalida{}
	for filas.Next() {
		var m MensajeSalida
		if err = filas.Scan(&m.IDMensaje, &m.TipoEvento, &m.VersionEvento, &m.Contenido, &m.IDCorrelacion, &m.FechaCreacion); err != nil {
			return nil, err
		}
		lista = append(lista, m)
	}
	return lista, filas.Err()
}
func (r *RepositorioPagosPostgres) MarcarPublicado(ctx context.Context, id uuid.UUID) error {
	_, err := r.conexion.Exec(ctx, `UPDATE mensajes_salida SET estado='PUBLICADO',fecha_publicacion=NOW()WHERE id_mensaje=$1`, id)
	return err
}
func (r *RepositorioPagosPostgres) RegistrarFallo(ctx context.Context, id uuid.UUID) error {
	_, err := r.conexion.Exec(ctx, `UPDATE mensajes_salida SET cantidad_intentos=cantidad_intentos+1 WHERE id_mensaje=$1`, id)
	return err
}

func (r *RepositorioPagosPostgres) RegistrarRespuesta(ctx context.Context, mensaje events.SobreMensaje, tipo string, contenido any) (bool, error) {
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	resultado, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado)
		VALUES($1,'payment-service.consultas',$2,$3,'{}') ON CONFLICT DO NOTHING`, mensaje.IDMensaje, mensaje.Tipo, mensaje.IDCorrelacion)
	if err != nil {
		return false, err
	}
	if resultado.RowsAffected() == 0 {
		return false, tx.Commit(ctx)
	}
	jsonContenido, err := json.Marshal(contenido)
	if err != nil {
		return false, err
	}
	if err = insertarSalida(ctx, tx, tipo, jsonContenido, mensaje.IDCorrelacion); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
