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
	autenticadas := app.Group("/api", middleware.Correlacion, middleware.Autenticacion(secreto))
	autenticadas.Get("/operaciones/:id", middleware.AutorizarRoles("CLIENTE", "TELLER"), g.ConsultarOperacion)

	autenticadas.Post("/clientes/registro", middleware.AutorizarRoles("TELLER"), clientes.Registrar)
	autenticadas.Post("/cuentas", middleware.AutorizarRoles("TELLER"), g.CrearCuenta)

	autenticadas.Get("/clientes/perfil", middleware.AutorizarRoles("CLIENTE"), clientes.Perfil)
	autenticadas.Put("/clientes/perfil", middleware.AutorizarRoles("CLIENTE"), clientes.ActualizarPerfil)
	autenticadas.Get("/cuentas", middleware.AutorizarRoles("CLIENTE"), g.ListarCuentas)
	autenticadas.Get("/cuentas/:idCuenta", middleware.AutorizarRoles("CLIENTE"), g.ConsultarCuenta)
	autenticadas.Get("/cuentas/:idCuenta/movimientos", middleware.AutorizarRoles("CLIENTE"), g.ListarMovimientos)
	autenticadas.Get("/pagos", middleware.AutorizarRoles("CLIENTE"), g.ListarPagos)
	autenticadas.Post("/pagos", middleware.AutorizarRoles("CLIENTE"), g.CrearPago)
	autenticadas.Get("/pagos/:idPago", middleware.AutorizarRoles("CLIENTE"), g.ConsultarPago)
	autenticadas.Get("/transferencias", middleware.AutorizarRoles("CLIENTE"), g.ListarTransferencias)
	autenticadas.Post("/transferencias", middleware.AutorizarRoles("CLIENTE"), g.Transferir)
	autenticadas.Get("/transferencias/:idTransferencia", middleware.AutorizarRoles("CLIENTE"), g.ConsultarTransferencia)

	autenticadas.Get("/auditoria/registros", middleware.AutorizarRoles("ADMIN"), auditoria.Registros)
	autenticadas.Get("/auditoria/traza/:idCorrelacion", middleware.AutorizarRoles("ADMIN"), auditoria.Traza)
	autenticadas.Get("/auditoria/notificaciones", middleware.AutorizarRoles("ADMIN"), auditoria.Notificaciones)
	autenticadas.Get("/administracion/clientes", middleware.AutorizarRoles("ADMIN"), clientes.Listar)
	autenticadas.Patch("/administracion/clientes/:idCliente/estado", middleware.AutorizarRoles("ADMIN"), clientes.CambiarEstado)
}
