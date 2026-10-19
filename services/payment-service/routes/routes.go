package routes

import (
	"github.com/Proyecto-SA-B-3/payment-service/controllers"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func RegistrarRutas(a *fiber.App) {
	a.Get("/metrics", adaptor.HTTPHandler(telemetry.MetricsHandler()))
	a.Get("/salud", controllers.ConsultarSalud)
	a.Get("/health", controllers.ConsultarSalud)
}
