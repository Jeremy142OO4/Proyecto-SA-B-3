package repositories

import (
	"context"
	"github.com/Proyecto-SA-B-3/transaction-service/events"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/transaction-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTransferFlowObservabilityDatabase(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("isolated TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "flow_" + uuid.New().String()[:8]
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrations, _ := filepath.Glob("../database/migrations/*.up.sql")
	for _, file := range migrations {
		b, _ := os.ReadFile(file)
		if _, err = db.Exec(ctx, string(b)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	oldTrace, oldMetric := otel.GetTracerProvider(), otel.GetMeterProvider()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	defer func() {
		tp.Shutdown(ctx)
		mp.Shutdown(ctx)
		otel.SetTracerProvider(oldTrace)
		otel.SetMeterProvider(oldMetric)
	}()
	repo := NuevoPostgres(db)
	cases := []string{"success", "kyc", "accounts", "debit", "credit", "external_failure", "timeout", "compensation_failure", "rollback"}
	for _, scenario := range cases {
		t.Run(scenario, func(t *testing.T) {
			id, client, corr, origin, destination := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
			start, root := tp.Tracer("test").Start(ctx, "request")
			traceID := root.SpanContext().TraceID()
			simulation := "EXITO"
			if scenario == "external_failure" || scenario == "compensation_failure" {
				simulation = "FALLO"
			}
			if scenario == "timeout" {
				simulation = "TIMEOUT"
			}
			transfer := models.Transferencia{IDTransferencia: id, IDCliente: client, IDCuentaOrigen: origin, IDCuentaDestino: destination, IDCorrelacion: corr, MontoCentavos: 100, Estado: models.ValidandoKYC, ResultadoExternoSimulado: simulation, FechaCreacion: time.Now().Add(-2 * time.Second)}
			message := func(kind string) events.SobreMensaje {
				return events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: corr, Tipo: kind}
			}
			initial := message(events.ComandoTransferencia)
			if ok, e := repo.Iniciar(start, initial, transfer); e != nil || !ok {
				t.Fatalf("start: %v", e)
			}
			root.End()
			if ok, e := repo.Iniciar(ctx, initial, transfer); e != nil || ok {
				t.Fatalf("duplicate start: %v", e)
			}
			kyc := scenario != "kyc" && scenario != "rollback"
			kycMessage := message(events.EventoKYCVerificado)
			if scenario == "rollback" {
				_, e := db.Exec(ctx, `CREATE FUNCTION reject_test_state() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'injected rollback'; END; $$ LANGUAGE plpgsql; CREATE TRIGGER reject_test_state BEFORE UPDATE OF estado ON transferencias FOR EACH ROW EXECUTE FUNCTION reject_test_state();`)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = repo.ProcesarResultadoKYC(ctx, kycMessage, events.ResultadoValidacionKYC{IDOperacion: id, IDCliente: client, Valido: false}); e == nil {
					t.Fatal("expected injected rollback")
				}
				var count int
				if e = db.QueryRow(ctx, `SELECT count(*) FROM mensajes_procesados WHERE id_mensaje=$1`, kycMessage.IDMensaje).Scan(&count); e != nil || count != 0 {
					t.Fatal("message survived rollback")
				}
				if _, e = db.Exec(ctx, `DROP TRIGGER reject_test_state ON transferencias; DROP FUNCTION reject_test_state();`); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := repo.ProcesarResultadoKYC(ctx, kycMessage, events.ResultadoValidacionKYC{IDOperacion: id, IDCliente: client, Valido: kyc}); e != nil {
				t.Fatal(e)
			}
			if kyc {
				if _, e := repo.ProcesarResultadoCuentas(ctx, message(events.EventoCuentasValidadas), events.ResultadoValidacionCuentas{IDOperacion: id, Valida: scenario != "accounts"}); e != nil {
					t.Fatal(e)
				}
				if scenario != "accounts" {
					kind := events.EventoDebitada
					if scenario == "debit" {
						kind = events.EventoDebitoRechazado
					}
					if _, e := repo.ProcesarResultado(ctx, message(kind), events.ResultadoMovimiento{IDOperacion: id}); e != nil {
						t.Fatal(e)
					}
					if scenario != "debit" {
						if simulation == "EXITO" {
							kind = events.EventoAcreditada
							if scenario == "credit" {
								kind = events.EventoCreditoRechazado
							}
							m := message(kind)
							if _, e := repo.ProcesarResultado(ctx, m, events.ResultadoMovimiento{IDOperacion: id}); e != nil {
								t.Fatal(e)
							}
							if ok, e := repo.ProcesarResultado(ctx, m, events.ResultadoMovimiento{IDOperacion: id}); e != nil || ok {
								t.Fatalf("duplicate: %v", e)
							}
						}
						if simulation != "EXITO" || scenario == "credit" {
							kind = events.EventoCuentaCompensada
							if scenario == "compensation_failure" {
								kind = events.EventoCompensacionRechazada
							}
							if _, e := repo.ProcesarResultado(ctx, message(kind), events.ResultadoMovimiento{IDOperacion: id}); e != nil {
								t.Fatal(e)
							}
						}
					}
				}
			}
			got, e := repo.Consultar(ctx, id)
			if e != nil || got.TraceID != traceID.String() {
				t.Fatalf("original trace lost: %+v %v", got, e)
			}
			pending, e := repo.ListarPendientes(ctx, 1000)
			if e != nil {
				t.Fatal(e)
			}
			for _, out := range pending {
				if out.IDCorrelacion == corr && trace.SpanContextFromContext(telemetry.Restore(ctx, out.TraceContext)).TraceID() != traceID {
					t.Fatal("outbox split trace")
				}
			}
			found := false
			for _, span := range exporter.GetSpans() {
				if span.Name == "transferencia.estado" && span.SpanContext.TraceID() == traceID {
					for _, event := range span.Events {
						if event.Name == telemetry.Phase2State(got.Estado) {
							found = true
						}
					}
				}
			}
			if !found {
				t.Fatal("terminal phase2 trace event missing")
			}
		})
	}
	var data metricdata.ResourceMetrics
	if err = reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}
	var outcomes, observations uint64
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name == "bank_transfer_results" {
				for _, point := range m.Data.(metricdata.Sum[int64]).DataPoints {
					outcomes += uint64(point.Value)
				}
			}
			if m.Name == "bank_transfer_flow_duration" {
				for _, point := range m.Data.(metricdata.Histogram[float64]).DataPoints {
					if len(point.Bounds) < 10 || point.Sum < 2*float64(point.Count) {
						t.Fatal("not an end-to-end histogram")
					}
					observations += point.Count
				}
			}
		}
	}
	if outcomes != uint64(len(cases)) || observations != outcomes {
		t.Fatalf("duplicate/absent measurements: outcomes=%d histogram=%d", outcomes, observations)
	}
}
