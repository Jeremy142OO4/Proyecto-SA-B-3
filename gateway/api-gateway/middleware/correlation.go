package middleware

import (
	"github.com/Proyecto-SA-B-3/api-gateway/internal/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const CorrelationLocal = "correlationId"

func Correlacion(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Get("X-Correlation-ID"))
	if err != nil || id==uuid.Nil {
		id = uuid.New()
	}
	c.SetUserContext(telemetry.WithCorrelation(c.UserContext(), id.String()))
	c.Locals(CorrelationLocal, id)
	c.Set("X-Correlation-ID", id.String())
	return c.Next()
}
