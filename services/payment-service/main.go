package main

import (
	"context"
	"github.com/Proyecto-SA-B-3/payment-service/config"
	"github.com/Proyecto-SA-B-3/payment-service/database"
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
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
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
	rabbit, e := messaging.Conectar(cfg.URLRabbitMQ)
	if e != nil {
		slog.Error("RabbitMQ no disponible", "error", e)
		os.Exit(1)
	}
	defer rabbit.Close()
	ch, e := rabbit.Channel()
	if e != nil {
		os.Exit(1)
	}
	if e = messaging.DeclararTopologia(ch); e != nil {
		slog.Error("topologia de pagos invalida", "error", e)
		os.Exit(1)
	}
	_ = ch.Close()
	repo := repositories.NuevoRepositorioPagosPostgres(bd)
	servicio := services.NuevoServicioPagos(repo)
	pub, e := messaging.NuevoPublicador(rabbit)
	if e != nil {
		os.Exit(1)
	}
	defer pub.Cerrar()
	pubReintentos, e := messaging.NuevoPublicador(rabbit)
	if e != nil {
		os.Exit(1)
	}
	defer pubReintentos.Cerrar()
	outbox := publishers.NuevoPublicadorOutbox(repo, pub, cfg.IntervaloOutbox)
	go outbox.Ejecutar(ctx)
	consComandos, e := consumers.NuevoConsumidorPagos(rabbit, servicio, pubReintentos, cfg.MaximoReintentos)
	if e != nil {
		os.Exit(1)
	}
	defer consComandos.Cerrar()
	consEventos, e := consumers.NuevoConsumidorPagos(rabbit, servicio, pubReintentos, cfg.MaximoReintentos)
	if e != nil {
		os.Exit(1)
	}
	defer consEventos.Cerrar()
	go func() {
		if e := consComandos.Ejecutar(ctx, messaging.ColaComandos, "payment-service.comandos"); e != nil && ctx.Err() == nil {
			slog.Error("consumidor de pagos termino", "error", e)
			cancelar()
		}
	}()
	go func() {
		if e := consEventos.Ejecutar(ctx, messaging.ColaEventosCuenta, "payment-service.eventos-cuenta"); e != nil && ctx.Err() == nil {
			slog.Error("consumidor de cuenta para pagos termino", "error", e)
			cancelar()
		}
	}()
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
