package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func tracing(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()
	old := otel.GetTracerProvider()
	ex := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(ex))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()); otel.SetTracerProvider(old) })
	return tp, ex
}
func TestOutboxRestartAndRetryPreserveTrace(t *testing.T) {
	tp, ex := tracing(t)
	ctx, root := tp.Tracer("test").Start(context.Background(), "request")
	original := root.SpanContext().TraceID()
	body := []byte(`{"idMensaje":"00000000-0000-4000-8000-000000000001","idCorrelacion":"00000000-0000-4000-8000-000000000002"}`)
	_, h, done := Publish(ctx, amqp.Table{"x-intentos": int32(1)}, body, "cuenta.debito.solicitado")
	done(nil)
	consumer, end := Consume(context.Background(), amqp.Delivery{Headers: h, Body: body, RoutingKey: "cuenta.debito.solicitado"})
	saved := Serialize(consumer)
	end(nil)
	root.End()
	// Simulate another process: only the durable string survives, no original context.
	restored := Restore(context.Background(), saved)
	_, retry, finish := Publish(restored, amqp.Table{"x-intentos": int32(2)}, body, "cuenta.compensacion.solicitada")
	finish(nil)
	next, closeNext := Consume(context.Background(), amqp.Delivery{Headers: retry, Body: body, Redelivered: true, RoutingKey: "cuenta.compensacion.solicitada"})
	closeNext(nil)
	if trace.SpanContextFromContext(next).TraceID() != original {
		t.Fatal("trace split across Outbox/retry")
	}
	if h["x-intentos"] != int32(1) || retry["x-intentos"] != int32(2) {
		t.Fatal("retry headers mutated")
	}
	for _, span := range ex.GetSpans() {
		if span.SpanContext.TraceID() != original {
			t.Fatal("unexpected trace root")
		}
	}
	if len(ex.GetSpans()) != 5 {
		t.Fatalf("expected request and four messaging spans, got %d", len(ex.GetSpans()))
	}
}
func TestInvalidContextAndNoBaggage(t *testing.T) {
	tracing(t)
	ctx, done := Consume(context.Background(), amqp.Delivery{Headers: amqp.Table{"traceparent": "bad", "baggage": "email=private@example.com"}, Body: []byte("{}")})
	defer done(nil)
	if !trace.SpanContextFromContext(ctx).IsValid() {
		t.Fatal("invalid parent did not create root")
	}
	if Serialize(ctx) == "" {
		t.Fatal("missing context")
	}
	if strings.Contains(Serialize(ctx), "private") {
		t.Fatal("baggage leaked")
	}
	if Serialize(context.Background()) != "" {
		t.Fatal("fabricated context")
	}
}
func TestHTTPStatusMetricsAndRouteCardinality(t *testing.T) {
	_, ex := tracing(t)
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	old := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer func() { _ = mp.Shutdown(context.Background()); otel.SetMeterProvider(old) }()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(HTTP())
	app.Get("/accounts/:id", func(c *fiber.Ctx) error { return fiber.ErrUnauthorized })
	app.Get("/broken", func(c *fiber.Ctx) error { return fiber.ErrInternalServerError })
	app.Get("/health/ready", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	for _, p := range []string{"/accounts/private-id", "/broken", "/health/ready"} {
		res, err := app.Test(httptest.NewRequest("GET", p, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}
	spans := ex.GetSpans()
	if len(spans) != 2 || spans[0].Name != "GET /accounts/:id" {
		t.Fatalf("unexpected spans: %+v", spans)
	}
	if spans[0].Status.Code == codes.Error || spans[1].Status.Code != codes.Error {
		t.Fatal("HTTP status misclassified")
	}
	var data metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &data); err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
				for _, p := range sum.DataPoints {
					total += p.Value
					if strings.Contains(p.Attributes.Encoded(nil), "private-id") {
						t.Fatal("raw path in labels")
					}
				}
			}
		}
	}
	if total != 2 {
		t.Fatalf("expected two business responses, got %d", total)
	}
}
func TestJSONLogsRedactSecretsAndIncludeContext(t *testing.T) {
	tp, _ := tracing(t)
	var out bytes.Buffer
	oldLogger := slog.Default()
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	defer func() { slog.SetDefault(oldLogger); log.SetOutput(oldWriter); log.SetFlags(oldFlags) }()
	ConfigureLogger(&out, "account-service", "test", "commit123")
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	ctx = WithCorrelation(ctx, "00000000-0000-4000-8000-000000000002")
	slog.WarnContext(ctx, "controlled failure", "error", errors.New("postgres://user:secret@host/db"), "password", "private-value", "email", "person@example.com")
	log.Printf("Authorization: Bearer sensitive-jwt")
	for _, secret := range []string{"sensitive-jwt", "private-value", "person@example.com", "postgres://"} {
		if strings.Contains(out.String(), secret) {
			t.Fatal("secret in logs")
		}
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"timestamp", "serviceName", "environment", "version", "level", "message", "correlationId", "traceId", "spanId"} {
		if _, ok := record[key]; !ok {
			t.Fatalf("missing %s", key)
		}
	}
	if record["level"] != "WARNING" || record["correlationId"] != "00000000-0000-4000-8000-000000000002" || record["traceId"] == nil {
		t.Fatal("log context missing")
	}
}
func TestOTLPExporterAndUnavailableCollector(t *testing.T) {
	var received atomic.Int32
	var tracesReceived atomic.Int32
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" && r.URL.Path != "/v1/metrics" {
			t.Errorf("unexpected OTLP path %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if len(b) == 0 {
			t.Error("empty OTLP payload")
		}
		received.Add(1)
		if r.URL.Path == "/v1/traces" {
			tracesReceived.Add(1)
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(200)
	}))
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "1")
	oldTP, oldMP, oldLog, oldWriter, oldEH := otel.GetTracerProvider(), otel.GetMeterProvider(), slog.Default(), log.Writer(), otel.GetErrorHandler()
	defer func() {
		otel.SetTracerProvider(oldTP)
		otel.SetMeterProvider(oldMP)
		slog.SetDefault(oldLog)
		log.SetOutput(oldWriter)
		otel.SetErrorHandler(oldEH)
	}()
	stop := Setup("test-service")
	_, span := otel.Tracer("test").Start(context.Background(), "exported")
	span.End()
	stop()
	if received.Load() == 0 || tracesReceived.Load() == 0 {
		t.Fatal("no OTLP export")
	}
	collector.Close()
	stop = Setup("test-service")
	start := time.Now()
	_, span = otel.Tracer("test").Start(context.Background(), "collector-down")
	span.End()
	if time.Since(start) > time.Second {
		t.Fatal("request blocked on Collector")
	}
	stop()
}
func TestPartnerHTTPTransport(t *testing.T) {
	tp, _ := tracing(t)
	ctx, root := tp.Tracer("test").Start(context.Background(), "transfer")
	defer root.End()
	expected := root.SpanContext().TraceID()
	partner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remote := Restore(context.Background(), fmtCarrier(r.Header.Get("traceparent")))
		if trace.SpanContextFromContext(remote).TraceID() != expected {
			t.Error("partner trace split")
		}
		w.WriteHeader(202)
	}))
	defer partner.Close()
	req, _ := http.NewRequestWithContext(ctx, "POST", partner.URL, strings.NewReader("{}"))
	res, err := (&http.Client{Transport: Transport{}}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if req.Header.Get("traceparent") != "" {
		t.Fatal("original request mutated")
	}
}
func fmtCarrier(parent string) string {
	b, _ := json.Marshal(map[string]string{"traceparent": parent})
	return string(b)
}
