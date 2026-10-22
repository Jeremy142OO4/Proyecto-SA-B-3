package controllers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BankCall func(context.Context, string, string, any) (bankrpc.Response, error)
type InterbankGateway struct {
	Enabled                                bool
	Bank, Partner, InboundKey, Rabbit, JWT string
	Call                                   BankCall
	Ready                                  func() bool
}

var tracePattern = regexp.MustCompile(`^00-([0-9a-f]{32})-([0-9a-f]{16})-[0-9a-f]{2}$`)

func bankError(c *fiber.Ctx, status int, code string) error {
	return c.Status(status).JSON(fiber.Map{"code": code, "message": code, "timestamp": time.Now().UTC()})
}
func (g InterbankGateway) Register(app *fiber.App) {
	if g.Call == nil {
		g.Call = bankrpc.Call
	}
	app.Get("/interbank/v1/health", func(c *fiber.Ctx) error {
		if !g.Enabled || len(g.InboundKey) < 16 || g.Ready != nil && !g.Ready() {
			return bankError(c, 503, "SERVICE_UNAVAILABLE")
		}
		return c.JSON(fiber.Map{"status": "UP", "bankCode": g.Bank})
	})
	peer := func(c *fiber.Ctx) error {
		if !g.Enabled {
			return bankError(c, 503, "SERVICE_UNAVAILABLE")
		}
		if len(g.InboundKey) < 16 || subtle.ConstantTimeCompare([]byte(c.Get("X-Api-Key")), []byte(g.InboundKey)) != 1 {
			return bankError(c, 401, "UNAUTHORIZED")
		}
		parts := tracePattern.FindStringSubmatch(c.Get("traceparent"))
		if len(parts) != 3 || parts[1] == strings.Repeat("0", 32) || parts[2] == strings.Repeat("0", 16) {
			return bankError(c, 400, "VALIDATION_ERROR")
		}
		return c.Next()
	}
	group := app.Group("/interbank/v1", peer)
	group.Post("/transfers", g.forward("receive", true))
	group.Get("/transfers/:id", g.forward("get", true))
	group.Post("/transfers/:id/result", g.forward("result", true))
	group.Post("/transfers/:id/cancel", g.forward("cancel", true))
	group.Post("/reversals", g.forward("reversal", true))
	group.Get("/reversals/:id", g.forward("reversal_get", true))
	group.Get("/reconciliation", g.forward("reconciliation", true))
	client := app.Group("/api/interbancarias", middleware.Correlacion, middleware.Autenticacion(g.JWT), middleware.AutorizarRoles("CLIENTE"))
	client.Get("/configuracion", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"bankCode": g.Bank, "partnerBankCode": g.Partner, "enabled": g.Enabled, "simulationEnabled": os.Getenv("ENTORNO") == "dev", "maxAmount": "10000.00"})
	})
	client.Get("/", g.forward("client_list", false))
	client.Get("/historial", g.forward("client_list", false))
	client.Post("/:id/restablecer", g.forward("resume", false))
	client.Post("/escenarios", g.forward("simulate", false))
	client.Post("/", g.forward("send", false))
	client.Get("/:id", g.forward("client_get", false))
}
func (g InterbankGateway) forward(action string, peer bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !g.Enabled {
			return bankError(c, 503, "SERVICE_UNAVAILABLE")
		}
		if len(c.Body()) > 65536 {
			return bankError(c, 400, "VALIDATION_ERROR")
		}
		id := c.Params("id")
		if id != "" {
			if _, e := uuid.Parse(id); e != nil {
				return bankError(c, 400, "VALIDATION_ERROR")
			}
		}
		scenario := ""
		if action == "simulate" {
			if os.Getenv("ENTORNO") != "dev" && os.Getenv("ENTORNO") != "desarrollo" {
				return bankError(c, 403, "SIMULATION_DISABLED")
			}
			var data map[string]json.RawMessage
			if json.Unmarshal(c.Body(), &data) != nil || json.Unmarshal(data["scenario"], &scenario) != nil {
				return bankError(c, 400, "VALIDATION_ERROR")
			}
			delete(data, "scenario")
			clean, _ := json.Marshal(data)
			c.Request().SetBody(clean)
		}
		body := json.RawMessage(append([]byte(nil), c.Body()...))
		if len(body) > 0 && !json.Valid(body) {
			return bankError(c, 400, "VALIDATION_ERROR")
		}
		request := struct {
			Date     string          `json:"date,omitempty"`
			Page     int             `json:"page,omitempty"`
			PageSize int             `json:"pageSize,omitempty"`
			Scenario string          `json:"scenario,omitempty"`
			Action   string          `json:"action"`
			ID       string          `json:"id,omitempty"`
			Key      string          `json:"key,omitempty"`
			Customer string          `json:"customer,omitempty"`
			Peer     string          `json:"peer,omitempty"`
			Body     json.RawMessage `json:"body,omitempty"`
		}{Scenario: scenario, Action: action, ID: id, Key: c.Get("Idempotency-Key"), Body: body}
		if action == "reconciliation" {
			request.Date = c.Query("date")
			request.Page = 1
			request.PageSize = 100
			for _, field := range []struct {
				name   string
				target *int
			}{{"page", &request.Page}, {"pageSize", &request.PageSize}} {
				if raw := c.Query(field.name); raw != "" {
					n, e := strconv.Atoi(raw)
					if e != nil || n < 1 {
						return bankError(c, 400, "VALIDATION_ERROR")
					}
					*field.target = n
				}
			}
		}
		if peer {
			request.Peer = g.Partner
		} else {
			request.Customer, _ = c.Locals("customerId").(string)
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 12*time.Second)
		defer cancel()
		// telemetry.HTTP has already extracted the incoming W3C context.
		c.Set("X-Trace-ID", trace.SpanContextFromContext(ctx).TraceID().String())
		result, e := g.Call(ctx, g.Rabbit, "transacciones.interbancarias", request)
		if e != nil {
			return bankError(c, 503, "SERVICE_UNAVAILABLE")
		}
		if result.Status < 200 || result.Status > 599 || !json.Valid(result.Body) {
			return bankError(c, 503, "SERVICE_UNAVAILABLE")
		}
		c.Type("json")
		return c.Status(result.Status).Send(result.Body)
	}
}
