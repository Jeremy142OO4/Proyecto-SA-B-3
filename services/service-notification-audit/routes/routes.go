package routes

import (
	"bank-usac/service-notification-audit/config"
	"bank-usac/service-notification-audit/controllers"
	"bank-usac/service-notification-audit/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, ac *controllers.AuditController, cfg *config.Config) {
	// Health Checks para Kubernetes
	app.Get("/health/live", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/health/ready", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	api := app.Group("/api/v1/audit")

	// Auditoría protegida (solo rol ADMIN)
	adminOnly := middleware.AuthMiddleware(cfg.JWTSecret, "ADMIN")
	api.Get("/logs", adminOnly, ac.GetRecent)
	api.Get("/trace/:correlationId", adminOnly, ac.GetByCorrelation)
	api.Get("/notifications", adminOnly, ac.GetNotifications)
}
