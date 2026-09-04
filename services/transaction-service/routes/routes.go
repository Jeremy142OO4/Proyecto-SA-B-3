package routes

import (
	"github.com/Proyecto-SA-B-3/transaction-service/controllers"
	"github.com/gofiber/fiber/v2"
)

func Registrar(app *fiber.App) { app.Get("/salud", controllers.Salud) }
