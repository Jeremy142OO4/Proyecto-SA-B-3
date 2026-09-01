package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/config"
	"github.com/Proyecto-SA-B-3/account-service/database"
	"github.com/Proyecto-SA-B-3/account-service/messaging"
	"github.com/Proyecto-SA-B-3/account-service/messaging/consumers"
	"github.com/Proyecto-SA-B-3/account-service/messaging/publishers"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
	"github.com/Proyecto-SA-B-3/account-service/routes"
	"github.com/Proyecto-SA-B-3/account-service/services"
	"github.com/gofiber/fiber/v2"
)

func main() {
	registrador := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(registrador)

	configuracion, err := config.CargarConfiguracion()
	if err != nil {
		slog.Error("no se pudo cargar la configuracion", "error", err)
		os.Exit(1)
	}

	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()

	conexionBaseDatos, err := database.ConectarBaseDatos(ctx, configuracion.URLBaseDatos)
	if err != nil {
		slog.Error("no se pudo iniciar la base de datos", "error", err)
		os.Exit(1)
	}
	defer conexionBaseDatos.Close()

	conexionRabbitMQ, err := messaging.ConectarRabbitMQ(configuracion.URLRabbitMQ)
	if err != nil {
		slog.Error("no se pudo iniciar RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer conexionRabbitMQ.Close()

	canalTopologia, err := conexionRabbitMQ.Channel()
	if err != nil {
		slog.Error("no se pudo crear el canal de topologia", "error", err)
		os.Exit(1)
	}
	if err := messaging.DeclararTopologia(canalTopologia); err != nil {
		_ = canalTopologia.Close()
		slog.Error("no se pudo declarar la topologia RabbitMQ", "error", err)
		os.Exit(1)
	}
	_ = canalTopologia.Close()

	repositorioCuentas := repositories.NuevoRepositorioCuentasPostgres(conexionBaseDatos)
	repositorioSalida := repositories.NuevoRepositorioSalidaPostgres(conexionBaseDatos)
	repositorioSolicitudes := repositories.NuevoRepositorioSolicitudesPostgres(conexionBaseDatos)
	servicioCuentas := services.NuevoServicioCuentas(repositorioCuentas)
	servicioCreacion := services.NuevoServicioCreacionCuentas(repositorioSolicitudes)
	publicadorComandos, err := messaging.NuevoPublicador(conexionRabbitMQ)
	if err != nil {
		slog.Error("no se pudo crear el publicador de comandos", "error", err)
		os.Exit(1)
	}
	defer publicadorComandos.Cerrar()
	publicadorEventos, err := messaging.NuevoPublicador(conexionRabbitMQ)
	if err != nil {
		slog.Error("no se pudo crear el publicador de eventos", "error", err)
		os.Exit(1)
	}
	defer publicadorEventos.Cerrar()

	consumidor, err := consumers.NuevoConsumidorCuentas(
		conexionRabbitMQ, servicioCuentas, servicioCreacion, repositorioSalida, publicadorComandos, configuracion.MaximoReintentos,
	)
	if err != nil {
		slog.Error("no se pudo crear el consumidor", "error", err)
		os.Exit(1)
	}
	defer consumidor.Cerrar()
	consumidorValidacion, err := consumers.NuevoConsumidorValidacionCliente(conexionRabbitMQ, servicioCreacion)
	if err != nil {
		slog.Error("no se pudo crear el consumidor de validaciones", "error", err)
		os.Exit(1)
	}
	defer consumidorValidacion.Cerrar()

	publicadorOutbox := publishers.NuevoPublicadorOutbox(repositorioSalida, publicadorEventos, configuracion.IntervaloOutbox)
	go publicadorOutbox.Ejecutar(ctx)
	go func() {
		if err := consumidor.Ejecutar(ctx); err != nil && ctx.Err() == nil {
			slog.Error("el consumidor termino inesperadamente", "error", err)
			cancelar()
		}
	}()
	go func() {
		if err := consumidorValidacion.Ejecutar(ctx); err != nil && ctx.Err() == nil {
			slog.Error("el consumidor de validaciones termino", "error", err)
			cancelar()
		}
	}()
	go services.EjecutarProcesoInactividad(ctx, servicioCuentas, 24*time.Hour)

	aplicacion := fiber.New(fiber.Config{
		AppName:               "account-service",
		DisableStartupMessage: true,
	})
	routes.RegistrarRutas(aplicacion)

	erroresServidor := make(chan error, 1)
	go func() {
		slog.Info("servicio de cuentas iniciado", "puerto", configuracion.PuertoHTTP, "entorno", configuracion.Entorno)
		erroresServidor <- aplicacion.Listen(":" + configuracion.PuertoHTTP)
	}()

	select {
	case <-ctx.Done():
		slog.Info("apagando servicio de cuentas")
	case err := <-erroresServidor:
		if err != nil && !errors.Is(err, fiber.ErrServiceUnavailable) {
			slog.Error("el servidor HTTP termino inesperadamente", "error", err)
		}
	}

	ctxApagado, cancelarApagado := context.WithTimeout(context.Background(), configuracion.TiempoEsperaApagado)
	defer cancelarApagado()

	canalFinalizacion := make(chan error, 1)
	go func() { canalFinalizacion <- aplicacion.Shutdown() }()

	select {
	case err := <-canalFinalizacion:
		if err != nil {
			slog.Error("no se pudo apagar el servidor correctamente", "error", err)
		}
	case <-ctxApagado.Done():
		slog.Error("se agoto el tiempo de apagado", "espera", configuracion.TiempoEsperaApagado.Round(time.Second))
	}
}
