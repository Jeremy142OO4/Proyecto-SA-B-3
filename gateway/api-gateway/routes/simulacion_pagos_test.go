package routes

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/operations"
	"github.com/Proyecto-SA-B-3/api-gateway/responses"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type simulationPublisher struct {
	gestor  *responses.Gestor
	calls   int
	message events.SobreMensaje
	mode    string
}

func (p *simulationPublisher) Publicar(_ context.Context, m events.SobreMensaje) error {
	p.calls++
	p.message = m
	p.gestor.Entregar(events.SobreMensaje{IDCorrelacion: m.IDCorrelacion, Tipo: "pago.compensando", Contenido: []byte(`{}`)})
	if p.mode == "timeout" {
		return nil
	}
	body := []byte(`{"estado":200,"cuerpo":{"habilitada":true,"version":1}}`)
	if p.mode == "invalid" {
		body = []byte(`{"estado":0}`)
	}
	p.gestor.Entregar(events.SobreMensaje{IDCorrelacion: m.IDCorrelacion, Tipo: "pago.simulacion.respondida", Contenido: body})
	return nil
}
func simulationToken(role string) string {
	b, _ := json.Marshal(map[string]any{"sub": "11111111-1111-4111-8111-111111111111", "role": role, "exp": time.Now().Add(time.Hour).Unix()})
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signed := header + "." + base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte("test-only"))
	mac.Write([]byte(signed))
	return signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func TestSimulationActualRoutesAuthorizationAndValidation(t *testing.T) {
	gestor := responses.Nuevo()
	publisher := &simulationPublisher{gestor: gestor}
	g := controllers.NuevoGateway(publisher, operations.NuevoStore(), gestor, 10*time.Millisecond)
	app := fiber.New()
	Registrar(app, g, &controllers.ControladorClientes{}, &controllers.ControladorAuditoria{}, "test-only", func() bool { return true })
	valid := `{"habilitada":true,"tasaFallosPorcentaje":37.5,"latenciaMs":10,"timeoutMs":1000,"forzarTimeout":false,"idAdministrador":"99999999-9999-4999-8999-999999999999"}`
	for _, role := range []string{"", "CLIENTE", "TELLER", "ADMIN"} {
		for _, method := range []string{"GET", "PUT"} {
			req := httptest.NewRequest(method, "/api/administracion/pagos/simulacion", strings.NewReader(valid))
			req.Header.Set("Content-Type", "application/json")
			if role != "" {
				req.Header.Set("Authorization", "Bearer "+simulationToken(role))
			}
			before := publisher.calls
			r, e := app.Test(req)
			if e != nil {
				t.Fatal(e)
			}
			r.Body.Close()
			expected := 403
			if role == "" {
				expected = 401
			}
			if role == "ADMIN" {
				expected = 200
			}
			if r.StatusCode != expected {
				t.Fatalf("%s %s: %d", role, method, r.StatusCode)
			}
			if role != "ADMIN" && publisher.calls != before {
				t.Fatal("unauthorized request reached broker")
			}
		}
	}
	var payload struct {
		IDAdministrador uuid.UUID `json:"idAdministrador"`
	}
	json.Unmarshal(publisher.message.Contenido, &payload)
	if payload.IDAdministrador.String() != "11111111-1111-4111-8111-111111111111" {
		t.Fatal("caller spoofed administrator")
	}
	for _, body := range []string{`{}`, `{"timeoutMs":0}`, `{"timeoutMs":100,"tasaFallosPorcentaje":101}`, `{"timeoutMs":100,"latenciaMs":-1}`, `{"timeoutMs":100,"latenciaMs":30001}`} {
		req := httptest.NewRequest("PUT", "/api/administracion/pagos/simulacion", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+simulationToken("ADMIN"))
		r, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 422 {
			t.Fatalf("invalid config accepted: %s %d", body, r.StatusCode)
		}
	}
	for _, tc := range []struct {
		mode   string
		status int
	}{{"invalid", 502}, {"timeout", 504}} {
		publisher.mode = tc.mode
		req := httptest.NewRequest("GET", "/api/administracion/pagos/simulacion", nil)
		req.Header.Set("Authorization", "Bearer "+simulationToken("ADMIN"))
		r, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != tc.status {
			t.Fatalf("%s: %d", tc.mode, r.StatusCode)
		}
	}
}
