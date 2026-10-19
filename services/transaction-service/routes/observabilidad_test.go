package routes

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/Proyecto-SA-B-3/transaction-service/repositories"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type lookup struct{ id uuid.UUID }

func (l lookup) Consultar(_ context.Context, id uuid.UUID) (models.Transferencia, error) {
	if id != l.id {
		return models.Transferencia{}, repositories.ErrNoEncontrada
	}
	return models.Transferencia{IDTransferencia: id, IDCorrelacion: id, Estado: models.Completada, TraceID: "0123456789abcdef0123456789abcdef"}, nil
}
func TestObservabilityEndpoints(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	stop := telemetry.Setup("transaction-service")
	defer stop()
	id := uuid.New()
	app := fiber.New()
	Registrar(app, lookup{id})
	for _, tc := range []struct {
		url    string
		status int
	}{{"/salud", 200}, {"/observabilidad/transferencias/invalid", 400}, {"/observabilidad/transferencias/" + uuid.New().String(), 404}, {"/observabilidad/transferencias/" + id.String(), 200}} {
		r, e := app.Test(httptest.NewRequest("GET", tc.url, nil))
		if e != nil {
			t.Fatal(e)
		}
		if r.StatusCode != tc.status {
			t.Fatalf("%s: %d", tc.url, r.StatusCode)
		}
		if strings.HasSuffix(tc.url, id.String()) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["estadoFase2"] != "APPROVED" || body["id_transaccion"] != id.String() || body["traceId"] == "" {
				t.Fatalf("missing trace identity: %v", body)
			}
		}
		r.Body.Close()
	}
	telemetry.TransferCommitted(context.Background(), id.String(), id.String(), "", models.Completada, 2*time.Second)
	telemetry.TransferCommitted(context.Background(), uuid.New().String(), id.String(), "", models.Compensada, 3*time.Second)
	r, e := app.Test(httptest.NewRequest("GET", "/metrics", nil))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	if r.StatusCode != 200 || !strings.Contains(string(b), "bank_transfer_flow_duration_seconds_bucket") || !strings.Contains(string(b), "bank_transfer_results_total") || !strings.Contains(string(b), `outcome="failure"`) {
		t.Fatalf("missing histogram/results: %s", b)
	}
	if strings.Contains(string(b), id.String()) {
		t.Fatal("transaction ID leaked into metric labels")
	}
}
