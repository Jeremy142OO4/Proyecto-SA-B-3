package routes

import (
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
)

func Registrar(app *fiber.App, g *controllers.Gateway, clientes *controllers.ControladorClientes, auditoria *controllers.ControladorAuditoria, secreto string, listo func() bool) {
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
	publico := app.Group("/api/clientes", middleware.Correlacion)
	publico.Get("/activacion", clientes.Activar)
	publico.Post("/login", clientes.Login)
	operaciones := app.Group("/api", middleware.Correlacion, middleware.Autenticacion(secreto))
	operaciones.Get("/operaciones/:id", middleware.AutorizarRoles("CLIENTE", "TELLER"), g.ConsultarOperacion)

	teller := app.Group("/api", middleware.Correlacion, middleware.Autenticacion(secreto), middleware.AutorizarRoles("TELLER"))
	teller.Post("/clientes/registro", clientes.Registrar)
	teller.Post("/cuentas", g.CrearCuenta)

	cliente := app.Group("/api", middleware.Correlacion, middleware.Autenticacion(secreto), middleware.AutorizarRoles("CLIENTE"))
	cliente.Get("/clientes/perfil", clientes.Perfil)
	cliente.Put("/clientes/perfil", clientes.ActualizarPerfil)
	cliente.Get("/cuentas", g.ListarCuentas)
	cliente.Get("/cuentas/:idCuenta", g.ConsultarCuenta)
	cliente.Get("/cuentas/:idCuenta/movimientos", g.ListarMovimientos)
	cliente.Get("/pagos", g.ListarPagos)
	cliente.Post("/pagos", g.CrearPago)
	cliente.Get("/pagos/:idPago", g.ConsultarPago)
	cliente.Get("/transferencias", g.ListarTransferencias)
	cliente.Post("/transferencias", g.Transferir)
	cliente.Get("/transferencias/:idTransferencia", g.ConsultarTransferencia)

	admin := app.Group("/api", middleware.Correlacion, middleware.Autenticacion(secreto), middleware.AutorizarRoles("ADMIN"))
	admin.Get("/auditoria/registros", auditoria.Registros)
	admin.Get("/auditoria/traza/:idCorrelacion", auditoria.Traza)
	admin.Get("/auditoria/notificaciones", auditoria.Notificaciones)
	admin.Get("/administracion/clientes", clientes.Listar)
	admin.Patch("/administracion/clientes/:idCliente/estado", clientes.CambiarEstado)
}
