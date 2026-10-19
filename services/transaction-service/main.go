package main

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/config"
	"github.com/Proyecto-SA-B-3/transaction-service/database"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/recovery"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/transaction-service/messaging"
	"github.com/Proyecto-SA-B-3/transaction-service/messaging/consumers"
	"github.com/Proyecto-SA-B-3/transaction-service/messaging/publishers"
	"github.com/Proyecto-SA-B-3/transaction-service/repositories"
	"github.com/Proyecto-SA-B-3/transaction-service/routes"
	"github.com/Proyecto-SA-B-3/transaction-service/services"
	"github.com/gofiber/fiber/v2"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	shutdownTelemetry := telemetry.Setup("transaction-service")
	defer shutdownTelemetry()
	cfg, e := config.CargarConfiguracion()
	if e != nil {
		slog.Error("configuracion invalida", "error", e)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, e := database.Conectar(ctx, cfg.URLBaseDatos)
	if e != nil {
		slog.Error("base de datos no disponible", "error", e)
		os.Exit(1)
	}
	defer db.Close()
	mensajeriaFin := make(chan struct{})
	go func() {
		defer close(mensajeriaFin)
		recovery.Run(ctx, func(ctx context.Context) error {
			rabbit, e := messaging.Conectar(cfg.URLRabbitMQ)
			if e != nil {
				slog.Error("RabbitMQ no disponible", "error", e)
				return e
			}
			defer rabbit.Close()
			ch, e := rabbit.Channel()
			if e != nil {
				return e
			}
			if e = messaging.DeclararTopologia(ch); e != nil {
				return e
			}
			ch.Close()
			repo := repositories.NuevoPostgres(db)
			serv := services.Nuevo(repo)
			pubComandos, e := messaging.NuevoPublicador(rabbit)
			if e != nil {
				return e
			}
			defer pubComandos.Cerrar()
			pubEventos, e := messaging.NuevoPublicador(rabbit)
			if e != nil {
				return e
			}
			defer pubEventos.Cerrar()
			cons, e := consumers.Nuevo(rabbit, serv, pubComandos, cfg.MaximoReintentos)
			if e != nil {
				return e
			}
			defer cons.Cerrar()

			outbox := publishers.Nuevo(repo, pubEventos, cfg.IntervaloOutbox)
			return recovery.Serve(ctx, func() { _ = rabbit.CloseDeadline(time.Now().Add(time.Second)) },
				[]func() bool{rabbit.IsClosed, pubComandos.Cerrado, pubEventos.Cerrado, cons.Cerrado},
				cons.Ejecutar, func(ctx context.Context) error { outbox.Ejecutar(ctx); return nil })

		})
	}()
	defer func() { cancel(); <-mensajeriaFin }()

	app := fiber.New(fiber.Config{AppName: "transaction-service", DisableStartupMessage: true})

	routes.Registrar(app, repositories.NuevoPostgres(db))
	errores := make(chan error, 1)
	go func() {
		slog.Info("transaction-service iniciado", "puerto", cfg.PuertoHTTP)
		errores <- app.Listen(":" + cfg.PuertoHTTP)
	}()
	select {
	case <-ctx.Done():
	case e := <-errores:
		if e != nil && !errors.Is(e, fiber.ErrServiceUnavailable) {
			slog.Error("HTTP detenido", "error", e)
		}
	}
	apagado, c := context.WithTimeout(context.Background(), cfg.TiempoEsperaApagado)
	defer c()
	fin := make(chan error, 1)
	go func() { fin <- app.Shutdown() }()
	select {
	case <-fin:
	case <-apagado.Done():
		slog.Error("timeout de apagado")
	}
}
