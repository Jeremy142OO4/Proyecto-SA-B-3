package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConectarBaseDatos(ctx context.Context, urlBaseDatos string) (*pgxpool.Pool, error) {
	configuracion, err := pgxpool.ParseConfig(urlBaseDatos)
	if err != nil {
		return nil, fmt.Errorf("configurar conexion a la base de datos: %w", err)
	}

	conexion, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		return nil, fmt.Errorf("crear conexion a la base de datos: %w", err)
	}

	if err := conexion.Ping(ctx); err != nil {
		conexion.Close()
		return nil, fmt.Errorf("verificar conexion a la base de datos: %w", err)
	}

	return conexion, nil
}
