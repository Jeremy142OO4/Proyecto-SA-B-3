package main

import (
	"github.com/Proyecto-SA-B-3/api-gateway/internal/telemetry"
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
)

func main() {
	shutdownTelemetry := telemetry.Setup("api-gateway")
	defer shutdownTelemetry()
	cfg, err := config.Cargar()
	if err != nil {
		log.Fatal(err)
	}
	store := operations.NuevoStore()
	gestorRespuestas := responses.Nuevo()
	publicador, err := messaging.NuevoCliente(cfg.URLRabbitMQ, store, gestorRespuestas)
	if err != nil {
		log.Fatal(err)
	}
	defer publicador.Cerrar()
	app := fiber.New(fiber.Config{DisableStartupMessage: true, ErrorHandler: responderErrorHTTP})
	app.Use(telemetry.HTTP(), cors.New(cors.Config{AllowOrigins: cfg.OrigenesCORS, AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Correlation-ID, traceparent, tracestate"}))
	solicitante := messaging.NuevoSolicitante(publicador, gestorRespuestas, cfg.TiempoPublicacion)
	gateway := controllers.NuevoGateway(publicador, store, gestorRespuestas, cfg.TiempoPublicacion, solicitante)
	controladorClientes := controllers.NuevoControladorClientes(solicitante)
	controladorAuditoria := controllers.NuevoControladorAuditoria(solicitante)

	routes.Registrar(app, gateway, controladorClientes, controladorAuditoria, cfg.SecretoJWT, publicador.Listo)
	bankCode := os.Getenv("INTERBANK_BANK_CODE")
	if bankCode == "" {
		bankCode = "BANK03"
	}
	partnerCode := os.Getenv("INTERBANK_PARTNER_BANK_CODE")
	if partnerCode == "" {
		partnerCode = "BANK14"
	}
	controllers.InterbankGateway{Enabled: os.Getenv("INTERBANK_ENABLED") == "true", Bank: bankCode, Partner: partnerCode,
		InboundKey: os.Getenv("INTERBANK_API_KEY"), Rabbit: cfg.URLRabbitMQ, JWT: cfg.SecretoJWT, Ready: publicador.Listo}.Register(app)
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
