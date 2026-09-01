package routes

import (
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
)

func Registrar(app *fiber.App, g *controllers.Gateway, secreto string, listo func() bool) {
	app.Get("/salud", func(c *fiber.Ctx) error {
		if !listo() {
			return c.Status(503).JSON(fiber.Map{"estado": "NO_DISPONIBLE"})
		}
		return c.JSON(fiber.Map{"estado": "OK", "servicio": "api-gateway"})
	})
	app.Get("/health/live", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"estado": "OK"}) })
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if !listo() {
			return c.SendStatus(503)
		}
		return c.JSON(fiber.Map{"estado": "OK"})
	})
	api := app.Group("/api", middleware.Correlacion, middleware.Autenticacion(secreto))
	api.Get("/cuentas", g.ListarCuentas)
	api.Post("/cuentas", g.CrearCuenta)
	api.Get("/cuentas/:idCuenta", g.ConsultarCuenta)
	api.Get("/cuentas/:idCuenta/movimientos", g.ListarMovimientos)
	api.Get("/pagos", g.ListarPagos)
	api.Post("/pagos", g.CrearPago)
	api.Get("/pagos/:idPago", g.ConsultarPago)
	api.Get("/transferencias", g.ListarTransferencias)
	api.Post("/transferencias", g.Transferir)
	api.Get("/transferencias/:idTransferencia", g.ConsultarTransferencia)
	api.Get("/operaciones/:id", g.ConsultarOperacion)
}
