package controllers

import "github.com/gofiber/fiber/v2"

func ConsultarSalud(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"servicio": "payment-service", "estado": "disponible"})
}
