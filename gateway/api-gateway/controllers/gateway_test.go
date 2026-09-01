package controllers

import (
	"context"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type publicadorFalso struct{ mensaje events.SobreMensaje }

func (p *publicadorFalso) Publicar(_ context.Context, m events.SobreMensaje) error {
	p.mensaje = m
	return nil
}
func TestTransferenciaAceptada(t *testing.T) {
	p := &publicadorFalso{}
	app := fiber.New()
	g := NuevoGateway(p, operations.NuevoStore(), responses.Nuevo(), time.Second)
	app.Post("/", middleware.Correlacion, func(c *fiber.Ctx) error {
		c.Locals("customerId", "11111111-1111-4111-8111-111111111111")
		return c.Next()
	}, g.Transferir)
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"idCuentaOrigen":"22222222-2222-4222-8222-222222222222","idCuentaDestino":"33333333-3333-4333-8333-333333333333","montoCentavos":1250}`))
	r.Header.Set("Content-Type", "application/json")
	resp, e := app.Test(r)
	if e != nil || resp.StatusCode != 202 {
		t.Fatalf("esperaba 202: %v %d", e, resp.StatusCode)
	}
	if p.mensaje.Tipo != events.ComandoTransferir {
		t.Fatalf("comando incorrecto %s", p.mensaje.Tipo)
	}
}
