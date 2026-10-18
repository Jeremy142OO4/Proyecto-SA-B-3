package repositories

import (
	"context"
	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	db, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())
	ctx, span := tp.Tracer("test").Start(context.Background(), "business transaction")
	expected := span.SpanContext().TraceID()
	_, err = NuevoRepositorioSalidaPostgres(db).RegistrarRespuesta(ctx, events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: uuid.New(), Tipo: "cuenta.consulta.solicitada"}, "telemetry-test", "cuenta.consultada", map[string]bool{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	span.End()
	pending, err := NuevoRepositorioSalidaPostgres(db).ListarPendientes(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range pending {
		restored := telemetry.Restore(context.Background(), m.TraceContext)
		if trace.SpanContextFromContext(restored).TraceID() == expected {
			found = true
		}
	}
	if !found {
		t.Fatal("Outbox lost persisted trace context")
	}
}
