package controllers

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ControladorAuditoria struct{ cliente ClienteCustomer }

func NuevoControladorAuditoria(cliente ClienteCustomer) *ControladorAuditoria {
	return &ControladorAuditoria{cliente: cliente}
}

func (ca *ControladorAuditoria) Registros(c *fiber.Ctx) error {
	limite := c.QueryInt("limite", 50)
	if limite < 1 || limite > 100 {
		limite = 50
	}
	return ca.enviar(c, "/api/v1/audit/logs?limit="+url.QueryEscape(strconv.Itoa(limite)))
}

func (ca *ControladorAuditoria) Traza(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("idCorrelacion"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "idCorrelacion invalido")
	}
	return ca.enviar(c, "/api/v1/audit/trace/"+id.String())
}

func (ca *ControladorAuditoria) Notificaciones(c *fiber.Ctx) error {
	limite := c.QueryInt("limite", 50)
	if limite < 1 || limite > 100 {
		limite = 50
	}
	return ca.enviar(c, "/api/v1/audit/notifications?limit="+url.QueryEscape(strconv.Itoa(limite)))
}

func (ca *ControladorAuditoria) enviar(c *fiber.Ctx, ruta string) error {
	correlacion := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	respuesta, err := ca.cliente.Solicitar(c.UserContext(), http.MethodGet, ruta, c.Get("Authorization"), correlacion.String(), nil)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "notification-audit-service no disponible")
	}
	c.Type("json")
	return c.Status(respuesta.Estado).Send(respuesta.Cuerpo)
}
