package main

import (
	"context"
	"github.com/Proyecto-SA-B-3/payment-service/config"
	"github.com/Proyecto-SA-B-3/payment-service/database"
	"github.com/Proyecto-SA-B-3/payment-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/payment-service/internal/interbank"
	"github.com/Proyecto-SA-B-3/payment-service/internal/recovery"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/payment-service/messaging"
	"github.com/Proyecto-SA-B-3/payment-service/messaging/consumers"
	"github.com/Proyecto-SA-B-3/payment-service/messaging/publishers"
	"github.com/Proyecto-SA-B-3/payment-service/repositories"
	"github.com/Proyecto-SA-B-3/payment-service/routes"
	"github.com/Proyecto-SA-B-3/payment-service/services"
	"github.com/gofiber/fiber/v2"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	shutdownTelemetry := telemetry.Setup("payment-service")
	defer shutdownTelemetry()
	cfg, e := config.CargarConfiguracion()
	if e != nil {
		slog.Error("configuracion invalida", "error", e)
		os.Exit(1)
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	bd, e := database.ConectarBaseDatos(ctx, cfg.URLBaseDatos)
	if e != nil {
		slog.Error("base de pagos no disponible", "error", e)
		os.Exit(1)
	}
	defer bd.Close()
	if os.Getenv("INTERBANK_ENABLED") == "true" {
		transport, err := interbank.New()
		if err != nil {
			slog.Error("configuracion interbancaria invalida")
			os.Exit(1)
		}
		go bankrpc.Run(ctx, cfg.URLRabbitMQ, interbank.Queue, transport.Handle)
	}
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
				slog.Error("topologia de pagos invalida", "error", e)
				return e
			}
			_ = ch.Close()
			repo := repositories.NuevoRepositorioPagosPostgres(bd)
			servicio := services.NuevoServicioPagos(repo)
			pub, e := messaging.NuevoPublicador(rabbit)
			if e != nil {
				return e
			}
			defer pub.Cerrar()
			pubReintentos, e := messaging.NuevoPublicador(rabbit)
			if e != nil {
				return e
			}
			defer pubReintentos.Cerrar()

			outbox := publishers.NuevoPublicadorOutbox(repo, pub, cfg.IntervaloOutbox)
			consComandos, e := consumers.NuevoConsumidorPagos(rabbit, servicio, pubReintentos, cfg.MaximoReintentos, repo)
			if e != nil {
				return e
			}
			defer consComandos.Cerrar()
			consEventos, e := consumers.NuevoConsumidorPagos(rabbit, servicio, pubReintentos, cfg.MaximoReintentos, repo)
			if e != nil {
				return e
			}
			defer consEventos.Cerrar()
			return recovery.Serve(ctx, func() { _ = rabbit.CloseDeadline(time.Now().Add(time.Second)) },
				[]func() bool{rabbit.IsClosed, pub.Cerrado, pubReintentos.Cerrado, consComandos.Cerrado, consEventos.Cerrado},
				func(ctx context.Context) error {
					return consComandos.Ejecutar(ctx, messaging.ColaComandos, "payment-service.comandos")
				},
				func(ctx context.Context) error {
					return consEventos.Ejecutar(ctx, messaging.ColaEventosCuenta, "payment-service.eventos-cuenta")
				},
				func(ctx context.Context) error { outbox.Ejecutar(ctx); return nil })

		})
	}()
	defer func() { cancelar(); <-mensajeriaFin }()

	app := fiber.New(fiber.Config{AppName: "payment-service", DisableStartupMessage: true})

	routes.RegistrarRutas(app)
	go func() {
		slog.Info("servicio de pagos iniciado", "puerto", cfg.PuertoHTTP, "entorno", cfg.Entorno)
		if e := app.Listen(":" + cfg.PuertoHTTP); e != nil && ctx.Err() == nil {
			cancelar()
		}
	}()
	<-ctx.Done()
	_ = app.Shutdown()
}
