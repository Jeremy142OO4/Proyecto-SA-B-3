package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

var ErrNoEncontrada = errors.New("transferencia no encontrada")

type Postgres struct{ db *pgxpool.Pool }

func NuevoPostgres(db *pgxpool.Pool) *Postgres { return &Postgres{db} }

func (r *Postgres) Iniciar(ctx context.Context, m events.SobreMensaje, t models.Transferencia) (bool, error) {
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	nuevo, e := registrarMensaje(ctx, tx, m, "transaction-service.comandos")
	if e != nil {
		return false, e
	}
	if !nuevo {
		return false, tx.Commit(ctx)
	}
	_, e = tx.Exec(ctx, `INSERT INTO transferencias(id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,id_correlacion,monto_centavos,moneda,descripcion,estado,resultado_externo_simulado,fecha_creacion,fecha_actualizacion) VALUES($1,$2,$3,$4,$5,$6,'GTQ',$7,$8,$9,$10,$10)`, t.IDTransferencia, t.IDCliente, t.IDCuentaOrigen, t.IDCuentaDestino, t.IDCorrelacion, t.MontoCentavos, t.Descripcion, t.Estado, t.ResultadoExternoSimulado, t.FechaCreacion)
	if e != nil {
		return false, e
	}
	p := events.SolicitudValidacionKYC{IDOperacion: t.IDTransferencia, IDCliente: t.IDCliente}
	if e = insertarSalida(ctx, tx, events.ComandoValidarKYC, m.IDCorrelacion, p, true); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}

func (r *Postgres) ProcesarResultadoKYC(ctx context.Context, m events.SobreMensaje, res events.ResultadoValidacionKYC) (bool, error) {
	tx, e := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	nuevo, e := registrarMensaje(ctx, tx, m, "transaction-service.validacion-kyc")
	if e != nil || !nuevo {
		if e == nil {
			e = tx.Commit(ctx)
		}
		return false, e
	}
	var transferencia models.Transferencia
	e = tx.QueryRow(ctx, `SELECT id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,monto_centavos,moneda,COALESCE(descripcion,''),estado FROM transferencias WHERE id_transferencia=$1 FOR UPDATE`, res.IDOperacion).Scan(&transferencia.IDTransferencia, &transferencia.IDCliente, &transferencia.IDCuentaOrigen, &transferencia.IDCuentaDestino, &transferencia.MontoCentavos, &transferencia.Moneda, &transferencia.Descripcion, &transferencia.Estado)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, ErrNoEncontrada
	}
	if e != nil {
		return false, e
	}
	if transferencia.Estado != models.ValidandoKYC {
		return false, tx.Commit(ctx)
	}
	if m.Tipo == events.EventoKYCRechazado || !res.Valido {
		codigo := "KYC_NO_VERIFICADO"
		if _, e = tx.Exec(ctx, `UPDATE transferencias SET estado='RECHAZADA',codigo_error=$1,fecha_actualizacion=NOW() WHERE id_transferencia=$2`, codigo, transferencia.IDTransferencia); e != nil {
			return false, e
		}
		if e = insertarSalida(ctx, tx, events.EventoRechazada, m.IDCorrelacion, datosTransferencia(transferencia, models.Rechazada, codigo, res.Motivo), false); e != nil {
			return false, e
		}
		return true, tx.Commit(ctx)
	}
	if _, e = tx.Exec(ctx, `UPDATE transferencias SET estado='VALIDANDO_CUENTAS',fecha_actualizacion=NOW() WHERE id_transferencia=$1`, transferencia.IDTransferencia); e != nil {
		return false, e
	}
	solicitud := events.SolicitudValidacionCuentas{IDOperacion: transferencia.IDTransferencia, IDCliente: transferencia.IDCliente, IDCuentaOrigen: transferencia.IDCuentaOrigen, IDCuentaDestino: transferencia.IDCuentaDestino, MontoCentavos: transferencia.MontoCentavos}
	if e = insertarSalida(ctx, tx, events.ComandoValidarCuentas, m.IDCorrelacion, solicitud, true); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}

func (r *Postgres) ProcesarResultadoCuentas(ctx context.Context, m events.SobreMensaje, res events.ResultadoValidacionCuentas) (bool, error) {
	tx, e := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	nuevo, e := registrarMensaje(ctx, tx, m, "transaction-service.validacion-cuentas")
	if e != nil || !nuevo {
		if e == nil {
			e = tx.Commit(ctx)
		}
		return false, e
	}
	var transferencia models.Transferencia
	e = tx.QueryRow(ctx, `SELECT id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,monto_centavos,moneda,COALESCE(descripcion,''),estado FROM transferencias WHERE id_transferencia=$1 FOR UPDATE`, res.IDOperacion).Scan(&transferencia.IDTransferencia, &transferencia.IDCliente, &transferencia.IDCuentaOrigen, &transferencia.IDCuentaDestino, &transferencia.MontoCentavos, &transferencia.Moneda, &transferencia.Descripcion, &transferencia.Estado)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, ErrNoEncontrada
	}
	if e != nil {
		return false, e
	}
	if transferencia.Estado != models.ValidandoCuentas {
		return false, tx.Commit(ctx)
	}
	if m.Tipo == events.EventoCuentasRechazadas || !res.Valida {
		codigo := res.Codigo
		if codigo == "" {
			codigo = "CUENTAS_NO_VALIDAS"
		}
		if _, e = tx.Exec(ctx, `UPDATE transferencias SET estado='RECHAZADA',codigo_error=$1,fecha_actualizacion=NOW() WHERE id_transferencia=$2`, codigo, transferencia.IDTransferencia); e != nil {
			return false, e
		}
		if e = insertarSalida(ctx, tx, events.EventoRechazada, m.IDCorrelacion, datosTransferencia(transferencia, models.Rechazada, codigo, res.Motivo), false); e != nil {
			return false, e
		}
		return true, tx.Commit(ctx)
	}
	if _, e = tx.Exec(ctx, `UPDATE transferencias SET estado='PENDIENTE',fecha_actualizacion=NOW() WHERE id_transferencia=$1`, transferencia.IDTransferencia); e != nil {
		return false, e
	}
	solicitud := events.SolicitudMovimiento{IDCuenta: transferencia.IDCuentaOrigen, IDOperacion: transferencia.IDTransferencia, MontoCentavos: transferencia.MontoCentavos}
	if e = insertarSalida(ctx, tx, events.ComandoDebito, m.IDCorrelacion, solicitud, true); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}

func (r *Postgres) ProcesarResultado(ctx context.Context, m events.SobreMensaje, res events.ResultadoMovimiento) (bool, error) {
	tx, e := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	nuevo, e := registrarMensaje(ctx, tx, m, "transaction-service.eventos-cuenta")
	if e != nil {
		return false, e
	}
	if !nuevo {
		return false, tx.Commit(ctx)
	}
	var t models.Transferencia
	e = tx.QueryRow(ctx, `SELECT id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,id_correlacion,monto_centavos,moneda,descripcion,estado,codigo_error,resultado_externo_simulado,fecha_creacion,fecha_actualizacion FROM transferencias WHERE id_transferencia=$1 FOR UPDATE`, res.IDOperacion).Scan(&t.IDTransferencia, &t.IDCliente, &t.IDCuentaOrigen, &t.IDCuentaDestino, &t.IDCorrelacion, &t.MontoCentavos, &t.Moneda, &t.Descripcion, &t.Estado, &t.CodigoError, &t.ResultadoExternoSimulado, &t.FechaCreacion, &t.FechaActualizacion)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, ErrNoEncontrada
	}
	if e != nil {
		return false, e
	}
	var estado models.Estado
	var salida string
	var payload any
	esComando := false
	switch m.Tipo {
	case events.EventoDebitada:
		if t.Estado != models.Pendiente {
			return false, tx.Commit(ctx)
		}
		if t.ResultadoExternoSimulado == "FALLO" || t.ResultadoExternoSimulado == "TIMEOUT" {
			estado = models.Compensando
			salida = events.ComandoCompensacion
			codigo := "FALLO_EXTERNO"
			if t.ResultadoExternoSimulado == "TIMEOUT" {
				codigo = "TIMEOUT_EXTERNO"
			}
			res.Codigo = codigo
			payload = events.SolicitudMovimiento{IDCuenta: t.IDCuentaOrigen, IDOperacion: t.IDTransferencia, MontoCentavos: t.MontoCentavos}
		} else {
			estado = models.Procesando
			salida = events.ComandoCredito
			payload = events.SolicitudMovimiento{IDCuenta: t.IDCuentaDestino, IDOperacion: t.IDTransferencia, MontoCentavos: t.MontoCentavos}
		}
		esComando = true
	case events.EventoDebitoRechazado:
		if t.Estado != models.Pendiente {
			return false, tx.Commit(ctx)
		}
		estado = models.Rechazada
		salida = events.EventoRechazada
		payload = datosTransferencia(t, estado, res.Codigo, res.Mensaje)
	case events.EventoAcreditada:
		if t.Estado != models.Procesando {
			return false, tx.Commit(ctx)
		}
		estado = models.Completada
		salida = events.EventoCompletada
		payload = datosTransferencia(t, estado, "", "")
	case events.EventoCreditoRechazado:
		if t.Estado != models.Procesando {
			return false, tx.Commit(ctx)
		}
		estado = models.Compensando
		salida = events.ComandoCompensacion
		payload = events.SolicitudMovimiento{IDCuenta: t.IDCuentaOrigen, IDOperacion: t.IDTransferencia, MontoCentavos: t.MontoCentavos}
		esComando = true
	case events.EventoCuentaCompensada:
		if t.Estado != models.Compensando {
			return false, tx.Commit(ctx)
		}
		estado = models.Compensada
		salida = events.EventoCompensada
		payload = datosTransferencia(t, estado, "", "")
	case events.EventoCompensacionRechazada:
		if t.Estado != models.Compensando {
			return false, tx.Commit(ctx)
		}
		estado = models.CompensacionFallida
		salida = events.EventoCompensacionFallida
		payload = datosTransferencia(t, estado, res.Codigo, res.Mensaje)
	default:
		return false, fmt.Errorf("evento no soportado %s", m.Tipo)
	}
	codigoError := res.Codigo
	// Al completar una compensación, conservar el motivo del fallo externo
	// registrado al recibir el evento de débito.
	if codigoError == "" {
		codigoError = t.CodigoError
	}
	_, e = tx.Exec(ctx, `UPDATE transferencias SET estado=$1,codigo_error=$2,fecha_actualizacion=NOW() WHERE id_transferencia=$3`, estado, codigoError, t.IDTransferencia)
	if e != nil {
		return false, e
	}
	if e = insertarSalida(ctx, tx, salida, t.IDCorrelacion, payload, esComando); e != nil {
		return false, e
	}
	if estado == models.Procesando || estado == models.Compensando {
		evento := events.EventoProcesando
		if estado == models.Compensando {
			evento = events.EventoCompensando
		}
		if e = insertarSalida(ctx, tx, evento, t.IDCorrelacion, datosTransferencia(t, estado, res.Codigo, ""), false); e != nil {
			return false, e
		}
	}
	return true, tx.Commit(ctx)
}

func datosTransferencia(t models.Transferencia, estado models.Estado, codigo, motivo string) map[string]any {
	datos := map[string]any{
		"idTransferencia":  t.IDTransferencia,
		"idCliente":        t.IDCliente,
		"idCuentaOrigen":   t.IDCuentaOrigen,
		"idCuentaDestino":  t.IDCuentaDestino,
		"montoCentavos":    t.MontoCentavos,
		"moneda":           t.Moneda,
		"estado":           estado,
		"descripcion":      t.Descripcion,
	}
	if codigo != "" {
		datos["codigo"] = codigo
	}
	if motivo != "" {
		datos["motivo"] = motivo
	}
	return datos
}

func (r *Postgres) Consultar(ctx context.Context, id uuid.UUID) (models.Transferencia, error) {
	var t models.Transferencia
	e := r.db.QueryRow(ctx, `SELECT id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,id_correlacion,monto_centavos,moneda,descripcion,estado,codigo_error,resultado_externo_simulado,fecha_creacion,fecha_actualizacion FROM transferencias WHERE id_transferencia=$1`, id).Scan(&t.IDTransferencia, &t.IDCliente, &t.IDCuentaOrigen, &t.IDCuentaDestino, &t.IDCorrelacion, &t.MontoCentavos, &t.Moneda, &t.Descripcion, &t.Estado, &t.CodigoError, &t.ResultadoExternoSimulado, &t.FechaCreacion, &t.FechaActualizacion)
	if errors.Is(e, pgx.ErrNoRows) {
		return t, ErrNoEncontrada
	}
	return t, e
}
func (r *Postgres) Historial(ctx context.Context, p events.SolicitudHistorial) ([]models.Transferencia, error) {
	lim, off := p.Limite, p.Desplazamiento
	if lim <= 0 || lim > 100 {
		lim = 25
	}
	if off < 0 {
		off = 0
	}
	consulta := `SELECT id_transferencia,id_cliente,id_cuenta_origen,id_cuenta_destino,id_correlacion,monto_centavos,moneda,descripcion,estado,codigo_error,resultado_externo_simulado,fecha_creacion,fecha_actualizacion FROM transferencias WHERE id_cliente=$1`
	argumentos := []any{p.IDCliente}
	posicion := 2
	if p.IDCuenta != nil {
		consulta += fmt.Sprintf(" AND (id_cuenta_origen=$%d OR id_cuenta_destino=$%d)", posicion, posicion+1)
		argumentos = append(argumentos, *p.IDCuenta, *p.IDCuenta)
		posicion += 2
	}
	if p.FechaDesde != "" {
		fecha, e := parsearFecha(p.FechaDesde, false)
		if e != nil {
			return nil, e
		}
		consulta += fmt.Sprintf(" AND fecha_creacion >= $%d", posicion)
		argumentos = append(argumentos, fecha)
		posicion++
	}
	if p.FechaHasta != "" {
		fecha, e := parsearFecha(p.FechaHasta, true)
		if e != nil {
			return nil, e
		}
		operador := "<"
		if len(strings.TrimSpace(p.FechaHasta)) != len("2006-01-02") {
			operador = "<="
		}
		consulta += fmt.Sprintf(" AND fecha_creacion %s $%d", operador, posicion)
		argumentos = append(argumentos, fecha)
		posicion++
	}
	if estado := normalizarEstado(p.Estado); estado != "" {
		consulta += fmt.Sprintf(" AND estado=$%d", posicion)
		argumentos = append(argumentos, estado)
		posicion++
	}
	consulta += fmt.Sprintf(" ORDER BY fecha_creacion DESC LIMIT $%d OFFSET $%d", posicion, posicion+1)
	argumentos = append(argumentos, lim, off)
	rows, e := r.db.Query(ctx, consulta, argumentos...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	lista := []models.Transferencia{}
	for rows.Next() {
		var t models.Transferencia
		if e = rows.Scan(&t.IDTransferencia, &t.IDCliente, &t.IDCuentaOrigen, &t.IDCuentaDestino, &t.IDCorrelacion, &t.MontoCentavos, &t.Moneda, &t.Descripcion, &t.Estado, &t.CodigoError, &t.ResultadoExternoSimulado, &t.FechaCreacion, &t.FechaActualizacion); e != nil {
			return nil, e
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}

func parsearFecha(valor string, fin bool) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if len(valor) == len("2006-01-02") {
		fecha, e := time.ParseInLocation("2006-01-02", valor, time.UTC)
		if e != nil {
			return time.Time{}, e
		}
		if fin {
			return fecha.Add(24 * time.Hour), nil
		}
		return fecha, nil
	}
	return time.Parse(time.RFC3339, valor)
}

func normalizarEstado(estado string) string {
	switch strings.ToUpper(strings.TrimSpace(estado)) {
	case "PENDING":
		return string(models.Pendiente)
	case "APPROVED":
		return string(models.Completada)
	case "FAILED":
		return string(models.Rechazada)
	case string(models.ValidandoKYC), string(models.ValidandoCuentas), string(models.Pendiente), string(models.Procesando), string(models.Completada), string(models.Rechazada), string(models.Compensando), string(models.Compensada), string(models.CompensacionFallida):
		return strings.ToUpper(strings.TrimSpace(estado))
	default:
		return ""
	}
}
func (r *Postgres) ResponderConsulta(ctx context.Context, m events.SobreMensaje, tipo string, p any) (bool, error) {
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	nuevo, e := registrarMensaje(ctx, tx, m, "transaction-service.consultas")
	if e != nil || !nuevo {
		if e == nil {
			e = tx.Commit(ctx)
		}
		return false, e
	}
	if e = insertarSalida(ctx, tx, tipo, m.IDCorrelacion, p, false); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}

func registrarMensaje(ctx context.Context, tx pgx.Tx, m events.SobreMensaje, c string) (bool, error) {
	res, e := tx.Exec(ctx, `INSERT INTO mensajes_procesados(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, m.IDMensaje, c, m.Tipo, m.IDCorrelacion)
	return res.RowsAffected() == 1, e
}
func insertarSalida(ctx context.Context, tx pgx.Tx, tipo string, c uuid.UUID, p any, cmd bool) error {
	b, e := json.Marshal(p)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO mensajes_salida(id_mensaje,tipo_evento,contenido,id_correlacion,es_comando) VALUES($1,$2,$3,$4,$5)`, uuid.New(), tipo, b, c, cmd)
	return e
}
func (r *Postgres) ListarPendientes(ctx context.Context, lim int) ([]MensajeSalida, error) {
	rows, e := r.db.Query(ctx, `SELECT id_mensaje,tipo_evento,contenido,id_correlacion,es_comando,fecha_creacion,cantidad_intentos FROM mensajes_salida WHERE estado='PENDIENTE' ORDER BY fecha_creacion LIMIT $1`, lim)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []MensajeSalida
	for rows.Next() {
		var m MensajeSalida
		if e = rows.Scan(&m.IDMensaje, &m.Tipo, &m.Contenido, &m.IDCorrelacion, &m.EsComando, &m.FechaCreacion, &m.Intentos); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *Postgres) MarcarPublicado(ctx context.Context, id uuid.UUID) error {
	_, e := r.db.Exec(ctx, `UPDATE mensajes_salida SET estado='PUBLICADO',fecha_publicacion=NOW() WHERE id_mensaje=$1`, id)
	return e
}
func (r *Postgres) RegistrarFallo(ctx context.Context, id uuid.UUID) error {
	_, e := r.db.Exec(ctx, `UPDATE mensajes_salida SET cantidad_intentos=cantidad_intentos+1 WHERE id_mensaje=$1`, id)
	return e
}
