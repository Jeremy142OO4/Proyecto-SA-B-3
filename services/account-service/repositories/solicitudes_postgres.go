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

type RepositorioSolicitudesPostgres struct{ conexion *pgxpool.Pool }

func NuevoRepositorioSolicitudesPostgres(conexion *pgxpool.Pool) *RepositorioSolicitudesPostgres {
	return &RepositorioSolicitudesPostgres{conexion: conexion}
}

func (r *RepositorioSolicitudesPostgres) Iniciar(ctx context.Context, mensaje events.SobreMensaje, solicitud events.SolicitudCrearCuenta) (bool, error) {
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("iniciar solicitud de cuenta: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	resultado, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados
		(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado)
		VALUES ($1,'account-service.creacion',$2,$3,'{}') ON CONFLICT DO NOTHING`,
		mensaje.IDMensaje, mensaje.Tipo, mensaje.IDCorrelacion)
	if err != nil {
		return false, fmt.Errorf("registrar comando de creacion: %w", err)
	}
	if resultado.RowsAffected() == 0 {
		return false, tx.Commit(ctx)
	}

	_, err = tx.Exec(ctx, `INSERT INTO solicitudes_creacion_cuenta
		(id_solicitud,id_cliente,tipo_cuenta,estado,id_correlacion)
		VALUES ($1,$2,$3,'PENDIENTE_VALIDACION',$4) ON CONFLICT DO NOTHING`,
		solicitud.IDSolicitud, solicitud.IDCliente, solicitud.TipoCuenta, mensaje.IDCorrelacion)
	if err != nil {
		return false, fmt.Errorf("guardar solicitud de cuenta: %w", err)
	}

	contenido, _ := json.Marshal(events.SolicitudValidarCliente{IDSolicitud: solicitud.IDSolicitud, IDCliente: solicitud.IDCliente})
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida
		(id_mensaje,tipo_evento,version_evento,contenido,id_correlacion)
		VALUES ($1,$2,1,$3,$4)`, uuid.New(), events.ComandoValidarCliente, contenido, mensaje.IDCorrelacion)
	if err != nil {
		return false, fmt.Errorf("guardar validacion de cliente en Outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("confirmar solicitud de cuenta: %w", err)
	}
	return true, nil
}

func (r *RepositorioSolicitudesPostgres) Completar(ctx context.Context, mensaje events.SobreMensaje, resultado events.ResultadoValidacionCliente) (*models.Cuenta, bool, error) {
	tx, err := r.conexion.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, false, fmt.Errorf("iniciar confirmacion de cuenta: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	duplicado, err := registrarRespuestaCliente(ctx, tx, mensaje)
	if err != nil {
		return nil, false, err
	}
	if duplicado {
		cuenta, errorCuenta := buscarCuentaDeSolicitud(ctx, tx, resultado.IDSolicitud)
		if errorCuenta != nil {
			return nil, false, errorCuenta
		}
		return cuenta, false, tx.Commit(ctx)
	}

	var solicitud models.SolicitudCreacionCuenta
	err = tx.QueryRow(ctx, `SELECT id_solicitud,id_cliente,tipo_cuenta,estado,id_correlacion,
		id_cuenta,COALESCE(motivo_rechazo,''),fecha_creacion,fecha_actualizacion
		FROM solicitudes_creacion_cuenta WHERE id_solicitud=$1 FOR UPDATE`, resultado.IDSolicitud).Scan(
		&solicitud.IDSolicitud, &solicitud.IDCliente, &solicitud.TipoCuenta, &solicitud.Estado,
		&solicitud.IDCorrelacion, &solicitud.IDCuenta, &solicitud.MotivoRechazo,
		&solicitud.FechaCreacion, &solicitud.FechaActualizacion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, ErrSolicitudNoEncontrada
	}
	if err != nil {
		return nil, false, fmt.Errorf("leer solicitud de cuenta: %w", err)
	}
	if solicitud.Estado != models.EstadoSolicitudPendiente {
		return nil, false, ErrSolicitudNoPendiente
	}

	numero, err := generarNumeroCuentaPostgres(ctx, tx)
	if err != nil {
		return nil, false, err
	}
	ahora := time.Now().UTC()
	cuenta := &models.Cuenta{IDCuenta: uuid.New(), IDCliente: solicitud.IDCliente, NumeroCuenta: numero,
		TipoCuenta: solicitud.TipoCuenta, SaldoCentavos: 0, Moneda: "GTQ", Estado: models.EstadoCuentaActiva,
		FechaCreacion: ahora, FechaActualizacion: ahora, Version: 1}
	_, err = tx.Exec(ctx, `INSERT INTO cuentas
		(id_cuenta,id_cliente,numero_cuenta,tipo_cuenta,saldo_centavos,moneda,estado,fecha_creacion,fecha_actualizacion,version)
		VALUES ($1,$2,$3,$4,0,'GTQ','ACTIVA',$5,$5,1)`, cuenta.IDCuenta, cuenta.IDCliente, cuenta.NumeroCuenta, cuenta.TipoCuenta, ahora)
	if err != nil {
		return nil, false, fmt.Errorf("crear cuenta validada: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE solicitudes_creacion_cuenta SET estado='COMPLETADA',id_cuenta=$1,
		fecha_actualizacion=$2 WHERE id_solicitud=$3`, cuenta.IDCuenta, ahora, solicitud.IDSolicitud)
	if err != nil {
		return nil, false, fmt.Errorf("completar solicitud: %w", err)
	}
	contenido, _ := json.Marshal(cuenta)
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida
		(id_mensaje,tipo_evento,version_evento,contenido,id_correlacion) VALUES ($1,$2,1,$3,$4)`,
		uuid.New(), events.EventoCuentaCreada, contenido, solicitud.IDCorrelacion)
	if err != nil {
		return nil, false, fmt.Errorf("guardar cuenta creada en Outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("confirmar cuenta creada: %w", err)
	}
	return cuenta, true, nil
}

func (r *RepositorioSolicitudesPostgres) Rechazar(ctx context.Context, mensaje events.SobreMensaje, resultado events.ResultadoValidacionCliente) (bool, error) {
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("iniciar rechazo de cuenta: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	duplicado, err := registrarRespuestaCliente(ctx, tx, mensaje)
	if err != nil {
		return false, err
	}
	if duplicado {
		return false, tx.Commit(ctx)
	}
	actualizacion, err := tx.Exec(ctx, `UPDATE solicitudes_creacion_cuenta SET estado='RECHAZADA',
		motivo_rechazo=$1,fecha_actualizacion=NOW() WHERE id_solicitud=$2 AND estado='PENDIENTE_VALIDACION'`,
		resultado.Motivo, resultado.IDSolicitud)
	if err != nil {
		return false, fmt.Errorf("rechazar solicitud: %w", err)
	}
	if actualizacion.RowsAffected() == 0 {
		return false, ErrSolicitudNoPendiente
	}
	contenido, _ := json.Marshal(resultado)
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida
		(id_mensaje,tipo_evento,version_evento,contenido,id_correlacion) VALUES ($1,$2,1,$3,$4)`,
		uuid.New(), events.EventoCreacionCuentaRechazada, contenido, mensaje.IDCorrelacion)
	if err != nil {
		return false, fmt.Errorf("guardar rechazo en Outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func registrarRespuestaCliente(ctx context.Context, tx pgx.Tx, mensaje events.SobreMensaje) (bool, error) {
	resultado, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados
		(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado)
		VALUES ($1,'account-service.validacion-cliente',$2,$3,'{}') ON CONFLICT DO NOTHING`,
		mensaje.IDMensaje, mensaje.Tipo, mensaje.IDCorrelacion)
	if err != nil {
		return false, fmt.Errorf("registrar respuesta de cliente: %w", err)
	}
	return resultado.RowsAffected() == 0, nil
}

func buscarCuentaDeSolicitud(ctx context.Context, tx pgx.Tx, idSolicitud uuid.UUID) (*models.Cuenta, error) {
	cuenta, err := escanearCuenta(tx.QueryRow(ctx, `SELECT `+columnasCuenta+` FROM cuentas c
		JOIN solicitudes_creacion_cuenta s ON s.id_cuenta=c.id_cuenta WHERE s.id_solicitud=$1`, idSolicitud))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSolicitudNoEncontrada
	}
	return cuenta, err
}

func generarNumeroCuentaPostgres(ctx context.Context, tx pgx.Tx) (string, error) {
	for intento := 0; intento < 5; intento++ {
		var numero string
		if err := tx.QueryRow(ctx, `SELECT '10'||LPAD(FLOOR(RANDOM()*10000000000)::BIGINT::TEXT,10,'0')`).Scan(&numero); err != nil {
			return "", err
		}
		var existe bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cuentas WHERE numero_cuenta=$1)`, numero).Scan(&existe); err != nil {
			return "", err
		}
		if !existe {
			return numero, nil
		}
	}
	return "", fmt.Errorf("no se pudo generar un numero de cuenta unico")
}
