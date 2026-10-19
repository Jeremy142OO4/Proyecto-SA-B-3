package routes

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/api-gateway/controllers"
	"github.com/Proyecto-SA-B-3/api-gateway/events"
	"github.com/Proyecto-SA-B-3/api-gateway/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type auditRPC struct {
	calls   int
	kind    string
	payload map[string]any
	err     error
	invalid bool
}

func (r *auditRPC) Solicitar(_ context.Context, kind string, _ uuid.UUID, payload any) (messaging.RespuestaRPC, error) {
	r.calls++
	r.kind = kind
	b, _ := json.Marshal(payload)
	r.payload = map[string]any{}
	json.Unmarshal(b, &r.payload)
	if r.err != nil {
		return messaging.RespuestaRPC{}, r.err
	}
	if r.invalid {
		return messaging.RespuestaRPC{}, nil
	}
	status, body := 200, []byte(`[]`)
	if kind == "auditoria.slo.solicitada" {
		status = 202
		body = []byte(`{"estado":"REGISTRADA"}`)
	}
	return messaging.RespuestaRPC{Estado: status, Cuerpo: body}, nil
}
func TestAuditActualGatewayFiltersAndSLOAuthorization(t *testing.T) {
	rpc := &auditRPC{}
	controller := controllers.NuevoControladorAuditoria(rpc)
	app := fiber.New()
	Registrar(app, nil, nil, controller, "test-only", func() bool { return true })
	alert := `{"idAlerta":"error-budget","servicio":"payment-service","nivel":"ERROR","resumen":"SLO incumplido","inicio":"2026-10-19T01:00:00Z"}`
	request := func(method, path, role, body string) int {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if role != "" {
			req.Header.Set("Authorization", "Bearer "+simulationToken(role))
		}
		response, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	for _, role := range []string{"", "CLIENTE", "TELLER", "ADMIN"} {
		for _, tc := range []struct{ method, path, body string }{{"GET", "/api/auditoria/registros", ""}, {"GET", "/api/auditoria/trazas/" + strings.Repeat("a", 32), ""}, {"GET", "/api/auditoria/traza/" + uuid.NewString(), ""}, {"GET", "/api/auditoria/notificaciones?tipo=SLO_ALERT", ""}, {"POST", "/api/administracion/alertas/slo", alert}} {
			before := rpc.calls
			got := request(tc.method, tc.path, role, tc.body)
			want := 403
			if role == "" {
				want = 401
			}
			if role == "ADMIN" {
				want = 200
				if tc.method == "POST" {
					want = 202
				}
			}
			if got != want {
				t.Fatalf("%s %s %s: %d", role, tc.method, tc.path, got)
			}
			if role != "ADMIN" && rpc.calls != before {
				t.Fatal("unauthorized query reached broker")
			}
		}
	}
	corr := uuid.NewString()
	values := url.Values{"correlationId": {corr}, "traceId": {strings.Repeat("A", 32)}, "servicio": {"payment-service"}, "nivel": {"error"}, "limite": {"2"}, "desplazamiento": {"3"}, "desde": {"2026-10-18T00:00:00Z"}, "hasta": {"2026-10-19T00:00:00Z"}}
	if got := request("GET", "/api/auditoria/registros?"+values.Encode(), "ADMIN", ""); got != 200 {
		t.Fatal(got)
	}
	if rpc.kind != events.ComandoAuditoriaRegistros || rpc.payload["correlationId"] != corr || rpc.payload["traceId"] != strings.Repeat("a", 32) || rpc.payload["servicio"] != "payment-service" || rpc.payload["nivel"] != "ERROR" || rpc.payload["limite"] != float64(2) || rpc.payload["desplazamiento"] != float64(3) {
		t.Fatalf("filters lost in RPC: %v", rpc.payload)
	}
	if got := request("GET", "/api/auditoria/registros?idCorrelacion="+corr, "ADMIN", ""); got != 200 || rpc.payload["correlationId"] != corr {
		t.Fatal("legacy correlation filter broken")
	}
	for _, query := range []string{"traceId=invalid", "traceId=" + strings.Repeat("0", 32), "correlationId=no-uuid", "nivel=DEBUG", "limite=101", "desplazamiento=-1", "limite=oops", "desde=yesterday", "desde=2026-10-19T00:00:00Z&hasta=2026-10-18T00:00:00Z", "servicio=" + url.QueryEscape("payment' OR TRUE --"), "correlationId=" + corr + "&idCorrelacion=" + uuid.NewString()} {
		before := rpc.calls
		if got := request("GET", "/api/auditoria/registros?"+query, "ADMIN", ""); got != 400 || before != rpc.calls {
			t.Fatalf("invalid filter accepted: %s %d", query, got)
		}
	}
	for _, body := range []string{`{}`, `{"idAlerta":"x","servicio":"payment-service","nivel":"INFO","resumen":"x","inicio":"2026-10-19T00:00:00Z"}`} {
		if got := request("POST", "/api/administracion/alertas/slo", "ADMIN", body); got != 422 {
			t.Fatal("invalid alert accepted")
		}
	}
	rpc.err = errors.New("private connection string")
	if got := request("GET", "/api/auditoria/registros", "ADMIN", ""); got != 503 {
		t.Fatal("RPC unavailable must be 503")
	}
	rpc.err = nil
	rpc.invalid = true
	if got := request("GET", "/api/auditoria/registros", "ADMIN", ""); got != 502 {
		t.Fatal("invalid RPC response must be 502")
	}
}
