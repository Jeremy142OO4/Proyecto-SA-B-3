package services

import (
	"context"
	"log/slog"
	"time"
)

func EjecutarProcesoInactividad(ctx context.Context, servicio ServicioCuentas, intervalo time.Duration) {
	temporizador := time.NewTicker(intervalo)
	defer temporizador.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-temporizador.C:
			cantidad, err := servicio.DesactivarCuentasInactivas(ctx)
			if err != nil {
				slog.Error("fallo el proceso de inactividad", "error", err)
				continue
			}
			if cantidad > 0 {
				slog.Info("cuentas inactivas desactivadas", "cantidad", cantidad)
			}
		}
	}
}
