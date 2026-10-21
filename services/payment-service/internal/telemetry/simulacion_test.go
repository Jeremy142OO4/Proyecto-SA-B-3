package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"log/slog"
	"testing"
)

func TestSimulatedErrorStructuredLog(t *testing.T) {
	old, writer, flags := slog.Default(), log.Writer(), log.Flags()
	defer func() { slog.SetDefault(old); log.SetOutput(writer); log.SetFlags(flags) }()
	var b bytes.Buffer
	ConfigureLogger(&b, "payment-service", "test", "test")
	tp, _ := tracing(t)
	ctx, span := tp.Tracer("test").Start(context.Background(), "provider")
	defer span.End()
	corr, id := uuid.NewString(), uuid.NewString()
	ctx = WithCorrelation(ctx, corr)
	ProveedorSimulado(ctx, id, "TIMEOUT", 7, 100)
	var record map[string]any
	if e := json.Unmarshal(bytes.TrimSpace(b.Bytes()), &record); e != nil {
		t.Fatal(e)
	}
	if record["correlationId"] != corr || record["traceId"] != span.SpanContext().TraceID().String() || record["errorCode"] != "SIMULATED_EXTERNAL_TIMEOUT" || record["paymentId"] != id || record["level"] != "ERROR" {
		t.Fatalf("structured simulation log incomplete: %v", record)
	}
}
