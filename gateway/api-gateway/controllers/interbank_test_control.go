package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/internal/bankrpc"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
)

// TestControl cannot change a partner URL or forward a bank API key. It only
// controls a local mock in a separately provisioned sandbox bank instance.
type TestControl struct {
	Enabled                                                        bool
	Environment, BaseURL, Key, PartnerURL, JWT, GrafanaURL, Rabbit string
	CloudRelay                                                     bool
}

func TestControlFromEnv(jwt string) TestControl {
	return TestControl{Enabled: os.Getenv("INTERBANK_TEST_MODE") == "true",
		Environment: os.Getenv("ENTORNO"), BaseURL: os.Getenv("INTERBANK_MOCK_CONTROL_URL"),
		Key: os.Getenv("MOCK_CONTROL_KEY"), PartnerURL: os.Getenv("INTERBANK_PARTNER_BASE_URL"),
		JWT: jwt, GrafanaURL: os.Getenv("INTERBANK_TEST_GRAFANA_URL"), Rabbit: os.Getenv("URL_RABBITMQ"), CloudRelay: os.Getenv("INTERBANK_CLOUD_TEST_RELAY") == "true"}
}

func (g TestControl) ready() bool {
	u, e := url.Parse(g.BaseURL)
	return e == nil && g.Enabled && g.Environment == "interbank-sandbox" && len(g.Key) >= 16 &&
		u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1") &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Path == "" &&
		g.PartnerURL == g.BaseURL+"/interbank/v1"
}

func (g TestControl) Register(app *fiber.App) {
	group := app.Group("/api/administracion/pruebas-interbancarias", middleware.Autenticacion(g.JWT), middleware.AutorizarRoles("ADMIN"))
	// Only this fixed, private service can receive test traffic. The bank's
	// partner URL, API keys and broker are never changed by the relay.
	if g.CloudRelay && !g.Enabled {
		group.Use(func(c *fiber.Ctx) error {
			return g.forward(c, "http://interbank-scenarios:8080"+c.Path())
		})
	}
	group.Get("/", g.handle)
	group.Put("/", g.handle)
	group.Post("/depositos", g.deposit)
	group.Get("/depositos/:id", g.deposit)
	group.Get("/saldo", func(c *fiber.Ctx) error {
		if !g.ready() {
			return c.SendStatus(503)
		}
		return g.forward(c, "http://127.0.0.1:8080/api/administracion/cuentas?cuenta=00000000-0000-4000-8000-000000000005")
	})
}

func (g TestControl) forward(c *fiber.Ctx, target string) error {
	if len(c.Body()) > 512 {
		return c.SendStatus(400)
	}
	req, err := http.NewRequestWithContext(c.UserContext(), c.Method(), target, bytes.NewReader(c.Body()))
	if err != nil {
		return c.SendStatus(503)
	}
	req.Header.Set("Authorization", c.Get("Authorization"))
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"mensaje": "El laboratorio en GCP no está disponible. BANK14 permanece sin cambios."})
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(data) > 65536 {
		return c.SendStatus(503)
	}
	c.Set("Cache-Control", "no-store")
	c.Type("json")
	return c.Status(response.StatusCode).Send(data)
}

// The demo account is provisioned only in the independent sandbox database.
// No caller can substitute a real customer, origin account or partner URL.
func (g TestControl) deposit(c *fiber.Ctx) error {
	if !g.ready() {
		return c.Status(503).JSON(fiber.Map{"mensaje": "Depósitos de prueba deshabilitados fuera del sandbox"})
	}
	var input struct {
		ID     string `json:"id"`
		Amount string `json:"amount"`
	}
	action := "client_get"
	var body json.RawMessage
	if c.Method() == "POST" {
		decoder := json.NewDecoder(bytes.NewReader(c.Body()))
		decoder.DisallowUnknownFields()
		if len(c.Body()) > 512 || decoder.Decode(&input) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			return c.SendStatus(400)
		}
		if !demoAmount.MatchString(input.Amount) || input.Amount == "0.00" {
			return c.SendStatus(400)
		}
		payload := map[string]string{"interbankTransferId": input.ID, "originBankCode": "BANK03", "destinationBankCode": "BANK14", "originAccount": "900000000003", "destinationAccount": "TEST-BANK14-001", "amount": input.Amount, "currency": "GTQ", "description": "Prueba aislada desde panel admin"}
		body, _ = json.Marshal(payload)
		action = "send"
	} else {
		input.ID = c.Params("id")
	}
	id, e := uuid.Parse(input.ID)
	if e != nil || id == uuid.Nil || len(input.ID) != 36 {
		return c.SendStatus(400)
	}
	request := struct {
		Action   string          `json:"action"`
		ID       string          `json:"id,omitempty"`
		Key      string          `json:"key,omitempty"`
		Customer string          `json:"customer"`
		Body     json.RawMessage `json:"body,omitempty"`
	}{action, input.ID, input.ID, "00000000-0000-4000-8000-000000000003", body}
	ctx, cancel := context.WithTimeout(c.UserContext(), 12*time.Second)
	defer cancel()
	result, e := bankrpc.Call(ctx, g.Rabbit, "transacciones.interbancarias", request)
	if e != nil {
		return c.Status(503).JSON(fiber.Map{"mensaje": "Respuesta incierta: consulta el mismo ID, no envíes otra operación"})
	}
	c.Type("json")
	return c.Status(result.Status).Send(result.Body)
}

var demoAmount = regexp.MustCompile(`^(?:[0-9]{1,4}\.[0-9]{2}|10000\.00)$`)

func (g TestControl) handle(c *fiber.Ctx) error {
	if !g.ready() {
		if c.Method() == "GET" {
			return c.JSON(fiber.Map{"enabled": false, "target": "BANK14", "mensaje": "Controles deshabilitados: este banco no está en el entorno aislado de pruebas."})
		}
		return c.Status(503).JSON(fiber.Map{"mensaje": "No hay un entorno aislado de pruebas conectado"})
	}
	var payload []byte
	if c.Method() == "PUT" {
		var value struct {
			Scenario string `json:"scenario"`
		}
		decoder := json.NewDecoder(bytes.NewReader(c.Body()))
		decoder.DisallowUnknownFields()
		if len(c.Body()) > 512 || decoder.Decode(&value) != nil || decoder.Decode(&struct{}{}) != io.EOF {
			return c.Status(400).JSON(fiber.Map{"mensaje": "Escenario inválido"})
		}
		switch value.Scenario {
		case "accept", "reject", "unavailable", "no_callback", "delayed":
		default:
			return c.Status(400).JSON(fiber.Map{"mensaje": "Escenario inválido"})
		}
		payload, _ = json.Marshal(value)
	}
	req, e := http.NewRequestWithContext(c.UserContext(), c.Method(), g.BaseURL+"/__mock/control", bytes.NewReader(payload))
	if e != nil {
		return c.SendStatus(503)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Control-Key", g.Key)
	actor, _ := c.Locals("customerId").(string)
	req.Header.Set("X-Mock-Actor", actor)
	client := http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := client.Do(req)
	if e != nil {
		return c.Status(503).JSON(fiber.Map{"mensaje": "El mock de pruebas no está disponible; BANK14 no fue modificado"})
	}
	defer response.Body.Close()
	data, e := io.ReadAll(io.LimitReader(response.Body, 8193))
	if e != nil || len(data) > 8192 || response.StatusCode != 200 {
		return c.Status(503).JSON(fiber.Map{"mensaje": "No fue posible confirmar la configuración del mock"})
	}
	var status map[string]any
	if json.Unmarshal(data, &status) != nil || status["target"] != "MOCK" {
		return c.SendStatus(503)
	}
	status["enabled"] = true
	// Never return arbitrary URLs or credentials from the upstream response.
	result := fiber.Map{"enabled": true, "target": "MOCK", "environment": os.Getenv("INTERBANK_TEST_LOCATION"), "scenario": status["scenario"], "delaySeconds": status["delaySeconds"], "accounts": status["accounts"], "updatedAt": status["updatedAt"], "callbackConfigured": status["callbackConfigured"]}
	if u, e := url.Parse(g.GrafanaURL); e == nil && (u.Scheme == "https" || (u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost"))) && u.User == nil && !strings.ContainsAny(g.GrafanaURL, "\r\n") {
		result["grafanaUrl"] = g.GrafanaURL
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(result)
}
