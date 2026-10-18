package repositories

import (
	"bank-usac/service-customer/internal/telemetry"
	"context"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"os"
	"testing"
)

func TestTelemetryOutboxDatabase(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("isolated TEST_DATABASE_URL required")
	}
	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())
	ctx, span := tp.Tracer("test").Start(context.Background(), "business transaction")
	expected := span.SpanContext().TraceID()
	repo := NewCustomerRepository(db)
	err = repo.RegistrarValidacionKYC(ctx, uuid.New(), uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	span.End()
	pending, err := repo.GetPendingOutbox(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range pending {
		if trace.SpanContextFromContext(telemetry.Restore(context.Background(), m.TraceContext)).TraceID() == expected {
			found = true
		}
	}
	if !found {
		t.Fatal("customer Outbox lost persisted trace context")
	}
}
