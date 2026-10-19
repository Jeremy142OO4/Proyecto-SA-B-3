package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const EventoAccountAnomalyDetected = "AccountAnomalyDetected"

type AnomaliaCuenta struct {
	IDAnomalia           uuid.UUID         `json:"idAnomalia"`
	IDCuenta             uuid.UUID         `json:"idCuenta"`
	IDCliente            uuid.UUID         `json:"idCliente"`
	IDMovimiento         uuid.UUID         `json:"idMovimiento"`
	IDOperacion          uuid.UUID         `json:"idOperacion"`
	TipoCuenta           models.TipoCuenta `json:"tipoCuenta"`
	Operaciones          int               `json:"operaciones"`
	MaxOperaciones       int               `json:"maxOperaciones"`
	VentanaMinutos       int               `json:"ventanaMinutos"`
	VersionConfiguracion int64             `json:"versionConfiguracion"`
	DetectadaEn          time.Time         `json:"detectadaEn"`
	Nivel                string            `json:"nivel"`
}

func (r *RepositorioCuentasPostgres) configuracionMonitoreo(ctx context.Context, tx pgx.Tx) (models.ConfiguracionMonitoreo, error) {
	c := r.monitoreoBase
	err := tx.QueryRow(ctx, `SELECT max_operaciones,ventana_minutos,version FROM configuracion_monitoreo_cuentas WHERE id=TRUE`).Scan(&c.MaxOperaciones, &c.VentanaMinutos, &c.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Origen = "CONFIGMAP"
		return c, nil
	}
	c.Origen = "ADMINISTRACION"
	return c, err
}

// The caller holds the account row lock. The movement, anomaly and outbox
// event either commit together or all roll back. Compensation is excluded.
func (r *RepositorioCuentasPostgres) detectarAnomalia(ctx context.Context, tx pgx.Tx, cuenta models.Cuenta, m models.MovimientoCuenta) (*AnomaliaCuenta, error) {
	if m.TipoMovimiento != models.TipoMovimientoDebito && m.TipoMovimiento != models.TipoMovimientoCredito {
		return nil, nil
	}
	c, err := r.configuracionMonitoreo(ctx, tx)
	if err != nil {
		return nil, err
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM movimientos_cuenta WHERE id_cuenta=$1
		AND tipo_movimiento IN ('DEBITO','CREDITO') AND fecha_creacion>$2::timestamptz-($3::int*interval '1 minute')
		AND fecha_creacion<=$2`, m.IDCuenta, m.FechaCreacion, c.VentanaMinutos).Scan(&count); err != nil {
		return nil, err
	}
	if count <= c.MaxOperaciones {
		return nil, nil
	}
	var reciente bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM anomalias_cuenta WHERE id_cuenta=$1
		AND detectada_en>$2::timestamptz-($3::int*interval '1 minute')
		AND max_operaciones=$4 AND ventana_minutos=$3)`, m.IDCuenta, m.FechaCreacion, c.VentanaMinutos, c.MaxOperaciones).Scan(&reciente)
	if err != nil || reciente {
		return nil, err
	}
	a := &AnomaliaCuenta{IDAnomalia: uuid.New(), IDCuenta: m.IDCuenta, IDCliente: cuenta.IDCliente,
		IDMovimiento: m.IDMovimiento, IDOperacion: m.IDOperacion, TipoCuenta: cuenta.TipoCuenta,
		Operaciones: count, MaxOperaciones: c.MaxOperaciones, VentanaMinutos: c.VentanaMinutos,
		VersionConfiguracion: c.Version, DetectadaEn: m.FechaCreacion, Nivel: "WARNING"}
	_, err = tx.Exec(ctx, `INSERT INTO anomalias_cuenta(id_anomalia,id_cuenta,id_movimiento,id_correlacion,
		operaciones,max_operaciones,ventana_minutos,version_configuracion,detectada_en) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.IDAnomalia, a.IDCuenta, a.IDMovimiento, m.IDCorrelacion, count, c.MaxOperaciones, c.VentanaMinutos, c.Version, m.FechaCreacion)
	if err != nil {
		return nil, err
	}
	err = guardarEventoCuenta(ctx, tx, EventoAccountAnomalyDetected, m.IDCorrelacion, a)
	return a, err
}

func guardarEventoCuenta(ctx context.Context, tx pgx.Tx, tipo string, correlacion uuid.UUID, contenido any) error {
	body, err := json.Marshal(contenido)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida(id_mensaje,tipo_evento,version_evento,contenido,id_correlacion,trace_context)
		VALUES($1,$2,1,$3,$4,$5)`, uuid.New(), tipo, body, correlacion, telemetry.Serialize(ctx))
	return err
}

// Admin commands use a dedicated response event so domain events sharing their
// correlation cannot accidentally satisfy a Gateway query.
func (r *RepositorioCuentasPostgres) EjecutarAdministracion(ctx context.Context, mensaje events.SobreMensaje) error {
	if mensaje.IDMensaje == uuid.Nil || mensaje.IDCorrelacion == uuid.Nil {
		return errors.New("mensaje administrativo invalido")
	}
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Reserve the message id before applying any effects; simultaneous retries
	// wait for the first transaction, then return without applying them again.
	inserted, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado)
		VALUES($1,'account-service.administracion',$2,$3,'{}') ON CONFLICT DO NOTHING`, mensaje.IDMensaje, mensaje.Tipo, mensaje.IDCorrelacion)
	if err != nil {
		return err
	}
	if inserted.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	status, body, err := r.administrar(ctx, tx, mensaje)
	if err != nil {
		return err
	}
	if err = guardarEventoCuenta(ctx, tx, "cuenta.administracion.respondida", mensaje.IDCorrelacion, map[string]any{"estado": status, "cuerpo": body}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *RepositorioCuentasPostgres) administrar(ctx context.Context, tx pgx.Tx, m events.SobreMensaje) (int, any, error) {
	invalid := func(message string) (int, any, error) { return 422, map[string]string{"error": message}, nil }
	switch m.Tipo {
	case "cuenta.administracion.listado.solicitado":
		var p struct {
			IDCuenta       *uuid.UUID `json:"idCuenta"`
			Limite         int        `json:"limite"`
			Desplazamiento int        `json:"desplazamiento"`
		}
		if json.Unmarshal(m.Contenido, &p) != nil {
			return invalid("consulta invalida")
		}
		if p.Limite < 1 || p.Limite > 100 {
			p.Limite = 25
		}
		if p.Desplazamiento < 0 {
			p.Desplazamiento = 0
		}
		rows, err := tx.Query(ctx, `SELECT `+columnasCuenta+` FROM cuentas WHERE ($3::uuid IS NULL OR id_cuenta=$3) ORDER BY fecha_creacion,id_cuenta LIMIT $1 OFFSET $2`, p.Limite, p.Desplazamiento, p.IDCuenta)
		if err != nil {
			return 0, nil, err
		}
		cuentas := make([]models.Cuenta, 0)
		for rows.Next() {
			cuenta, err := escanearCuenta(rows)
			if err != nil {
				rows.Close()
				return 0, nil, err
			}
			cuentas = append(cuentas, *cuenta)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return 0, nil, err
		}
		var total int64
		if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM cuentas WHERE ($1::uuid IS NULL OR id_cuenta=$1)`, p.IDCuenta).Scan(&total); err != nil {
			return 0, nil, err
		}
		return 200, map[string]any{"cuentas": cuentas, "total": total}, nil
	case "cuenta.monitoreo.consulta.solicitada":
		c, err := r.configuracionMonitoreo(ctx, tx)
		return 200, c, err
	case "cuenta.monitoreo.actualizacion.solicitada":
		var p struct {
			models.ConfiguracionMonitoreo
			IDAdministrador uuid.UUID `json:"idAdministrador"`
		}
		if json.Unmarshal(m.Contenido, &p) != nil || !p.Valida() || p.IDAdministrador == uuid.Nil {
			return invalid("Use entre 1 y 10000 operaciones y entre 1 y 1440 minutos")
		}
		err := tx.QueryRow(ctx, `INSERT INTO configuracion_monitoreo_cuentas(id,max_operaciones,ventana_minutos,id_administrador)
			VALUES(TRUE,$1,$2,$3) ON CONFLICT(id) DO UPDATE SET max_operaciones=$1,ventana_minutos=$2,
			id_administrador=$3,version=configuracion_monitoreo_cuentas.version+1,actualizado_en=clock_timestamp() RETURNING version`, p.MaxOperaciones, p.VentanaMinutos, p.IDAdministrador).Scan(&p.Version)
		if err != nil {
			return 0, nil, err
		}
		p.Origen = "ADMINISTRACION"
		if err = guardarEventoCuenta(ctx, tx, "cuenta.monitoreo.configuracion.actualizada", m.IDCorrelacion, p); err != nil {
			return 0, nil, err
		}
		return 200, p.ConfiguracionMonitoreo, nil
	case "cuenta.bloqueo.solicitado":
		var p struct {
			IDCuenta        uuid.UUID `json:"idCuenta"`
			IDAdministrador uuid.UUID `json:"idAdministrador"`
			Motivo          string    `json:"motivo"`
		}
		if json.Unmarshal(m.Contenido, &p) != nil || p.IDCuenta == uuid.Nil || p.IDAdministrador == uuid.Nil || len(p.Motivo) < 5 || len(p.Motivo) > 300 {
			return invalid("Indique la cuenta y un motivo entre 5 y 300 caracteres")
		}
		cuenta, err := bloquearCuenta(ctx, tx, p.IDCuenta)
		if errors.Is(err, ErrCuentaNoEncontrada) {
			return 404, map[string]string{"error": "Cuenta no encontrada"}, nil
		}
		if err != nil {
			return 0, nil, err
		}
		if cuenta.Estado == models.EstadoCuentaCerrada {
			return 409, map[string]string{"error": "La cuenta esta cerrada"}, nil
		}
		if cuenta.Estado != models.EstadoCuentaBloqueada {
			updated, updateErr := escanearCuenta(tx.QueryRow(ctx, `UPDATE cuentas SET estado='BLOQUEADA',version=version+1,fecha_actualizacion=clock_timestamp() WHERE id_cuenta=$1 RETURNING `+columnasCuenta, p.IDCuenta))
			err = updateErr
			if err != nil {
				return 0, nil, err
			}
			cuenta = *updated
			if err = guardarEventoCuenta(ctx, tx, "cuenta.bloqueada", m.IDCorrelacion, map[string]any{"idCuenta": cuenta.IDCuenta, "idCliente": cuenta.IDCliente, "tipoCuenta": cuenta.TipoCuenta, "estado": cuenta.Estado, "motivo": p.Motivo, "idAdministrador": p.IDAdministrador}); err != nil {
				return 0, nil, err
			}
		}
		return 200, cuenta, nil
	default:
		return invalid("Comando administrativo no soportado")
	}
}
