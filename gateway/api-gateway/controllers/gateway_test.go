package controllers

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type publicadorFalso struct {
	mensaje   events.SobreMensaje
	respuestas *responses.Gestor
}

func (p *publicadorFalso) Publicar(_ context.Context, m events.SobreMensaje) error {
	p.mensaje = m
	if m.Tipo == events.ComandoConsultarCuenta && p.respuestas != nil {
		contenido, _ := json.Marshal(map[string]any{
			"idCliente": "11111111-1111-4111-8111-111111111111",
			"idCuenta":  "22222222-2222-4222-8222-222222222222",
		})
		p.respuestas.Entregar(events.SobreMensaje{
			IDMensaje: uuid.New(), IDCorrelacion: m.IDCorrelacion,
			Tipo: events.EventoCuentaConsultada, Contenido: contenido,
		})
	}
	return nil
}
func TestTransferenciaAceptada(t *testing.T) {
	gestor := responses.Nuevo()
	p := &publicadorFalso{respuestas: gestor}
	app := fiber.New()
	g := NuevoGateway(p, operations.NuevoStore(), gestor, time.Second)
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

func TestOperacionSoloEsVisibleParaSuPropietario(t *testing.T) {
	almacen := operations.NuevoStore()
	almacen.Crear(operations.Operacion{
		OperationID: "44444444-4444-4444-8444-444444444444",
		CustomerID:  "11111111-1111-4111-8111-111111111111",
	})
	g := NuevoGateway(&publicadorFalso{}, almacen, responses.Nuevo(), time.Second)
	app := fiber.New()
	app.Get("/:id", func(c *fiber.Ctx) error {
		c.Locals("customerId", "99999999-9999-4999-8999-999999999999")
		return g.ConsultarOperacion(c)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/44444444-4444-4444-8444-444444444444", nil))
	if err != nil || resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperaba 403: %v %d", err, resp.StatusCode)
	}
}

func TestConsultaPropagaCorrelationID(t *testing.T) {
	gestor := responses.Nuevo()
	p := &publicadorFalso{respuestas: gestor}
	g := NuevoGateway(p, operations.NuevoStore(), gestor, time.Second)
	app := fiber.New()
	app.Get("/:id", middleware.Correlacion, func(c *fiber.Ctx) error {
		c.Locals("customerId", "11111111-1111-4111-8111-111111111111")
		return g.ConsultarCuenta(c)
	})
	req := httptest.NewRequest("GET", "/22222222-2222-4222-8222-222222222222", nil)
	req.Header.Set("X-Correlation-ID", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperaba 200: %v %d", err, resp.StatusCode)
	}
	if p.mensaje.IDCorrelacion.String() != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("correlationId no propagado: %s", p.mensaje.IDCorrelacion)
	}
}
