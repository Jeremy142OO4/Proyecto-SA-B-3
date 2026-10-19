package controllers

import (
	"encoding/json"
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
	filters, err := filtrosAuditoria(c)
	if err != nil {
		return err
	}
	return ca.enviar(c, events.ComandoAuditoriaRegistros, filters)
}

func (ca *ControladorAuditoria) Traza(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("idCorrelacion"))
	if err != nil || id == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "idCorrelacion invalido")
	}
	return ca.enviar(c, events.ComandoAuditoriaTraza, map[string]any{"idCorrelacion": id})
}

func (ca *ControladorAuditoria) Notificaciones(c *fiber.Ctx) error {
	limite := c.QueryInt("limite", 50)
	if limite < 1 || limite > 100 {
		limite = 50
	}

	consulta := map[string]any{
		"limite":       limite,
		"destinatario": c.Query("destinatario"),
		"tipo":         c.Query("tipo"), "nivel": c.Query("nivel"),
		"estado": c.Query("estado"),
	}
	if valor := c.Query("idCorrelacion"); valor != "" {
		idCorrelacion, err := uuid.Parse(valor)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "idCorrelacion invalido")
		}
		consulta["idCorrelacion"] = idCorrelacion
	}

	return ca.enviar(c, events.ComandoAuditoriaNotificaciones, consulta)
}

func (ca *ControladorAuditoria) enviar(c *fiber.Ctx, tipo string, contenido any) error {
	correlacion := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	respuesta, err := ca.solicitante.Solicitar(c.UserContext(), tipo, correlacion, contenido)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "notification-audit-service no disponible")
	}
	if respuesta.Estado < 200 || respuesta.Estado > 599 || !json.Valid(respuesta.Cuerpo) {
		respuesta = messaging.RespuestaRPC{Estado: fiber.StatusBadGateway, Cuerpo: []byte(`{"error":"respuesta invalida"}`)}
	}
	c.Type("json")
	return c.Status(respuesta.Estado).Send(respuesta.Cuerpo)
}
