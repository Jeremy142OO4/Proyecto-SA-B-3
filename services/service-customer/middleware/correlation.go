package middleware

import (
	"bank-usac/service-customer/internal/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const CorrelationHeader = "X-Correlation-ID"

func CorrelationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		corrID := c.Get(CorrelationHeader)
		if id,err:=uuid.Parse(corrID);err!=nil||id==uuid.Nil {
			corrID = uuid.New().String()
		}
		c.SetUserContext(telemetry.WithCorrelation(c.UserContext(), corrID))
		c.Locals("correlationId", corrID)
		c.Set(CorrelationHeader, corrID)
		return c.Next()
	}
}

func GetCorrelationID(c *fiber.Ctx) uuid.UUID {
	val := c.Locals("correlationId")
	if str, ok := val.(string); ok {
		if id, err := uuid.Parse(str); err == nil {
			return id
		}
	}
	return uuid.New()
}
