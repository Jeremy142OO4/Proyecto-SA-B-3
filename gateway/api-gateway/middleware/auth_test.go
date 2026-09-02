package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAutorizarRolesPermiteRolConfigurado(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		c.Locals("role", "CLIENTE")
		return c.Next()
	}, AutorizarRoles("CLIENTE"), func(c *fiber.Ctx) error { return c.SendStatus(204) })
	respuesta, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil || respuesta.StatusCode != 204 {
		t.Fatalf("esperaba 204: %v %d", err, respuesta.StatusCode)
	}
}

func TestAutorizarRolesRechazaRolDiferente(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		c.Locals("role", "TELLER")
		return c.Next()
	}, AutorizarRoles("ADMIN"), func(c *fiber.Ctx) error { return c.SendStatus(204) })
	respuesta, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil || respuesta.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperaba 403: %v %d", err, respuesta.StatusCode)
	}
}
