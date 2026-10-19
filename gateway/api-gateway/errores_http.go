package main

import (
	"github.com/Proyecto-SA-B-3/api-gateway/internal/telemetry"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log/slog"
)

// Operational failures are JSON logs collected by the infrastructure agent.
// This handler deliberately has no broker or audit-service dependency.
func responderErrorHTTP(c *fiber.Ctx, err error) error {
	status, message := 500, "error interno"
	if fe, ok := err.(*fiber.Error); ok {
		status = fe.Code
		message = fe.Message
	}
	correlation, ok := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	if !ok || correlation == uuid.Nil {
		correlation = uuid.New()
	}
	level := slog.LevelWarn
	if status >= 500 {
		level = slog.LevelError
	}
	route := "unmatched"
	if c.Route() != nil && c.Route().Path != "" {
		route = c.Route().Path
	}
	ctx := telemetry.WithCorrelation(c.UserContext(), correlation.String())
	slog.Log(ctx, level, "HTTP request failed", "statusCode", status, "errorCode", codigoHTTP(status), "httpMethod", c.Method(), "httpRoute", route)
	return c.Status(status).JSON(fiber.Map{"error": message, "correlationId": correlation.String()})
}

func codigoHTTP(status int) string {
	switch {
	case status >= 500:
		return "ERROR_TECNICO_HTTP"
	case status == 401:
		return "NO_AUTORIZADO"
	case status == 403:
		return "ACCESO_PROHIBIDO"
	case status == 404:
		return "NO_ENCONTRADO"
	case status == 409:
		return "CONFLICTO"
	case status == 429:
		return "LIMITE_EXCEDIDO"
	default:
		return "DATOS_INVALIDOS"
	}
}
