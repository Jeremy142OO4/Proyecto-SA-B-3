package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const CorrelationLocal = "correlationId"

func Correlacion(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Get("X-Correlation-ID"))
	if err != nil {
		id = uuid.New()
	}
	c.Locals(CorrelationLocal, id)
	c.Set("X-Correlation-ID", id.String())
	return c.Next()
}
