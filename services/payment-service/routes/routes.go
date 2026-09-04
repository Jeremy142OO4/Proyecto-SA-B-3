package routes

import (
	"github.com/Proyecto-SA-B-3/payment-service/controllers"
	"github.com/gofiber/fiber/v2"
)

func RegistrarRutas(a *fiber.App) {
	a.Get("/salud", controllers.ConsultarSalud)
	a.Get("/health", controllers.ConsultarSalud)
}
