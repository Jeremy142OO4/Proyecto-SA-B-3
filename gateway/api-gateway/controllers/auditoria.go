package controllers

import (
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/messaging"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ControladorAuditoria struct{ solicitante SolicitanteRPC }

func NuevoControladorAuditoria(solicitante SolicitanteRPC) *ControladorAuditoria {
	return &ControladorAuditoria{solicitante: solicitante}
}

func (ca *ControladorAuditoria) Registros(c *fiber.Ctx) error {
	limite := c.QueryInt("limite", 50)
	if limite < 1 || limite > 100 {
		limite = 50
	}
	return ca.enviar(c, events.ComandoAuditoriaRegistros, map[string]int{"limite": limite})
}

func (ca *ControladorAuditoria) Traza(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("idCorrelacion"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "idCorrelacion invalido")
	}
	return ca.enviar(c, events.ComandoAuditoriaTraza, map[string]any{"idCorrelacion": id})
}

func (ca *ControladorAuditoria) Notificaciones(c *fiber.Ctx) error {
	limite := c.QueryInt("limite", 50)
	if limite < 1 || limite > 100 {
		limite = 50
	}
	return ca.enviar(c, events.ComandoAuditoriaNotificaciones, map[string]int{"limite": limite})
}

func (ca *ControladorAuditoria) enviar(c *fiber.Ctx, tipo string, contenido any) error {
	correlacion := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	respuesta, err := ca.solicitante.Solicitar(c.UserContext(), tipo, correlacion, contenido)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "notification-audit-service no disponible")
	}
	if respuesta.Estado == 0 {
		respuesta = messaging.RespuestaRPC{Estado: fiber.StatusBadGateway, Cuerpo: []byte(`{"error":"respuesta invalida"}`)}
	}
	c.Type("json")
	return c.Status(respuesta.Estado).Send(respuesta.Cuerpo)
}
