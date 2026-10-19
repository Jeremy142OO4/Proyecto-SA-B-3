package telemetry

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

type sourceTraceKey struct{}

func SourceTraceID(ctx context.Context) string {
	if source, ok := ctx.Value(sourceTraceKey{}).(string); ok {
		return source
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}
