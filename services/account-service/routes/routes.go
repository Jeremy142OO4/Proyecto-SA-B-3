package routes

import (
	"github.com/Proyecto-SA-B-3/account-service/controllers"
	"github.com/gofiber/fiber/v2"
)

func RegistrarRutas(aplicacion *fiber.App) {
	aplicacion.Get("/salud", controllers.ConsultarSalud)
	aplicacion.Get("/health", controllers.ConsultarSalud)
}
