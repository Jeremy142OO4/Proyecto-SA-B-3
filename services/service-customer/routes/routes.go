package routes

import (
	"bank-usac/service-customer/config"
	"bank-usac/service-customer/controllers"
	"bank-usac/service-customer/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, cc *controllers.CustomerController, cfg *config.Config) {
	app.Use(middleware.CorrelationMiddleware())

	// Health Checks para Kubernetes
	app.Get("/health/live", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/health/ready", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	api := app.Group("/api/v1/customers")

	// Rutas Públicas
	api.Post("/register", cc.Register)
	api.Get("/activate", cc.Activate)
	api.Post("/login", cc.Login)

	// Rutas Protegidas (JWT)
	api.Get("/me", middleware.AuthMiddleware(cfg.JWTSecret), cc.GetProfile)
	api.Put("/me", middleware.AuthMiddleware(cfg.JWTSecret), cc.UpdateProfile)
	adminOnly := middleware.AuthMiddleware(cfg.JWTSecret, "ADMIN")
	api.Get("/", adminOnly, cc.ListCustomers)
	api.Patch("/:id/status", adminOnly, cc.UpdateCustomerStatus)
}
