package controllers

import (
	"strconv"

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
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	logs, err := ac.svc.GetRecentAudits(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error al consultar auditoría"})
	}
	return c.Status(fiber.StatusOK).JSON(logs)
}

func (ac *AuditController) GetByCorrelation(c *fiber.Ctx) error {
	corrStr := c.Params("correlationId")
	corrID, err := uuid.Parse(corrStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "correlationId inválido"})
	}

	logs, err := ac.svc.GetAuditByCorrelation(c.Context(), corrID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error al consultar auditoría"})
	}
	return c.Status(fiber.StatusOK).JSON(logs)
}

func (ac *AuditController) GetNotifications(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	filter := models.NotificationFilter{
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

	logs, err := ac.svc.GetNotifications(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error al consultar notificaciones"})
	}
	return c.Status(fiber.StatusOK).JSON(logs)
}
