package controllers

import "github.com/gofiber/fiber/v2"

func ConsultarSalud(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"servicio": "account-service",
		"estado":   "disponible",
	})
}
