// Package telemetry implements the versioned Bank USAC telemetry contract.
// Kept local to each independent module so service Docker build contexts remain isolated.
package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var BuildVersion = "unknown"
var metricIdentity []attribute.KeyValue
var knownEvents = map[string]bool{"SLOAlertTriggered": true, "auditoria.slo.solicitada": true, "auditoria.slo.respondida": true, "pago.simulacion.consulta.solicitada": true, "pago.simulacion.actualizacion.solicitada": true, "pago.simulacion.respondida": true, "pago.simulacion.configuracion.actualizada": true, "pago.proveedor.fallido": true, "pago.compensando": true, "pago.compensacion.fallida": true, "AccountAnomalyDetected": true, "cuenta.bloqueada": true, "cuenta.monitoreo.configuracion.actualizada": true, "cuenta.administracion.respondida": true, "cuenta.administracion.listado.solicitado": true, "cuenta.monitoreo.consulta.solicitada": true, "cuenta.monitoreo.actualizacion.solicitada": true, "cuenta.bloqueo.solicitado": true, "auditoria.error.http": true, "auditoria.notificaciones.solicitadas": true, "auditoria.registros.solicitados": true, "auditoria.traza.solicitada": true, "cliente.activacion.solicitada": true, "cliente.activado": true, "cliente.actualizacion.solicitada": true, "cliente.actualizado": true, "cliente.creado": true, "cliente.dpi.consulta.solicitada": true, "cliente.estado.actualizado": true, "cliente.estado.solicitado": true, "cliente.kyc.estado.actualizado": true, "cliente.kyc.estado.solicitado": true, "cliente.kyc.rechazado": true, "cliente.kyc.validacion.solicitada": true, "cliente.kyc.verificado": true, "cliente.listado.solicitado": true, "cliente.login.solicitado": true, "cliente.perfil.respondido": true, "cliente.perfil.solicitado": true, "cliente.rechazado": true, "cliente.registro.solicitado": true, "cliente.validacion.fallida": true, "cliente.validacion.solicitada": true, "cliente.validado": true, "cuenta.acreditada": true, "cuenta.comando.fallido": true, "cuenta.compensacion.rechazada": true, "cuenta.compensacion.solicitada": true, "cuenta.compensada": true, "cuenta.consulta.solicitada": true, "cuenta.consultada": true, "cuenta.creacion.rechazada": true, "cuenta.creacion.solicitada": true, "cuenta.creada": true, "cuenta.credito.rechazado": true, "cuenta.credito.solicitado": true, "cuenta.debitada": true, "cuenta.debito.rechazado": true, "cuenta.debito.solicitado": true, "cuenta.desactivada": true, "cuenta.historial.consultado": true, "cuenta.historial.solicitado": true, "cuenta.movimientos.consultados": true, "cuenta.movimientos.solicitados": true, "cuenta.transferencia.rechazada": true, "cuenta.transferencia.validacion.solicitada": true, "cuenta.transferencia.validada": true, "notificacion.correo-activacion.solicitado": true, "pago.completado": true, "pago.consulta.solicitada": true, "pago.consultado": true, "pago.historial.consultado": true, "pago.historial.solicitado": true, "pago.mensaje.fallido": true, "pago.procesamiento.solicitado": true, "pago.procesando": true, "pago.rechazado": true, "pago.timeout": true, "transferencia.compensacion.fallida": true, "transferencia.compensada": true, "transferencia.compensando": true, "transferencia.completada": true, "transferencia.consulta.solicitada": true, "transferencia.consultada": true, "transferencia.historial.consultado": true, "transferencia.historial.solicitado": true, "transferencia.procesando": true, "transferencia.rechazada": true, "transferencia.solicitada": true}

func eventName(k string) string {
	if knownEvents[k] {
		return k
	}
	return "other"
}

type correlationKey struct{}

var propagator = propagation.TraceContext{}
var buckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2, 5, 8, 10, 15, 20, 25, 30, 60, 120, 180, 300, 480, 900, 3600, 86400}

// Setup never dials synchronously to an optional Collector during request handling.
// OTLP/HTTP honors standard OTEL_EXPORTER_OTLP_* endpoint and TLS settings.
func Setup(service string) func() {
	environment := firstEnv("DEPLOYMENT_ENVIRONMENT", "APP_ENV", "ENTORNO")
	switch environment {
	case "desarrollo":
		environment = "dev"
	case "produccion":
		environment = "prod"
	}
	if environment == "" {
		environment = "local"
	}
	version := firstEnv("SERVICE_VERSION", "IMAGE_TAG")
	if version == "" {
		version = BuildVersion
	}
	metricIdentity = []attribute.KeyValue{attribute.String("service", service), attribute.String("environment", environment)}
	ConfigureLogger(os.Stdout, service, environment, version)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {
		slog.Warn("telemetry export failed", "errorCode", "OTEL_EXPORT_FAILED")
	}))
	res := resource.NewWithAttributes("",
		attribute.String("service.name", service),
		attribute.String("service.version", version),
		attribute.String("deployment.environment.name", environment))
	ratio := 1.0
	if raw := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed >= 0 && parsed <= 1 {
			ratio = parsed
		} else {
			slog.Warn("invalid trace sampling ratio; using 1")
		}
	}
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res), sdktrace.WithSampler(gatewaySampler(environment, ratio))}
	metricOpts := []sdkmetric.Option{sdkmetric.WithResource(res)}
	if os.Getenv("OTEL_SDK_DISABLED") != "true" {
		if firstEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
			exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithTimeout(2*time.Second))
			if err == nil {
				opts = append(opts, sdktrace.WithBatcher(exporter, sdktrace.WithMaxQueueSize(1024), sdktrace.WithBatchTimeout(time.Second), sdktrace.WithExportTimeout(2*time.Second)))
			} else {
				slog.Warn("trace exporter configuration invalid", "errorCode", "OTEL_CONFIG")
			}
		}
		if firstEnv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
			exporter, err := otlpmetrichttp.New(context.Background(), otlpmetrichttp.WithTimeout(2*time.Second))
			if err == nil {
				metricOpts = append(metricOpts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(15*time.Second), sdkmetric.WithTimeout(2*time.Second))))
			} else {
				slog.Warn("metric exporter configuration invalid", "errorCode", "OTEL_CONFIG")
			}
		}
	}
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		opts = append(opts, sdktrace.WithSampler(sdktrace.NeverSample()))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	mp := sdkmetric.NewMeterProvider(metricOpts...)
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagator)
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}
}

// Development must record banking flows even when an upstream proxy supplies
// traceparent with sampled=0. Keep the upstream TraceID, but make the recording
// decision at our gateway boundary. Production retains normal parent sampling.
func gatewaySampler(environment string, ratio float64) sdktrace.Sampler {
	root := sdktrace.TraceIDRatioBased(ratio)
	if environment == "dev" || environment == "local" {
		return sdktrace.ParentBased(root, sdktrace.WithRemoteParentNotSampled(root))
	}
	return sdktrace.ParentBased(root)
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func WithCorrelation(ctx context.Context, id string) context.Context {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil {
		return ctx
	}
	id = parsed.String()
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("correlationId", id))
	return context.WithValue(ctx, correlationKey{}, id)
}

// Serialize is stored beside the Outbox payload in the SAME business transaction.
func Serialize(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	if len(carrier) == 0 {
		return ""
	}
	b, _ := json.Marshal(carrier)
	return string(b)
}
func Restore(ctx context.Context, saved string) context.Context {
	carrier := propagation.MapCarrier{}
	if json.Unmarshal([]byte(saved), &carrier) != nil {
		return ctx
	}
	return propagator.Extract(ctx, carrier)
}
func AMQPHeaders(ctx context.Context) amqp.Table {
	out := amqp.Table{}
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	for k, v := range carrier {
		out[k] = v
	}
	return out
}
func headers(in amqp.Table) propagation.MapCarrier {
	out := propagation.MapCarrier{}
	for _, k := range []string{"traceparent", "tracestate"} {
		switch v := in[k].(type) {
		case string:
			out[k] = v
		case []byte:
			out[k] = string(v)
		}
	}
	return out
}
func envelope(body []byte) (correlation, message string) {
	var e struct {
		Correlation string `json:"idCorrelacion"`
		Message     string `json:"idMensaje"`
	}
	_ = json.Unmarshal(body, &e)
	return e.Correlation, e.Message
}
func endSpan(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, "operation failed")
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
	}
	span.End()
}

// Publish copies headers, preserving retry metadata and never modifying caller maps.
func Publish(ctx context.Context, in amqp.Table, body []byte, key string) (context.Context, amqp.Table, func(error)) {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		ctx = propagator.Extract(ctx, headers(in))
	}
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "amqp publish "+eventName(key), trace.WithSpanKind(trace.SpanKindProducer))
	correlation, message := envelope(body)
	ctx = WithCorrelation(ctx, correlation)
	span.SetAttributes(attribute.String("messageId", message))
	out := amqp.Table{}
	for k, v := range in {
		out[k] = v
	}
	carrier := propagation.MapCarrier{}
	propagator.Inject(ctx, carrier)
	for k, v := range carrier {
		out[k] = v
	}
	return ctx, out, func(err error) {
		action := "published"
		if err != nil {
			action = "publish_failed"
		}
		count, _ := otel.Meter("bank-usac").Int64Counter("bank_messages")
		if count != nil {
			count.Add(ctx, 1, metric.WithAttributes(append(metricIdentity, attribute.String("event_type", eventName(key)), attribute.String("action", action))...))
		}
		endSpan(span, err)
	}
}
func Consume(ctx context.Context, d amqp.Delivery) (context.Context, func(error)) {
	ctx = propagator.Extract(ctx, headers(d.Headers))
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "amqp process "+eventName(d.RoutingKey), trace.WithSpanKind(trace.SpanKindConsumer))
	correlation, message := envelope(d.Body)
	ctx = WithCorrelation(ctx, correlation)
	span.SetAttributes(attribute.String("messageId", message), attribute.Bool("messaging.redelivered", d.Redelivered))
	start := time.Now()
	return ctx, func(err error) {
		outcome := "processed"
		if err != nil {
			outcome = "failed"
		}
		meter := otel.Meter("bank-usac")
		count, _ := meter.Int64Counter("bank_messages")
		duration, _ := meter.Float64Histogram("bank_message_processing", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(buckets...))
		attrs := metric.WithAttributes(append(metricIdentity, attribute.String("event_type", eventName(d.RoutingKey)), attribute.String("action", outcome))...)
		if count != nil {
			count.Add(ctx, 1, attrs)
		}
		if duration != nil {
			duration.Record(ctx, time.Since(start).Seconds(), attrs)
		}
		slog.InfoContext(ctx, "amqp processing finished", "outcome", outcome)
		endSpan(span, err)
	}
}

// HTTP handles errors before recording status so 401/403/500 are not recorded as 200.
func HTTP() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if path == "/salud" || strings.HasPrefix(path, "/health/") || path == "/health" || path == "/ready" || path == "/healthz" || path == "/readyz" || path == "/metrics" {
			return c.Next()
		}
		carrier := propagation.MapCarrier{"traceparent": c.Get("traceparent"), "tracestate": c.Get("tracestate")}
		ctx := propagator.Extract(c.UserContext(), carrier)
		ctx, span := otel.Tracer("bank-usac").Start(ctx, "http request", trace.WithSpanKind(trace.SpanKindServer))
		c.SetUserContext(ctx)
		defer span.End()
		response := propagation.MapCarrier{}
		propagator.Inject(ctx, response)
		for k, v := range response {
			c.Set(k, v)
		}
		start := time.Now()
		err := c.Next()
		if err != nil {
			if handled := c.App().ErrorHandler(c, err); handled != nil {
				c.Status(500)
				err = handled
			}
		}
		route := "unmatched"
		if r := c.Route(); r != nil && r.Path != "" && r.Path != "/*" {
			route = r.Path
		}
		status := c.Response().StatusCode()
		span.SetName(c.Method() + " " + route)
		span.SetAttributes(attribute.String("http.route", route), attribute.Int("http.response.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, "HTTP server error")
		}
		ctx = WithCorrelation(ctx, c.GetRespHeader("X-Correlation-ID"))
		attrs := metric.WithAttributes(append(metricIdentity, attribute.String("method", c.Method()), attribute.String("route", route), attribute.String("status_class", strconv.Itoa(status/100)+"xx"))...)
		meter := otel.Meter("bank-usac")
		count, _ := meter.Int64Counter("bank_http_requests")
		duration, _ := meter.Float64Histogram("bank_http_duration", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(buckets...))
		if count != nil {
			count.Add(ctx, 1, attrs)
		}
		if duration != nil {
			duration.Record(ctx, time.Since(start).Seconds(), attrs)
		}
		slog.InfoContext(ctx, "http request finished", "method", c.Method(), "route", route, "status", status)
		return nil
	}
}

// JSON handler adds context IDs at emission time, not through mutable global state.
type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	var correlation, tid, sid any
	if id, ok := ctx.Value(correlationKey{}).(string); ok {
		correlation = id
	}
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		tid = sc.TraceID().String()
		sid = sc.SpanID().String()
	}
	r.AddAttrs(slog.Any("correlationId", correlation), slog.Any("traceId", tid), slog.Any("spanId", sid))
	return h.Handler.Handle(ctx, r)
}
func (h contextHandler) WithAttrs(a []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(a)}
}
func (h contextHandler) WithGroup(g string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(g)}
}

// Legacy Printf can contain arbitrary secrets. Keep a safe signal rather than the raw text.
type legacyWriter struct{ logger *slog.Logger }

func (w legacyWriter) Write(b []byte) (int, error) {
	w.logger.Info("legacy log emitted; unstructured details omitted")
	return len(b), nil
}

func ConfigureLogger(w io.Writer, service, environment, version string) {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			a.Key = "timestamp"
			a.Value = slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339Nano))
		}
		if a.Key == slog.MessageKey {
			a.Key = "message"
		}
		if a.Key == slog.LevelKey && a.Value.String() == "WARN" {
			a.Value = slog.StringValue("WARNING")
		}
		lower := strings.ToLower(a.Key)
		if _, isError := a.Value.Any().(error); isError {
			a.Value = slog.StringValue("[REDACTED]")
		}
		if lower == "authorization" || lower == "cookie" || strings.Contains(lower, "connection") || strings.Contains(lower, "photo") || lower == "address" || lower == "fullname" {
			a.Value = slog.StringValue("[REDACTED]")
		}
		if strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "email") || strings.Contains(lower, "document") || strings.Contains(lower, "payload") || strings.Contains(lower, "body") || lower == "error" {
			a.Value = slog.StringValue("[REDACTED]")
		}
		return a
	}})
	logger := slog.New(contextHandler{handler}).With("serviceName", service, "environment", environment, "version", version)
	slog.SetDefault(logger)
	log.SetFlags(0)
	log.SetOutput(legacyWriter{logger})
}

// Database wraps a repository operation without recording SQL text or parameters.
func Database(ctx context.Context, name string) (context.Context, func()) {
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "postgres "+name, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("db.system.name", "postgresql")))
	return ctx, func() { span.End() }
}

// Transport propagates context to a partner while preserving cancellation and TLS validation.
type Transport struct{ Base http.RoundTripper }

func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, span := otel.Tracer("bank-usac").Start(req.Context(), "http partner request", trace.WithSpanKind(trace.SpanKindClient))
	clone := req.Clone(ctx)
	propagator.Inject(ctx, propagation.HeaderCarrier(clone.Header))
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	response, err := base.RoundTrip(clone)
	if response != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", response.StatusCode))
		if response.StatusCode >= 500 {
			span.SetStatus(codes.Error, "partner server error")
		}
	}
	endSpan(span, err)
	return response, err
}
