package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ConectarBaseDatos(ctx context.Context, url string) (*pgxpool.Pool, error) {
	conexion, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("conectar base de pagos: %w", err)
	}
	if err = conexion.Ping(ctx); err != nil {
		conexion.Close()
		return nil, fmt.Errorf("verificar base de pagos: %w", err)
	}
	return conexion, nil
}
