package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositorioSalidaPostgres struct {
	conexion *pgxpool.Pool
}

func NuevoRepositorioSalidaPostgres(conexion *pgxpool.Pool) *RepositorioSalidaPostgres {
	return &RepositorioSalidaPostgres{conexion: conexion}
}

func (r *RepositorioSalidaPostgres) ListarPendientes(ctx context.Context, limite int) ([]MensajeSalida, error) {
	filas, err := r.conexion.Query(ctx, `SELECT id_mensaje, tipo_evento, version_evento,
		contenido, id_correlacion, fecha_creacion, cantidad_intentos
		FROM mensajes_salida WHERE estado = 'PENDIENTE'
		ORDER BY fecha_creacion LIMIT $1`, limite)
	if err != nil {
		return nil, fmt.Errorf("listar mensajes de salida: %w", err)
	}
	defer filas.Close()

	mensajes := make([]MensajeSalida, 0)
	for filas.Next() {
		var mensaje MensajeSalida
		if err := filas.Scan(&mensaje.IDMensaje, &mensaje.TipoEvento, &mensaje.VersionEvento,
			&mensaje.Contenido, &mensaje.IDCorrelacion, &mensaje.FechaCreacion, &mensaje.CantidadIntentos); err != nil {
			return nil, fmt.Errorf("leer mensaje de salida: %w", err)
		}
		mensajes = append(mensajes, mensaje)
	}
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("recorrer mensajes de salida: %w", err)
	}
	return mensajes, nil
}

func (r *RepositorioSalidaPostgres) MarcarPublicado(ctx context.Context, idMensaje uuid.UUID) error {
	_, err := r.conexion.Exec(ctx, `UPDATE mensajes_salida SET estado = 'PUBLICADO',
		fecha_publicacion = NOW() WHERE id_mensaje = $1`, idMensaje)
	if err != nil {
		return fmt.Errorf("marcar mensaje publicado: %w", err)
	}
	return nil
}

func (r *RepositorioSalidaPostgres) RegistrarFalloPublicacion(ctx context.Context, idMensaje uuid.UUID) error {
	_, err := r.conexion.Exec(ctx, `UPDATE mensajes_salida SET cantidad_intentos = cantidad_intentos + 1
		WHERE id_mensaje = $1`, idMensaje)
	if err != nil {
		return fmt.Errorf("registrar fallo de publicacion: %w", err)
	}
	return nil
}

func (r *RepositorioSalidaPostgres) RegistrarRechazo(
	ctx context.Context,
	mensaje events.SobreMensaje,
	consumidor string,
	tipoEvento string,
	contenido any,
) (bool, error) {
	transaccion, err := r.conexion.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("iniciar registro de rechazo: %w", err)
	}
	defer func() { _ = transaccion.Rollback(ctx) }()

	resultado, err := transaccion.Exec(ctx, `INSERT INTO mensajes_procesados (
		id_mensaje, nombre_consumidor, tipo_mensaje, id_correlacion, resultado
	) VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, mensaje.IDMensaje, consumidor,
		mensaje.Tipo, mensaje.IDCorrelacion, []byte(`{"rechazado":true}`))
	if err != nil {
		return false, fmt.Errorf("registrar mensaje rechazado: %w", err)
	}
	if resultado.RowsAffected() == 0 {
		if err := transaccion.Commit(ctx); err != nil {
			return false, fmt.Errorf("confirmar rechazo duplicado: %w", err)
		}
		return false, nil
	}

	contenidoJSON, err := json.Marshal(contenido)
	if err != nil {
		return false, fmt.Errorf("serializar evento de rechazo: %w", err)
	}
	_, err = transaccion.Exec(ctx, `INSERT INTO mensajes_salida (
		id_mensaje, tipo_evento, version_evento, contenido, id_correlacion
	) VALUES ($1,$2,1,$3,$4)`, uuid.New(), tipoEvento, contenidoJSON, mensaje.IDCorrelacion)
	if err != nil {
		return false, fmt.Errorf("guardar evento de rechazo: %w", err)
	}

	if err := transaccion.Commit(ctx); err != nil {
		return false, fmt.Errorf("confirmar evento de rechazo: %w", err)
	}
	return true, nil
}

func (r *RepositorioSalidaPostgres) RegistrarRespuesta(ctx context.Context, mensaje events.SobreMensaje, consumidor, tipoEvento string, contenido any) (bool, error) {
	tx, err := r.conexion.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	resultado, err := tx.Exec(ctx, `INSERT INTO mensajes_procesados
		(id_mensaje,nombre_consumidor,tipo_mensaje,id_correlacion,resultado)
		VALUES($1,$2,$3,$4,'{}') ON CONFLICT DO NOTHING`, mensaje.IDMensaje, consumidor, mensaje.Tipo, mensaje.IDCorrelacion)
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
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida(id_mensaje,tipo_evento,version_evento,contenido,id_correlacion)
		VALUES($1,$2,1,$3,$4)`, uuid.New(), tipoEvento, jsonContenido, mensaje.IDCorrelacion)
	if err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
