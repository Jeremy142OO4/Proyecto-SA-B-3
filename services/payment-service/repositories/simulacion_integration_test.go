package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/payment-service/events"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/Proyecto-SA-B-3/payment-service/models"
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
	"strings"
	"testing"
	"time"
)

func TestAdvancedPaymentSimulationDatabase(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("isolated TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := "payment_" + uuid.New().String()[:8]
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := filepath.Glob("../database/migrations/*.up.sql")
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if _, e = db.Exec(ctx, string(b)); e != nil {
			t.Fatalf("%s: %v", f, e)
		}
	}
	// The deployment step is repeatable on an already migrated database.
	b, _ := os.ReadFile("../database/migrations/000005_simulacion_avanzada.up.sql")
	if _, e = db.Exec(ctx, string(b)); e != nil {
		t.Fatal(e)
	}
	oldT, oldM := otel.GetTracerProvider(), otel.GetMeterProvider()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	defer func() { tp.Shutdown(ctx); mp.Shutdown(ctx); otel.SetTracerProvider(oldT); otel.SetMeterProvider(oldM) }()
	repo := NuevoRepositorioPagosPostgres(db)
	msg := func(kind string, corr uuid.UUID) events.SobreMensaje {
		return events.SobreMensaje{IDMensaje: uuid.New(), IDCorrelacion: corr, Tipo: kind}
	}
	setConfig := func(c models.ConfiguracionSimulacion) events.SobreMensaje {
		t.Helper()
		payload, _ := json.Marshal(struct {
			models.ConfiguracionSimulacion
			IDAdministrador uuid.UUID `json:"idAdministrador"`
		}{c, uuid.New()})
		m := msg(ActualizarSimulacion, uuid.New())
		m.Contenido = payload
		if e := repo.AdministrarSimulacion(ctx, m); e != nil {
			t.Fatal(e)
		}
		return m
	}
	newPayment := func(kind string) (uuid.UUID, uuid.UUID, trace.TraceID) {
		t.Helper()
		id, corr, client := uuid.New(), uuid.New(), uuid.New()
		start, span := tp.Tracer("test").Start(ctx, "payment request")
		original := span.SpanContext().TraceID()
		_, _, e := repo.Iniciar(start, msg(events.ComandoProcesarPago, corr), events.SolicitudPago{IDPago: id, IDCliente: client, IDCuentaOrigen: uuid.New(), MontoCentavos: 100, TipoPago: kind, ResultadoSimulado: "EXITO", Beneficiario: "sensitive beneficiary", Concepto: "private concept"})
		span.End()
		if e != nil {
			t.Fatal(e)
		}
		if e = repo.ProcesarResultadoKYC(ctx, msg(events.EventoKYCVerificado, corr), events.ResultadoValidacionKYC{IDOperacion: id, IDCliente: client, Valido: true}); e != nil {
			t.Fatal(e)
		}
		if e = repo.ProcesarResultadoValidacionCuenta(ctx, msg(events.EventoCuentaValidada, corr), events.ResultadoValidacionCuenta{IDOperacion: id, IDCliente: client, Valida: true}); e != nil {
			t.Fatal(e)
		}
		return id, corr, original
	}
	counts := func() (int64, uint64) {
		t.Helper()
		var data metricdata.ResourceMetrics
		if e := reader.Collect(ctx, &data); e != nil {
			t.Fatal(e)
		}
		var n int64
		var samples uint64
		for _, scope := range data.ScopeMetrics {
			for _, m := range scope.Metrics {
				if m.Name == "bank_payment_external_results" {
					for _, p := range m.Data.(metricdata.Sum[int64]).DataPoints {
						n += p.Value
					}
				}
				if m.Name == "bank_payment_external_duration" {
					for _, p := range m.Data.(metricdata.Histogram[float64]).DataPoints {
						samples += p.Count
						if len(p.Bounds) < 10 {
							t.Fatal("not a histogram")
						}
					}
				}
			}
		}
		return n, samples
	}
	cases := []struct {
		name   string
		config models.ConfiguracionSimulacion
		result models.ResultadoSimulado
	}{
		{"slow_success", models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 1000, LatenciaMs: 30}, models.ResultadoExito},
		{"intermittent_100", models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 1000, LatenciaMs: 5, TasaFallosPorcentaje: 100}, models.ResultadoFallo},
		{"forced_timeout", models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 20, ForzarTimeout: true}, models.ResultadoTimeout},
		{"latency_timeout", models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 10, LatenciaMs: 100}, models.ResultadoTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := setConfig(tc.config)
			var version int64
			db.QueryRow(ctx, `SELECT version FROM configuracion_simulacion_pagos`).Scan(&version)
			if e = repo.AdministrarSimulacion(ctx, m); e != nil {
				t.Fatal(e)
			}
			var again int64
			db.QueryRow(ctx, `SELECT version FROM configuracion_simulacion_pagos`).Scan(&again)
			if again != version {
				t.Fatal("duplicate administration changed config")
			}
			id, corr, original := newPayment("EXTERNO")
			event := msg(events.EventoCuentaDebitada, corr)
			before, samplesBefore := counts()
			started := time.Now()
			if ok, e := repo.ProcesarResultadoCuenta(ctx, event, events.ResultadoMovimiento{IDOperacion: id}); e != nil || !ok {
				t.Fatalf("provider: %v", e)
			}
			expectedDelay := tc.config.LatenciaMs
			if tc.result == models.ResultadoTimeout {
				expectedDelay = tc.config.TimeoutMs
			}
			if time.Since(started) < time.Duration(expectedDelay)*time.Millisecond {
				t.Fatal("configured delay not applied")
			}
			if ok, e := repo.ProcesarResultadoCuenta(ctx, event, events.ResultadoMovimiento{IDOperacion: id}); e != nil || ok {
				t.Fatal("same-message duplicate")
			}
			if ok, e := repo.ProcesarResultadoCuenta(ctx, msg(events.EventoCuentaDebitada, corr), events.ResultadoMovimiento{IDOperacion: id}); e != nil || ok {
				t.Fatal("new-message duplicate")
			}
			var n int
			db.QueryRow(ctx, `SELECT count(*) FROM simulaciones_pago WHERE id_pago=$1`, id).Scan(&n)
			if n != 1 {
				t.Fatal("duplicate simulation")
			}
			pay, e := repo.BuscarPorID(ctx, id)
			if e != nil || pay.ResultadoSimulado != tc.result {
				t.Fatalf("wrong outcome %+v %v", pay, e)
			}
			var commands int
			db.QueryRow(ctx, `SELECT count(*) FROM mensajes_salida WHERE id_correlacion=$1 AND tipo_evento=$2`, corr, events.ComandoSolicitarCompensacion).Scan(&commands)
			if tc.result == models.ResultadoExito {
				if pay.Estado != models.EstadoPagoCompletado || commands != 0 {
					t.Fatal("success compensated")
				}
			} else {
				if pay.Estado != models.EstadoPagoCompensando || commands != 1 {
					t.Fatal("failure must compensate once")
				}
				var payload []byte
				db.QueryRow(ctx, `SELECT contenido FROM mensajes_salida WHERE id_correlacion=$1 AND tipo_evento=$2`, corr, events.ComandoSolicitarCompensacion).Scan(&payload)
				var command events.SolicitudMovimiento
				json.Unmarshal(payload, &command)
				if command.IDCuenta != pay.IDCuentaOrigen || command.IDOperacion != id || command.MontoCentavos != 100 {
					t.Fatal("wrong compensation command")
				}
				if _, e = repo.ProcesarResultadoCuenta(ctx, msg(events.EventoCuentaCompensada, corr), events.ResultadoMovimiento{IDOperacion: id}); e != nil {
					t.Fatal(e)
				}
				pay, _ = repo.BuscarPorID(ctx, id)
				if pay.Estado != models.EstadoPagoRechazado {
					t.Fatal("compensation not finalized")
				}
			}
			after, samples := counts()
			if after != before+1 || samples != samplesBefore+1 {
				t.Fatalf("duplicate metrics: %d %d %d %d", before, after, samplesBefore, samples)
			}
			pending, e := repo.ListarSalida(ctx, 1000)
			if e != nil {
				t.Fatal(e)
			}
			for _, out := range pending {
				if out.IDCorrelacion == corr && trace.SpanContextFromContext(telemetry.Restore(ctx, out.TraceContext)).TraceID() != original {
					t.Fatal("outbox lost payment trace")
				}
			}
			provider, compensation := false, false
			for _, span := range exporter.GetSpans() {
				if span.SpanContext.TraceID() != original {
					continue
				}
				if span.Name == "pago.proveedor.simulado" {
					provider = true
				}
				for _, event := range span.Events {
					if event.Name == "COMPENSATED" {
						compensation = true
					}
				}
			}
			if !provider || (tc.result != models.ResultadoExito && !compensation) {
				t.Fatal("missing provider/compensation spans")
			}
		})
	}
	t.Run("snapshot_cancellation_and_restart", func(t *testing.T) {
		setConfig(models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 10000, LatenciaMs: 5000, TasaFallosPorcentaje: 100})
		id, corr, _ := newPayment("EXTERNO")
		event := msg(events.EventoCuentaDebitada, corr)
		wait, cancel := context.WithCancel(ctx)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := repo.ProcesarResultadoCuenta(wait, event, events.ResultadoMovimiento{IDOperacion: id})
			done <- err
		}()
		found := false
		for i := 0; i < 500; i++ {
			var n int
			db.QueryRow(ctx, `SELECT count(*) FROM simulaciones_pago WHERE id_pago=$1`, id).Scan(&n)
			if n == 1 {
				found = true
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		if !found {
			t.Fatal("plan not persisted")
		}
		// An admin update must finish while the old provider call is still waiting.
		setConfig(models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 1000})
		select {
		case err := <-done:
			t.Fatalf("admin was blocked until provider completed: %v", err)
		default:
		}
		before, _ := counts()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
		if after, _ := counts(); after != before {
			t.Fatal("cancelled provider counted")
		}
		// Advance this test fixture's clock so retry need not wait another 5 seconds.
		db.Exec(ctx, `UPDATE simulaciones_pago SET inicio=clock_timestamp()-INTERVAL '10 seconds' WHERE id_pago=$1`, id)
		restarted := NuevoRepositorioPagosPostgres(db)
		if _, e := restarted.ProcesarResultadoCuenta(ctx, event, events.ResultadoMovimiento{IDOperacion: id}); e != nil {
			t.Fatal(e)
		}
		pay, _ := repo.BuscarPorID(ctx, id)
		if pay.ResultadoSimulado != models.ResultadoFallo || pay.Estado != models.EstadoPagoCompensando {
			t.Fatal("new config changed in-flight outcome")
		}
		if _, e := repo.ProcesarResultadoCuenta(ctx, msg(events.EventoCompensacionRechazada, corr), events.ResultadoMovimiento{IDOperacion: id}); e != nil {
			t.Fatal(e)
		}
		pay, _ = repo.BuscarPorID(ctx, id)
		if pay.Estado != models.EstadoPagoCompensacionFallida {
			t.Fatal("compensation failure hidden")
		}
		var count int
		db.QueryRow(ctx, `SELECT count(*) FROM simulaciones_pago WHERE id_pago=$1`, id).Scan(&count)
		if count != 1 {
			t.Fatal("new plan after restart")
		}
	})
	t.Run("internal_bypass_and_invalid_config", func(t *testing.T) {
		setConfig(models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 10, ForzarTimeout: true, TasaFallosPorcentaje: 100})
		before, _ := counts()
		id, corr, _ := newPayment("INTERNO")
		if _, e := repo.ProcesarResultadoCuenta(ctx, msg(events.EventoCuentaDebitada, corr), events.ResultadoMovimiento{IDOperacion: id}); e != nil {
			t.Fatal(e)
		}
		pay, _ := repo.BuscarPorID(ctx, id)
		if pay.Estado != models.EstadoPagoCompletado {
			t.Fatal("internal payment affected by simulation")
		}
		if after, _ := counts(); after != before {
			t.Fatal("internal payment in external metric")
		}
		var version, after int64
		db.QueryRow(ctx, `SELECT version FROM configuracion_simulacion_pagos`).Scan(&version)
		setConfig(models.ConfiguracionSimulacion{TasaFallosPorcentaje: 101, TimeoutMs: 1000})
		db.QueryRow(ctx, `SELECT version FROM configuracion_simulacion_pagos`).Scan(&after)
		if after != version {
			t.Fatal("invalid config changed version")
		}
	})

	t.Run("concurrent_duplicate_debits", func(t *testing.T) {
		setConfig(models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 1000, LatenciaMs: 50, TasaFallosPorcentaje: 100})
		id, corr, _ := newPayment("EXTERNO")
		before, _ := counts()
		messages := []events.SobreMensaje{msg(events.EventoCuentaDebitada, corr), msg(events.EventoCuentaDebitada, corr)}
		result := make(chan struct {
			index int
			err   error
		}, 2)
		for i, m := range messages {
			go func(i int, m events.SobreMensaje) {
				_, err := NuevoRepositorioPagosPostgres(db).ProcesarResultadoCuenta(ctx, m, events.ResultadoMovimiento{IDOperacion: id})
				result <- struct {
					index int
					err   error
				}{i, err}
			}(i, m)
		}
		for range messages {
			r := <-result
			if r.err != nil {
				if _, err := repo.ProcesarResultadoCuenta(ctx, messages[r.index], events.ResultadoMovimiento{IDOperacion: id}); err != nil {
					t.Fatal(err)
				}
			}
		}
		var commands int
		db.QueryRow(ctx, `SELECT count(*) FROM mensajes_salida WHERE id_correlacion=$1 AND tipo_evento=$2`, corr, events.ComandoSolicitarCompensacion).Scan(&commands)
		if commands != 1 {
			t.Fatal("concurrent workers duplicated compensation")
		}
		if after, _ := counts(); after != before+1 {
			t.Fatal("concurrent workers duplicated metric")
		}
	})
	t.Run("rollback_is_not_an_external_result", func(t *testing.T) {
		setConfig(models.ConfiguracionSimulacion{Habilitada: true, TimeoutMs: 1000, TasaFallosPorcentaje: 100})
		id, corr, _ := newPayment("EXTERNO")
		before, _ := counts()
		event := msg(events.EventoCuentaDebitada, corr)
		_, e := db.Exec(ctx, `CREATE FUNCTION fail_test_update() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END; $$ LANGUAGE plpgsql; CREATE TRIGGER fail_test_update BEFORE UPDATE ON pagos FOR EACH ROW EXECUTE FUNCTION fail_test_update();`)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = repo.ProcesarResultadoCuenta(ctx, event, events.ResultadoMovimiento{IDOperacion: id}); e == nil {
			t.Fatal("expected rollback")
		}
		db.Exec(ctx, `DROP TRIGGER fail_test_update ON pagos; DROP FUNCTION fail_test_update();`)
		if after, _ := counts(); after != before {
			t.Fatal("rolled-back outcome counted")
		}
		var state string
		db.QueryRow(ctx, `SELECT estado FROM pagos WHERE id_pago=$1`, id).Scan(&state)
		if state != "PROCESANDO" {
			t.Fatal("rollback changed payment state")
		}
		if _, e = repo.ProcesarResultadoCuenta(ctx, event, events.ResultadoMovimiento{IDOperacion: id}); e != nil {
			t.Fatal(e)
		}
		if after, _ := counts(); after != before+1 {
			t.Fatal("retry missing result")
		}
	})
	for _, span := range exporter.GetSpans() {
		for _, a := range span.Attributes {
			if strings.Contains(string(a.Key), "beneficiario") || strings.Contains(string(a.Key), "concepto") {
				t.Fatal("sensitive trace attribute")
			}
		}
	}
}
