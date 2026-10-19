package routes

import (
	"github.com/Proyecto-SA-B-3/transaction-service/controllers"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func Registrar(app *fiber.App, repo ...controllers.TransferLookup) {
	app.Get("/salud", controllers.Salud)
	app.Get("/metrics", adaptor.HTTPHandler(telemetry.MetricsHandler()))
	if len(repo) > 0 {
		app.Get("/observabilidad/transferencias/:idTransferencia", controllers.TrazaTransferencia(repo[0]))
	}
}
