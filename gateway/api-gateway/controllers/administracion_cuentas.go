package controllers

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
)

func (g *Gateway) AdministrarCuentas(c *fiber.Ctx) error {
	var id *uuid.UUID
	if value := c.Query("cuenta"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil || parsed == uuid.Nil {
			return fiber.NewError(422, "Cuenta invalida")
		}
		id = &parsed
	}
	return g.administrarCuenta(c, "cuenta.administracion.listado.solicitado", fiber.Map{"limite": limite(c), "desplazamiento": desplazamiento(c), "idCuenta": id})
}
func (g *Gateway) ConsultarMonitoreoCuentas(c *fiber.Ctx) error {
	return g.administrarCuenta(c, "cuenta.monitoreo.consulta.solicitada", fiber.Map{})
}
func (g *Gateway) ConfigurarMonitoreoCuentas(c *fiber.Ctx) error {
	var input struct {
		MaxOperaciones int `json:"maxOperaciones"`
		VentanaMinutos int `json:"ventanaMinutos"`
	}
	if c.BodyParser(&input) != nil {
		return fiber.NewError(400, "JSON invalido")
	}
	if input.MaxOperaciones < 1 || input.MaxOperaciones > 10000 || input.VentanaMinutos < 1 || input.VentanaMinutos > 1440 {
		return fiber.NewError(422, "Use entre 1 y 10000 operaciones y entre 1 y 1440 minutos")
	}
	return g.administrarCuenta(c, "cuenta.monitoreo.actualizacion.solicitada", fiber.Map{"maxOperaciones": input.MaxOperaciones, "ventanaMinutos": input.VentanaMinutos, "idAdministrador": idCliente(c)})
}
func (g *Gateway) BloquearCuenta(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("idCuenta"))
	if err != nil || id == uuid.Nil {
		return fiber.NewError(422, "Cuenta invalida")
	}
	var input struct {
		Motivo string `json:"motivo"`
	}
	if c.BodyParser(&input) != nil {
		return fiber.NewError(400, "JSON invalido")
	}
	input.Motivo = strings.TrimSpace(input.Motivo)
	if len(input.Motivo) < 5 || len(input.Motivo) > 300 {
		return fiber.NewError(422, "Indique un motivo entre 5 y 300 caracteres")
	}
	return g.administrarCuenta(c, "cuenta.bloqueo.solicitado", fiber.Map{"idCuenta": id, "motivo": input.Motivo, "idAdministrador": idCliente(c)})
}
func (g *Gateway) administrarCuenta(c *fiber.Ctx, tipo string, payload any) error {
	corr := c.Locals(middleware.CorrelationLocal).(uuid.UUID)
	ch, cancel := g.respuestas.RegistrarTipo(corr, "cuenta.administracion.respondida")
	defer cancel()
	message, err := events.Nuevo(tipo, corr, payload)
	if err != nil {
		return err
	}
	ctx, finish := context.WithTimeout(c.UserContext(), g.timeout)
	defer finish()
	if err = g.publicador.Publicar(ctx, message); err != nil {
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
		return fiber.NewError(504, "Account Service no respondio a tiempo; consulta el estado antes de reintentar")
	}
}
