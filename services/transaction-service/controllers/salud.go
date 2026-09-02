package controllers

import "github.com/gofiber/fiber/v2"

func Salud(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"estado": "OK", "servicio": "transaction-service"})
}
