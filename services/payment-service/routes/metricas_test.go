package routes

import (
	"context"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentMetricsEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	stop := telemetry.Setup("payment-service")
	defer stop()
	telemetry.ProveedorConfirmado(context.Background(), "EXITO", 0.1)
	telemetry.ProveedorConfirmado(context.Background(), "TIMEOUT", 0.2)
	telemetry.ProveedorConfirmado(context.Background(), "FALLO", 0.05)
	app := fiber.New()
	RegistrarRutas(app)
	for _, path := range []string{"/salud", "/health", "/metrics"} {
		r, e := app.Test(httptest.NewRequest("GET", path, nil))
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(r.Body)
		r.Body.Close()
		if r.StatusCode != 200 {
			t.Fatalf("%s: %d", path, r.StatusCode)
		}
		if path == "/metrics" {
			for _, want := range []string{"bank_payment_external_duration_seconds_bucket", "bank_payment_external_results_total", `outcome="success"`, `outcome="failure"`, `outcome="timeout"`} {
				if !strings.Contains(string(b), want) {
					t.Fatalf("metric missing %s: %s", want, b)
				}
			}
		}
	}
}
