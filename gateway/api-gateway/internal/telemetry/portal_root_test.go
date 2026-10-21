package telemetry

import (
	"context"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestPortalStartsRealRootAndLinksExternalProxy(t *testing.T) {
	id, _ := trace.TraceIDFromHex("c148b97a2cd9e4f357c312a2353ed161")
	pid, _ := trace.SpanIDFromHex("57591b2b5ecfd05f")
	for _, sampled := range []bool{false, true} {
		for _, path := range []string{"/api/transferencias", "/api/pagos", "/api/clientes/registro"} {
			ex := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(gatewaySampler("dev", 1)), sdktrace.WithSyncer(ex))
			flags := trace.TraceFlags(0)
			if sampled {
				flags = trace.FlagsSampled
			}
			parent := trace.NewSpanContext(trace.SpanContextConfig{TraceID: id, SpanID: pid, Remote: true, TraceFlags: flags})
			ctx := trace.ContextWithRemoteSpanContext(context.Background(), parent)
			ctx, span := tp.Tracer("test").Start(ctx, "http request", portalSpanOptions(path, ctx)...)
			ownID := span.SpanContext().TraceID()
			if ownID == id {
				t.Fatal("portal inherited an external proxy root")
			}
			saved := Serialize(ctx)
			_, child := tp.Tracer("test").Start(Restore(context.Background(), saved), "business step")
			if child.SpanContext().TraceID() != ownID {
				t.Fatal("business flow split into another trace")
			}
			child.End()
			span.End()
			found := false
			for _, s := range ex.GetSpans() {
				if s.Name != "http request" {
					continue
				}
				found = true
				if s.Parent.IsValid() {
					t.Fatal("portal span still has a missing external parent")
				}
				if len(s.Links) != 1 || s.Links[0].SpanContext.TraceID() != id {
					t.Fatal("external correlation link missing")
				}
			}
			if !found {
				t.Fatal("root was not exported")
			}
			tp.Shutdown(context.Background())
		}
	}
}

func TestInterbankKeepsExternalTraceParent(t *testing.T) {
	ex := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(ex))
	defer tp.Shutdown(context.Background())
	id, _ := trace.TraceIDFromHex("c148b97a2cd9e4f357c312a2353ed161")
	pid, _ := trace.SpanIDFromHex("57591b2b5ecfd05f")
	parent := trace.NewSpanContext(trace.SpanContextConfig{TraceID: id, SpanID: pid, Remote: true, TraceFlags: trace.FlagsSampled})
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), parent)
	_, span := tp.Tracer("test").Start(ctx, "partner request", portalSpanOptions("/interbank/v1/transfers", ctx)...)
	span.End()
	s := ex.GetSpans()[0]
	if s.SpanContext.TraceID() != id || s.Parent.SpanID() != pid {
		t.Fatal("interbank trace propagation changed")
	}
}
