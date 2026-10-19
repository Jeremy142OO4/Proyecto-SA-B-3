package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
	"time"
)

func (ca *ControladorAuditoria) RegistrarAlertaSLO(c *fiber.Ctx) error {
	var req struct {
		AlertID       string     `json:"idAlerta"`
		Service       string     `json:"servicio"`
		Severity      string     `json:"nivel"`
		Summary       string     `json:"resumen"`
		StartedAt     time.Time  `json:"inicio"`
		CorrelationID *uuid.UUID `json:"correlationId,omitempty"`
		TraceID       string     `json:"traceId,omitempty"`
	}
	if c.BodyParser(&req) != nil {
		return fiber.NewError(400, "JSON de alerta SLO invalido")
	}
	req.AlertID = strings.TrimSpace(req.AlertID)
	req.Service = strings.TrimSpace(req.Service)
	req.Summary = strings.TrimSpace(req.Summary)
	req.Severity = strings.ToUpper(strings.TrimSpace(req.Severity))
	req.TraceID = strings.ToLower(strings.TrimSpace(req.TraceID))
	if req.Severity == "" {
		req.Severity = "WARNING"
	}
	if len(req.AlertID) < 1 || len(req.AlertID) > 200 || !auditServicePattern.MatchString(req.Service) || len(req.Summary) < 1 || len(req.Summary) > 1000 || req.StartedAt.IsZero() {
		return fiber.NewError(422, "Indique idAlerta, servicio, resumen e inicio RFC3339 validos")
	}
	if req.Severity != "WARNING" && req.Severity != "ERROR" {
		return fiber.NewError(422, "Una alerta SLO debe ser WARNING o ERROR")
	}
	if req.TraceID != "" && (!auditTracePattern.MatchString(req.TraceID) || req.TraceID == strings.Repeat("0", 32)) {
		return fiber.NewError(422, "traceId invalido")
	}
	if req.CorrelationID != nil && *req.CorrelationID == uuid.Nil {
		return fiber.NewError(422, "correlationId invalido")
	}
	return ca.enviar(c, "auditoria.slo.solicitada", req)
}
