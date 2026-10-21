package telemetry

import (
	"context"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestGatewaySamplingForRemoteUnsampledParent(t *testing.T) {
	id, _ := trace.TraceIDFromHex("8ed62e5570fa9fbe5a43f8a62ac16bb2")
	parentID, _ := trace.SpanIDFromHex("32e009043f6948a1")
	for _, tc := range []struct {
		env     string
		ratio   float64
		sampled bool
	}{
		{"dev", 1, true}, {"local", 1, true}, {"dev", 0, false}, {"prod", 1, false},
	} {
		t.Run(tc.env+gatewaySampler(tc.env, tc.ratio).Description(), func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(gatewaySampler(tc.env, tc.ratio)), sdktrace.WithSyncer(exporter))
			defer provider.Shutdown(context.Background())
			parent := trace.NewSpanContext(trace.SpanContextConfig{TraceID: id, SpanID: parentID, Remote: true})
			ctx := trace.ContextWithRemoteSpanContext(context.Background(), parent)
			ctx, span := provider.Tracer("test").Start(ctx, "POST /api/transferencias")
			if span.SpanContext().TraceID() != id {
				t.Fatal("upstream TraceID was replaced")
			}
			if span.SpanContext().IsSampled() != tc.sampled || span.IsRecording() != tc.sampled {
				t.Fatal("incorrect sampling decision")
			}
			restored := Restore(context.Background(), Serialize(ctx))
			downstream := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1))))
			_, child := downstream.Tracer("test").Start(restored, "transferencia.inicio")
			if child.SpanContext().TraceID() != id || child.SpanContext().IsSampled() != tc.sampled {
				t.Fatal("broker context lost sampling/trace identity")
			}
			child.End()
			downstream.Shutdown(context.Background())
			span.End()
			expected := 0
			if tc.sampled {
				expected = 1
			}
			if len(exporter.GetSpans()) != expected {
				t.Fatal("recorded spans did not reach exporter")
			}
		})
	}
}
