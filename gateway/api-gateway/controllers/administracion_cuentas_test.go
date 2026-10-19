package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

type adminPublisher struct {
	gestor  *responses.Gestor
	message events.SobreMensaje
}

func (p *adminPublisher) Publicar(ctx context.Context, m events.SobreMensaje) error {
	p.message = m
	p.gestor.Entregar(events.SobreMensaje{IDCorrelacion: m.IDCorrelacion, Tipo: "cuenta.bloqueada", Contenido: []byte(`{}`)})
	p.gestor.Entregar(events.SobreMensaje{IDCorrelacion: m.IDCorrelacion, Tipo: "cuenta.administracion.respondida", Contenido: []byte(`{"estado":200,"cuerpo":{"maxOperaciones":2,"ventanaMinutos":5}}`)})
	return nil
}
func TestMonitoreoAdministracionRBACYRespuestaFiltrada(t *testing.T) {
	gestor := responses.Nuevo()
	p := &adminPublisher{gestor: gestor}
	g := NuevoGateway(p, operations.NuevoStore(), gestor, time.Second)
	app := fiber.New()
	app.Put("/", middleware.Correlacion, middleware.Autenticacion("test-only"), middleware.AutorizarRoles("ADMIN"), g.ConfigurarMonitoreoCuentas)
	for _, role := range []string{"CLIENTE", "TELLER", "ADMIN"} {
		payload, _ := json.Marshal(map[string]any{"sub": uuid.NewString(), "role": role, "exp": time.Now().Add(time.Hour).Unix()})
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		signed := header + "." + base64.RawURLEncoding.EncodeToString(payload)
		mac := hmac.New(sha256.New, []byte("test-only"))
		mac.Write([]byte(signed))
		token := signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		req := httptest.NewRequest("PUT", "/", strings.NewReader(`{"maxOperaciones":2,"ventanaMinutos":5}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		result, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		result.Body.Close()
		expected := 403
		if role == "ADMIN" {
			expected = 200
		}
		if result.StatusCode != expected {
			t.Fatalf("role %s: %d", role, result.StatusCode)
		}
	}
	var payload struct {
		IDAdministrador uuid.UUID `json:"idAdministrador"`
	}
	_ = json.Unmarshal(p.message.Contenido, &payload)
	if payload.IDAdministrador == uuid.Nil {
		t.Fatal("administrator identity not propagated")
	}
}
