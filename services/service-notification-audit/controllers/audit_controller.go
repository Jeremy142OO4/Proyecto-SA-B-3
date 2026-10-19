package controllers

import (
	"strconv"
	"strings"
	"time"

	"bank-usac/service-notification-audit/models"
	"bank-usac/service-notification-audit/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AuditController struct {
	svc services.AuditService
}

func NewAuditController(svc services.AuditService) *AuditController {
	return &AuditController{svc: svc}
}

func (ac *AuditController) GetRecent(c *fiber.Ctx) error {

	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil {
		return fiber.NewError(400, "limit invalido")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return fiber.NewError(400, "offset invalido")
	}
	filter := models.AuditFilter{Limit: limit, Offset: offset, TraceID: c.Query("traceId"), Service: c.Query("service"), Severity: models.EventSeverity(strings.ToUpper(c.Query("severity")))}
	if raw := c.Query("correlationId"); raw != "" {
		id, e := uuid.Parse(raw)
		if e != nil {
			return fiber.NewError(400, "correlationId invalido")
		}
		filter.CorrelationID = &id
	}
	for _, key := range []string{"from", "to"} {
		if raw := c.Query(key); raw != "" {
			value, e := time.Parse(time.RFC3339Nano, raw)
			if e != nil {
				return fiber.NewError(400, key+" invalido")
			}
			if key == "from" {
				filter.Since = &value
			} else {
				filter.Until = &value
			}
		}
	}
	if err := filter.Normalize(); err != nil {
		return fiber.NewError(400, err.Error())
	}
	logs, err := ac.svc.QueryAudits(c.UserContext(), filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al consultar auditoria"})
	}
	return c.JSON(logs)
}

func (ac *AuditController) GetByCorrelation(c *fiber.Ctx) error {
	corrStr := c.Params("correlationId")
	corrID, err := uuid.Parse(corrStr)
	if err != nil || corrID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "correlationId inválido"})
	}

	logs, err := ac.svc.GetAuditByCorrelation(c.UserContext(), corrID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error al consultar auditoría"})
	}
	return c.Status(fiber.StatusOK).JSON(logs)
}

func (ac *AuditController) GetNotifications(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	filter := models.NotificationFilter{
		Type: c.Query("type"), Severity: models.EventSeverity(c.Query("severity")),
		Limit:     limit,
		Recipient: c.Query("recipient"),
		Status:    models.NotificationStatus(c.Query("status")),
	}
	if correlation := c.Query("correlationId"); correlation != "" {
		correlationID, err := uuid.Parse(correlation)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "correlationId inválido"})
		}
		filter.CorrelationID = &correlationID
	}

	logs, err := ac.svc.GetNotifications(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error al consultar notificaciones"})
	}
	return c.Status(fiber.StatusOK).JSON(logs)
}
