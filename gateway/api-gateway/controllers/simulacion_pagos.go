package controllers

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"math"
)

func (g *Gateway) ConsultarSimulacionPagos(c *fiber.Ctx) error {
	return g.administrarSimulacionPagos(c, "pago.simulacion.consulta.solicitada", fiber.Map{})
}
func (g *Gateway) ConfigurarSimulacionPagos(c *fiber.Ctx) error {
	var p struct {
		Habilitada           bool    `json:"habilitada"`
		TasaFallosPorcentaje float64 `json:"tasaFallosPorcentaje"`
		LatenciaMs           int     `json:"latenciaMs"`
		TimeoutMs            int     `json:"timeoutMs"`
		ForzarTimeout        bool    `json:"forzarTimeout"`
	}
	if c.BodyParser(&p) != nil {
		return fiber.NewError(400, "JSON invalido")
	}
	if math.IsNaN(p.TasaFallosPorcentaje) || math.IsInf(p.TasaFallosPorcentaje, 0) || p.TasaFallosPorcentaje < 0 || p.TasaFallosPorcentaje > 100 || p.LatenciaMs < 0 || p.LatenciaMs > 30000 || p.TimeoutMs < 1 || p.TimeoutMs > 30000 {
		return fiber.NewError(422, "Tasa de 0 a 100, latencia de 0 a 30000 ms y timeout de 1 a 30000 ms")
	}
	return g.administrarSimulacionPagos(c, "pago.simulacion.actualizacion.solicitada", fiber.Map{"habilitada": p.Habilitada, "tasaFallosPorcentaje": p.TasaFallosPorcentaje, "latenciaMs": p.LatenciaMs, "timeoutMs": p.TimeoutMs, "forzarTimeout": p.ForzarTimeout, "idAdministrador": idCliente(c)})
}
func (g *Gateway) administrarSimulacionPagos(c *fiber.Ctx, tipo string, payload any) error {
	corr := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	ch, cancel := g.respuestas.RegistrarTipo(corr, "pago.simulacion.respondida")
	defer cancel()
	m, err := events.Nuevo(tipo, corr, payload)
	if err != nil {
		return err
	}
	ctx, finish := context.WithTimeout(c.UserContext(), g.timeout)
	defer finish()
	if err = g.publicador.Publicar(ctx, m); err != nil {
		return fiber.NewError(503, "Mensajeria no disponible")
	}
	select {
	case response := <-ch:
		var result struct {
			Estado int             `json:"estado"`
			Cuerpo json.RawMessage `json:"cuerpo"`
		}
		if json.Unmarshal(response.Contenido, &result) != nil || result.Estado < 200 || result.Estado > 599 {
			return fiber.NewError(502, "Respuesta administrativa invalida")
		}
		c.Type("json")
		return c.Status(result.Estado).Send(result.Cuerpo)
	case <-ctx.Done():
		return fiber.NewError(504, "Payment Service no respondio; consulta la configuracion antes de reintentar")
	}
}
