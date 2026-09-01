package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositorioCuentasPostgres struct {
	conexion *pgxpool.Pool
}

func NuevoRepositorioCuentasPostgres(conexion *pgxpool.Pool) *RepositorioCuentasPostgres {
	return &RepositorioCuentasPostgres{conexion: conexion}
}

func (r *RepositorioCuentasPostgres) Crear(ctx context.Context, cuenta *models.Cuenta) error {
	const consulta = `
		INSERT INTO cuentas (
			id_cuenta, id_cliente, numero_cuenta, tipo_cuenta, saldo_centavos,
			moneda, estado, ultima_actividad, fecha_creacion, fecha_actualizacion, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.conexion.Exec(ctx, consulta,
		cuenta.IDCuenta, cuenta.IDCliente, cuenta.NumeroCuenta, cuenta.TipoCuenta,
		cuenta.SaldoCentavos, cuenta.Moneda, cuenta.Estado, cuenta.UltimaActividad,
		cuenta.FechaCreacion, cuenta.FechaActualizacion, cuenta.Version,
	)
	if err != nil {
		return fmt.Errorf("crear cuenta: %w", err)
	}
	return nil
}

func (r *RepositorioCuentasPostgres) BuscarPorID(ctx context.Context, idCuenta uuid.UUID) (*models.Cuenta, error) {
	return r.buscarUna(ctx, `SELECT `+columnasCuenta+` FROM cuentas WHERE id_cuenta = $1`, idCuenta)
}

func (r *RepositorioCuentasPostgres) BuscarPorNumero(ctx context.Context, numeroCuenta string) (*models.Cuenta, error) {
	return r.buscarUna(ctx, `SELECT `+columnasCuenta+` FROM cuentas WHERE numero_cuenta = $1`, numeroCuenta)
}

func (r *RepositorioCuentasPostgres) ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]models.Cuenta, error) {
	filas, err := r.conexion.Query(ctx, `SELECT `+columnasCuenta+` FROM cuentas WHERE id_cliente = $1 ORDER BY fecha_creacion`, idCliente)
	if err != nil {
		return nil, fmt.Errorf("listar cuentas del cliente: %w", err)
	}
	defer filas.Close()

	cuentas := make([]models.Cuenta, 0)
	for filas.Next() {
		cuenta, err := escanearCuenta(filas)
		if err != nil {
			return nil, err
		}
		cuentas = append(cuentas, *cuenta)
	}
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("recorrer cuentas del cliente: %w", err)
	}
	return cuentas, nil
}

func (r *RepositorioCuentasPostgres) ProcesarMovimiento(ctx context.Context, solicitud SolicitudMovimientoCuenta) (ResultadoMovimiento, error) {
	transaccion, err := r.conexion.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ResultadoMovimiento{}, fmt.Errorf("iniciar transaccion de movimiento: %w", err)
	}
	defer func() { _ = transaccion.Rollback(ctx) }()

	duplicado, err := mensajeYaProcesado(ctx, transaccion, solicitud.IDMensaje, solicitud.NombreConsumidor)
	if err != nil {
		return ResultadoMovimiento{}, err
	}
	if duplicado {
		movimiento, err := buscarMovimientoPorOperacion(ctx, transaccion, solicitud.IDOperacion, solicitud.TipoMovimiento)
		if err != nil {
			return ResultadoMovimiento{}, err
		}
		if err := transaccion.Commit(ctx); err != nil {
			return ResultadoMovimiento{}, fmt.Errorf("confirmar lectura de mensaje duplicado: %w", err)
		}
		return ResultadoMovimiento{Movimiento: movimiento, Duplicado: true}, nil
	}

	cuenta, err := bloquearCuenta(ctx, transaccion, solicitud.IDCuenta)
	if err != nil {
		return ResultadoMovimiento{}, err
	}
	if cuenta.Estado != models.EstadoCuentaActiva {
		return ResultadoMovimiento{}, ErrCuentaNoActiva
	}

	saldoAnterior := cuenta.SaldoCentavos
	saldoNuevo, err := calcularSaldoNuevo(ctx, transaccion, cuenta, solicitud)
	if err != nil {
		return ResultadoMovimiento{}, err
	}

	ahora := time.Now().UTC()
	movimiento := models.MovimientoCuenta{
		IDMovimiento:          uuid.New(),
		IDCuenta:              solicitud.IDCuenta,
		IDOperacion:           solicitud.IDOperacion,
		IDCorrelacion:         solicitud.IDCorrelacion,
		TipoMovimiento:        solicitud.TipoMovimiento,
		MontoCentavos:         solicitud.MontoCentavos,
		SaldoAnteriorCentavos: saldoAnterior,
		SaldoNuevoCentavos:    saldoNuevo,
		Descripcion:           solicitud.Descripcion,
		FechaCreacion:         ahora,
	}

	if err := actualizarSaldo(ctx, transaccion, solicitud.IDCuenta, saldoNuevo, ahora, cuenta.Version); err != nil {
		return ResultadoMovimiento{}, err
	}
	if err := insertarMovimiento(ctx, transaccion, movimiento); err != nil {
		return ResultadoMovimiento{}, err
	}
	if err := registrarMensajeProcesado(ctx, transaccion, solicitud, movimiento.IDMovimiento); err != nil {
		return ResultadoMovimiento{}, err
	}
	if err := insertarEventoSalida(ctx, transaccion, solicitud, movimiento); err != nil {
		return ResultadoMovimiento{}, err
	}

	if err := transaccion.Commit(ctx); err != nil {
		return ResultadoMovimiento{}, fmt.Errorf("confirmar movimiento: %w", err)
	}
	return ResultadoMovimiento{Movimiento: movimiento}, nil
}

func (r *RepositorioCuentasPostgres) ListarMovimientos(ctx context.Context, idCuenta uuid.UUID, limite, desplazamiento int) ([]models.MovimientoCuenta, error) {
	filas, err := r.conexion.Query(ctx, `SELECT id_movimiento,id_cuenta,id_operacion,id_correlacion,
		tipo_movimiento,monto_centavos,saldo_anterior_centavos,saldo_nuevo_centavos,
		descripcion,fecha_creacion FROM movimientos_cuenta WHERE id_cuenta=$1
		ORDER BY fecha_creacion DESC LIMIT $2 OFFSET $3`, idCuenta, limite, desplazamiento)
	if err != nil {
		return nil, fmt.Errorf("listar movimientos: %w", err)
	}
	defer filas.Close()
	movimientos := make([]models.MovimientoCuenta, 0)
	for filas.Next() {
		var m models.MovimientoCuenta
		if err := filas.Scan(&m.IDMovimiento, &m.IDCuenta, &m.IDOperacion, &m.IDCorrelacion,
			&m.TipoMovimiento, &m.MontoCentavos, &m.SaldoAnteriorCentavos, &m.SaldoNuevoCentavos,
			&m.Descripcion, &m.FechaCreacion); err != nil {
			return nil, fmt.Errorf("leer movimiento: %w", err)
		}
		movimientos = append(movimientos, m)
	}
	return movimientos, filas.Err()
}

func (r *RepositorioCuentasPostgres) DesactivarCuentasInactivas(ctx context.Context, fechaLimite time.Time, saldoMaximoCentavos int64) (int64, error) {
	var cantidad int64
	err := r.conexion.QueryRow(ctx, `WITH desactivadas AS (
		UPDATE cuentas SET estado='INACTIVA',fecha_actualizacion=NOW(),version=version+1
		WHERE estado='ACTIVA' AND saldo_centavos < $1
		AND COALESCE(ultima_actividad,fecha_creacion) < $2
		RETURNING id_cuenta,id_cliente,numero_cuenta,estado
	), eventos AS (
		INSERT INTO mensajes_salida (id_mensaje,tipo_evento,version_evento,contenido,id_correlacion)
		SELECT gen_random_uuid(),$3,1,jsonb_build_object('idCuenta',id_cuenta,'idCliente',id_cliente,
		'numeroCuenta',numero_cuenta,'estado',estado),gen_random_uuid() FROM desactivadas
		RETURNING 1
	) SELECT COUNT(*) FROM eventos`, saldoMaximoCentavos, fechaLimite, events.EventoCuentaDesactivada).Scan(&cantidad)
	if err != nil {
		return 0, fmt.Errorf("desactivar cuentas inactivas: %w", err)
	}
	return cantidad, nil
}

const columnasCuenta = `id_cuenta, id_cliente, numero_cuenta, tipo_cuenta, saldo_centavos,
	moneda, estado, ultima_actividad, fecha_creacion, fecha_actualizacion, version`

type escaneador interface {
	Scan(destinos ...any) error
}

func (r *RepositorioCuentasPostgres) buscarUna(ctx context.Context, consulta string, argumento any) (*models.Cuenta, error) {
	cuenta, err := escanearCuenta(r.conexion.QueryRow(ctx, consulta, argumento))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCuentaNoEncontrada
	}
	return cuenta, err
}

func escanearCuenta(fila escaneador) (*models.Cuenta, error) {
	var cuenta models.Cuenta
	err := fila.Scan(
		&cuenta.IDCuenta, &cuenta.IDCliente, &cuenta.NumeroCuenta, &cuenta.TipoCuenta,
		&cuenta.SaldoCentavos, &cuenta.Moneda, &cuenta.Estado, &cuenta.UltimaActividad,
		&cuenta.FechaCreacion, &cuenta.FechaActualizacion, &cuenta.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("leer cuenta: %w", err)
	}
	return &cuenta, nil
}

func mensajeYaProcesado(ctx context.Context, tx pgx.Tx, idMensaje uuid.UUID, consumidor string) (bool, error) {
	var existe bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM mensajes_procesados WHERE id_mensaje = $1 AND nombre_consumidor = $2
	)`, idMensaje, consumidor).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("consultar idempotencia: %w", err)
	}
	return existe, nil
}

func bloquearCuenta(ctx context.Context, tx pgx.Tx, idCuenta uuid.UUID) (models.Cuenta, error) {
	cuenta, err := escanearCuenta(tx.QueryRow(ctx, `SELECT `+columnasCuenta+` FROM cuentas WHERE id_cuenta = $1 FOR UPDATE`, idCuenta))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Cuenta{}, ErrCuentaNoEncontrada
	}
	if err != nil {
		return models.Cuenta{}, err
	}
	return *cuenta, nil
}

func calcularSaldoNuevo(ctx context.Context, tx pgx.Tx, cuenta models.Cuenta, solicitud SolicitudMovimientoCuenta) (int64, error) {
	switch solicitud.TipoMovimiento {
	case models.TipoMovimientoDebito:
		if cuenta.SaldoCentavos < solicitud.MontoCentavos {
			return 0, ErrFondosInsuficientes
		}
		return cuenta.SaldoCentavos - solicitud.MontoCentavos, nil
	case models.TipoMovimientoCredito:
		return cuenta.SaldoCentavos + solicitud.MontoCentavos, nil
	case models.TipoMovimientoCompensacion:
		var existe bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM movimientos_cuenta
			WHERE id_operacion = $1 AND tipo_movimiento = 'DEBITO'
		)`, solicitud.IDOperacion).Scan(&existe)
		if err != nil {
			return 0, fmt.Errorf("verificar debito original: %w", err)
		}
		if !existe {
			return 0, ErrMovimientoNoEncontrado
		}
		return cuenta.SaldoCentavos + solicitud.MontoCentavos, nil
	default:
		return 0, fmt.Errorf("tipo de movimiento no soportado: %s", solicitud.TipoMovimiento)
	}
}

func actualizarSaldo(ctx context.Context, tx pgx.Tx, idCuenta uuid.UUID, saldoNuevo int64, ahora time.Time, version int64) error {
	resultado, err := tx.Exec(ctx, `UPDATE cuentas SET saldo_centavos = $1, ultima_actividad = $2,
		fecha_actualizacion = $2, version = version + 1 WHERE id_cuenta = $3 AND version = $4`,
		saldoNuevo, ahora, idCuenta, version)
	if err != nil {
		return fmt.Errorf("actualizar saldo: %w", err)
	}
	if resultado.RowsAffected() != 1 {
		return fmt.Errorf("la cuenta fue modificada concurrentemente")
	}
	return nil
}

func insertarMovimiento(ctx context.Context, tx pgx.Tx, movimiento models.MovimientoCuenta) error {
	_, err := tx.Exec(ctx, `INSERT INTO movimientos_cuenta (
		id_movimiento, id_cuenta, id_operacion, id_correlacion, tipo_movimiento,
		monto_centavos, saldo_anterior_centavos, saldo_nuevo_centavos, descripcion, fecha_creacion
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		movimiento.IDMovimiento, movimiento.IDCuenta, movimiento.IDOperacion,
		movimiento.IDCorrelacion, movimiento.TipoMovimiento, movimiento.MontoCentavos,
		movimiento.SaldoAnteriorCentavos, movimiento.SaldoNuevoCentavos,
		movimiento.Descripcion, movimiento.FechaCreacion)
	if err != nil {
		return fmt.Errorf("registrar movimiento: %w", err)
	}
	return nil
}

func registrarMensajeProcesado(ctx context.Context, tx pgx.Tx, solicitud SolicitudMovimientoCuenta, idMovimiento uuid.UUID) error {
	resultado, err := json.Marshal(map[string]any{"idMovimiento": idMovimiento})
	if err != nil {
		return fmt.Errorf("serializar resultado del mensaje: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_procesados (
		id_mensaje, nombre_consumidor, tipo_mensaje, id_correlacion, resultado
	) VALUES ($1,$2,$3,$4,$5)`, solicitud.IDMensaje, solicitud.NombreConsumidor,
		solicitud.TipoMovimiento, solicitud.IDCorrelacion, resultado)
	if err != nil {
		return fmt.Errorf("registrar mensaje procesado: %w", err)
	}
	return nil
}

func insertarEventoSalida(ctx context.Context, tx pgx.Tx, solicitud SolicitudMovimientoCuenta, movimiento models.MovimientoCuenta) error {
	contenido, err := json.Marshal(movimiento)
	if err != nil {
		return fmt.Errorf("serializar evento de movimiento: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida (
		id_mensaje, tipo_evento, version_evento, contenido, id_correlacion
	) VALUES ($1,$2,1,$3,$4)`, uuid.New(), solicitud.TipoEventoExitoso, contenido, solicitud.IDCorrelacion)
	if err != nil {
		return fmt.Errorf("registrar evento de salida: %w", err)
	}
	return nil
}

func buscarMovimientoPorOperacion(ctx context.Context, tx pgx.Tx, idOperacion uuid.UUID, tipo models.TipoMovimiento) (models.MovimientoCuenta, error) {
	var movimiento models.MovimientoCuenta
	err := tx.QueryRow(ctx, `SELECT id_movimiento, id_cuenta, id_operacion, id_correlacion,
		tipo_movimiento, monto_centavos, saldo_anterior_centavos, saldo_nuevo_centavos,
		descripcion, fecha_creacion FROM movimientos_cuenta
		WHERE id_operacion = $1 AND tipo_movimiento = $2`, idOperacion, tipo).Scan(
		&movimiento.IDMovimiento, &movimiento.IDCuenta, &movimiento.IDOperacion,
		&movimiento.IDCorrelacion, &movimiento.TipoMovimiento, &movimiento.MontoCentavos,
		&movimiento.SaldoAnteriorCentavos, &movimiento.SaldoNuevoCentavos,
		&movimiento.Descripcion, &movimiento.FechaCreacion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.MovimientoCuenta{}, ErrMovimientoNoEncontrado
	}
	if err != nil {
		return models.MovimientoCuenta{}, fmt.Errorf("buscar movimiento procesado: %w", err)
	}
	return movimiento, nil
}
