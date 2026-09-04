package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Proyecto-SA-B-3/api-gateway/config"
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/messaging"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/Proyecto-SA-B-3/api-gateway/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg, err := config.Cargar()
	if err != nil {
		log.Fatal(err)
	}
	conexion, err := messaging.Conectar(cfg.URLRabbitMQ)
	if err != nil {
		log.Fatal(err)
	}
	defer conexion.Close()
	canal, err := conexion.Channel()
	if err != nil {
		log.Fatal(err)
	}
	if err = messaging.DeclararTopologia(canal); err != nil {
		log.Fatal(err)
	}
	canal.Close()
	publicador, err := messaging.NuevoPublicador(conexion)
	if err != nil {
		log.Fatal(err)
	}
	defer publicador.Cerrar()
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, e error) error {
		codigo := 500
		mensaje := "error interno"
		if fe, ok := e.(*fiber.Error); ok {
			codigo = fe.Code
			mensaje = fe.Message
		}
		return c.Status(codigo).JSON(fiber.Map{"error": mensaje, "correlationId": c.GetRespHeader("X-Correlation-ID")})
	}})
	app.Use(logger.New(logger.Config{Format: "{\"timestamp\":\"${time}\",\"level\":\"info\",\"service\":\"api-gateway\",\"method\":\"${method}\",\"path\":\"${path}\",\"status\":${status}}\n"}), cors.New(cors.Config{AllowOrigins: cfg.OrigenesCORS, AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Correlation-ID"}))
	store := operations.NuevoStore()
	gestorRespuestas := responses.Nuevo()
	if err = messaging.ConsumirRespuestas(conexion, store, gestorRespuestas); err != nil {
		log.Fatal(err)
	}
	gateway := controllers.NuevoGateway(publicador, store, gestorRespuestas, cfg.TiempoPublicacion)
	solicitante := messaging.NuevoSolicitante(publicador, gestorRespuestas, cfg.TiempoPublicacion)
	controladorClientes := controllers.NuevoControladorClientes(solicitante)
	controladorAuditoria := controllers.NuevoControladorAuditoria(solicitante)
	routes.Registrar(app, gateway, controladorClientes, controladorAuditoria, cfg.SecretoJWT, func() bool { return !conexion.IsClosed() })
	go func() {
		if err := app.Listen(":" + cfg.PuertoHTTP); err != nil {
			log.Printf("servidor detenido: %v", err)
		}
	}()
	senales := make(chan os.Signal, 1)
	signal.Notify(senales, syscall.SIGINT, syscall.SIGTERM)
	<-senales
	if err := app.Shutdown(); err != nil {
		log.Printf("cierre HTTP: %v", err)
	}
}
