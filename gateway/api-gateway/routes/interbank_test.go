package routes

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/internal/bankrpc"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInterbankRoutesAuthorization(t *testing.T) {
	app := fiber.New()
	calls := 0
	var envelope map[string]any
	controllers.InterbankGateway{Enabled: true, Bank: "BANK03", Partner: "BANK14", InboundKey: "local-test-inbound-only", JWT: "test-only", Call: func(ctx context.Context, url, queue string, body any) (bankrpc.Response, error) {
		calls++
		b, _ := json.Marshal(body)
		_ = json.Unmarshal(b, &envelope)
		return bankrpc.Reply(202, map[string]string{"status": "PROCESSING"}), nil
	}}.Register(app)
	request := func(path, key, tp, role string) int {
		t.Helper()
		req := httptest.NewRequest("POST", path, strings.NewReader(`{"amount":"10.00"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", key)
		req.Header.Set("traceparent", tp)
		if role != "" {
			req.Header.Set("Authorization", "Bearer "+simulationToken(role))
		}
		r, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		return r.StatusCode
	}
	valid := "00-" + strings.Repeat("a", 32) + "-" + strings.Repeat("b", 16) + "-01"
	for _, tc := range []struct {
		key, tp string
		status  int
	}{{"", valid, 401}, {"wrong", valid, 401}, {"local-test-inbound-only", "", 400}, {"local-test-inbound-only", "00-" + strings.Repeat("0", 32) + "-" + strings.Repeat("b", 16) + "-01", 400}, {"local-test-inbound-only", valid, 202}} {
		if code := request("/interbank/v1/transfers", tc.key, tc.tp, ""); code != tc.status {
			t.Fatalf("status=%d want %d", code, tc.status)
		}
	}
	if calls != 1 || envelope["peer"] != "BANK14" {
		t.Fatalf("untrusted request reached service calls=%d payload=%v", calls, envelope)
	}
	for _, role := range []string{"", "ADMIN", "TELLER", "CLIENTE"} {
		expected := 403
		if role == "" {
			expected = 401
		}
		if role == "CLIENTE" {
			expected = 202
		}
		if code := request("/api/interbancarias/", "", "", role); code != expected {
			t.Fatalf("role=%s code=%d", role, code)
		}
	}
	if envelope["customer"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("missing JWT owner: %v", envelope)
	}
	if code := request("/interbank/v1/transfers/"+uuid.NewString()+"/result", "local-test-inbound-only", valid, ""); code != 202 {
		t.Fatal(code)
	}
	health, e := app.Test(httptest.NewRequest("GET", "/interbank/v1/health", nil))
	if e != nil || health.StatusCode != 200 {
		t.Fatal(e)
	}
	health.Body.Close()
	for _, tc := range []struct{ method, path, action string }{
		{"POST", "/interbank/v1/transfers/" + uuid.NewString() + "/cancel", "cancel"},
		{"POST", "/interbank/v1/reversals", "reversal"},
		{"GET", "/interbank/v1/reversals/" + uuid.NewString(), "reversal_get"},
		{"GET", "/interbank/v1/reconciliation?date=2026-10-18&page=2&pageSize=10", "reconciliation"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", "local-test-inbound-only")
		req.Header.Set("traceparent", valid)
		r, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 202 || envelope["action"] != tc.action {
			t.Fatalf("route %s: %d %v", tc.path, r.StatusCode, envelope)
		}
		if tc.action == "reconciliation" && (envelope["date"] != "2026-10-18" || envelope["page"] != float64(2) || envelope["pageSize"] != float64(10)) {
			t.Fatal("reconciliation query lost")
		}
	}
	disabled := fiber.New()
	controllers.InterbankGateway{Bank: "BANK03"}.Register(disabled)
	r, e := disabled.Test(httptest.NewRequest("POST", "/interbank/v1/transfers", strings.NewReader("{}")))
	if e != nil || r.StatusCode != 503 {
		t.Fatalf("disabled API: %v", e)
	}
	r.Body.Close()
}
