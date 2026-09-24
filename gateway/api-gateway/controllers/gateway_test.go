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
	mensaje    events.SobreMensaje
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
	if m.Tipo == events.ComandoHistorialTransferencias && p.respuestas != nil {
		contenido, _ := json.Marshal(map[string]any{
			"idCliente":      "11111111-1111-4111-8111-111111111111",
			"transferencias": []any{},
		})
		p.respuestas.Entregar(events.SobreMensaje{
			IDMensaje: uuid.New(), IDCorrelacion: m.IDCorrelacion,
			Tipo: events.EventoHistorialTransferencias, Contenido: contenido,
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
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"idCuentaOrigen":"22222222-2222-4222-8222-222222222222","idCuentaDestino":"33333333-3333-4333-8333-333333333333","tipoCuentaDestino":"AHORRO","montoCentavos":1250}`))
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
	app.Get("/:idCuenta", middleware.Correlacion, func(c *fiber.Ctx) error {
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

func TestHistorialPropagaFiltros(t *testing.T) {
	gestor := responses.Nuevo()
	publicador := &publicadorFalso{respuestas: gestor}
	gateway := NuevoGateway(publicador, operations.NuevoStore(), gestor, time.Second)
	app := fiber.New()
	app.Get("/", middleware.Correlacion, func(c *fiber.Ctx) error {
		c.Locals("customerId", "11111111-1111-4111-8111-111111111111")
		return gateway.ListarTransferencias(c)
	})

	peticion := httptest.NewRequest("GET", "/?idCuenta=22222222-2222-4222-8222-222222222222&fechaDesde=2026-09-01&fechaHasta=2026-09-10&estado=COMPLETADA", nil)
	respuesta, err := app.Test(peticion)
	if err != nil || respuesta.StatusCode != fiber.StatusOK {
		t.Fatalf("esperaba 200: %v %d", err, respuesta.StatusCode)
	}

	var filtros events.SolicitudHistorial
	if err = json.Unmarshal(publicador.mensaje.Contenido, &filtros); err != nil {
		t.Fatalf("decodificar filtros publicados: %v", err)
	}
	if filtros.IDCuenta == nil || filtros.IDCuenta.String() != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("cuenta no propagada: %v", filtros.IDCuenta)
	}
	if filtros.FechaDesde != "2026-09-01" || filtros.FechaHasta != "2026-09-10" || filtros.Estado != "COMPLETADA" {
		t.Fatalf("filtros no propagados: %+v", filtros)
	}
}

func TestPagoExternoPropagaResultadoSimulado(t *testing.T) {
	gestor := responses.Nuevo()
	publicador := &publicadorFalso{respuestas: gestor}
	gateway := NuevoGateway(publicador, operations.NuevoStore(), gestor, time.Second)
	app := fiber.New()
	app.Post("/", middleware.Correlacion, func(c *fiber.Ctx) error {
		c.Locals("customerId", "11111111-1111-4111-8111-111111111111")
		return gateway.CrearPago(c)
	})

	peticion := httptest.NewRequest("POST", "/", strings.NewReader(`{"idCuentaOrigen":"22222222-2222-4222-8222-222222222222","beneficiario":"Proveedor externo","concepto":"Prueba","montoCentavos":1000,"tipoPago":"EXTERNO","resultadoSimulado":"TIMEOUT"}`))
	peticion.Header.Set("Content-Type", "application/json")
	respuesta, err := app.Test(peticion)
	if err != nil || respuesta.StatusCode != fiber.StatusAccepted {
		t.Fatalf("esperaba 202: %v %d", err, respuesta.StatusCode)
	}

	var solicitud events.SolicitudPago
	if err = json.Unmarshal(publicador.mensaje.Contenido, &solicitud); err != nil {
		t.Fatalf("decodificar pago publicado: %v", err)
	}
	if solicitud.ResultadoSimulado != "TIMEOUT" {
		t.Fatalf("resultado no propagado: %+v", solicitud)
	}
}
