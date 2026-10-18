package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Proyecto-SA-B-3/api-gateway/config"
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/messaging"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/Proyecto-SA-B-3/api-gateway/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
)

func main() {
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
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, e error) error {
		codigo := 500
		mensaje := "error interno"
		if fe, ok := e.(*fiber.Error); ok {
			codigo = fe.Code
			mensaje = fe.Message
		}
		correlacion := uuid.New()
		if id, ok := c.Locals(middleware.CorrelationLocal).(uuid.UUID); ok && id != uuid.Nil {
			correlacion = id
		}
		if err := publicarErrorHTTP(publicador, c, correlacion, codigo, mensaje); err != nil {
			log.Printf("no se pudo registrar error HTTP en auditoría: %v", err)
		}
		return c.Status(codigo).JSON(fiber.Map{"error": mensaje, "correlationId": correlacion.String()})
	}})
	app.Use(logger.New(logger.Config{Format: "{\"timestamp\":\"${time}\",\"level\":\"info\",\"service\":\"api-gateway\",\"method\":\"${method}\",\"path\":\"${path}\",\"status\":${status}}\n"}), cors.New(cors.Config{AllowOrigins: cfg.OrigenesCORS, AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Correlation-ID"}))
	solicitante := messaging.NuevoSolicitante(publicador, gestorRespuestas, cfg.TiempoPublicacion)
	gateway := controllers.NuevoGateway(publicador, store, gestorRespuestas, cfg.TiempoPublicacion, solicitante)
	controladorClientes := controllers.NuevoControladorClientes(solicitante)
	controladorAuditoria := controllers.NuevoControladorAuditoria(solicitante)
	routes.Registrar(app, gateway, controladorClientes, controladorAuditoria, cfg.SecretoJWT, publicador.Listo)
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

func publicarErrorHTTP(publicador *messaging.Publicador, c *fiber.Ctx, correlacion uuid.UUID, codigo int, mensaje string) error {
	contenido, err := json.Marshal(events.ErrorHTTP{
		StatusCode:    codigo,
		Codigo:        codigoHTTP(codigo),
		Mensaje:       mensaje,
		Metodo:        c.Method(),
		Ruta:          c.Path(),
		CustomerID:    valorLocal(c, "customerId"),
		Rol:           valorLocal(c, "role"),
		CorrelationID: correlacion,
	})
	if err != nil {
		return err
	}
	evento := events.SobreMensaje{
		IDMensaje:     uuid.New(),
		IDCorrelacion: correlacion,
		Tipo:          events.EventoErrorHTTP,
		Version:       1,
		OcurridoEn:    time.Now().UTC(),
		Productor:     "api-gateway",
		Contenido:     contenido,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	return publicador.PublicarEvento(ctx, evento)
}

func valorLocal(c *fiber.Ctx, key string) string {
	value, _ := c.Locals(key).(string)
	return value
}

func codigoHTTP(status int) string {
	switch {
	case status >= 500:
		return "ERROR_TECNICO_HTTP"
	case status == 401:
		return "NO_AUTORIZADO"
	case status == 403:
		return "ACCESO_PROHIBIDO"
	case status == 404:
		return "NO_ENCONTRADO"
	case status == 409:
		return "CONFLICTO"
	case status == 429:
		return "LIMITE_EXCEDIDO"
	default:
		return "DATOS_INVALIDOS"
	}
}
