package repositories

import (
	"bank-usac/service-notification-audit/models"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditHistoricalFiltersDatabase(t *testing.T) {
	address := os.Getenv("TEST_AUDIT_DATABASE_URL")
	if address == "" {
		t.Skip("isolated TEST_AUDIT_DATABASE_URL required")
	}
	admin, e := sqlx.Connect("postgres", address)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schema := "audit_query_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, e = admin.Exec("CREATE SCHEMA " + schema); e != nil {
		t.Fatal(e)
	}
	defer admin.Exec("DROP SCHEMA " + schema + " CASCADE")
	parsed, _ := url.Parse(address)
	q := parsed.Query()
	q.Set("search_path", schema+",public")
	parsed.RawQuery = q.Encode()
	db, e := sqlx.Connect("postgres", parsed.String())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := filepath.Glob("../database/migrations/*.up.sql")
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if _, e = db.Exec(string(b)); e != nil {
			t.Fatalf("%s: %v", f, e)
		}
	}
	b, _ := os.ReadFile("../database/migrations/000004_consulta_auditoria_trazas.up.sql")
	if _, e = db.Exec(string(b)); e != nil {
		t.Fatal(e)
	}
	repo := NewAuditRepository(db)
	ctx := context.Background()
	corr, other := uuid.New(), uuid.New()
	traceA, traceB := strings.Repeat("a", 32), strings.Repeat("b", 32)
	now := time.Now().UTC().Truncate(time.Second)
	inputs := []struct {
		corr           uuid.UUID
		trace, service string
		level          models.EventSeverity
	}{{corr, traceA, "payment-service", models.EventError}, {corr, traceA, "account-service", models.EventWarning}, {other, traceB, "payment-service", models.EventError}, {corr, "", "customer-service", models.EventInfo}}
	ids := make([]uuid.UUID, len(inputs))
	for i, in := range inputs {
		ids[i] = uuid.New()
		row := &models.AuditLog{ID: ids[i], EventID: uuid.New(), CorrelationID: in.corr, TraceID: in.trace, EventType: "test.event", Producer: in.service, Severity: in.level, Version: 1, Payload: json.RawMessage(`{"idOperacion":"test"}`), OccurredAt: now.Add(time.Duration(i) * time.Second), RecordedAt: now}
		if e := repo.SaveAuditLog(ctx, row); e != nil {
			t.Fatal(e)
		}
		if e := repo.SaveAuditLog(ctx, row); e != nil {
			t.Fatal(e)
		}
	}
	since, until := now.Add(time.Second), now.Add(3*time.Second)
	cases := []struct {
		name string
		f    models.AuditFilter
		want int
	}{
		{"all", models.AuditFilter{}, 4}, {"correlation", models.AuditFilter{CorrelationID: &corr}, 3}, {"trace", models.AuditFilter{TraceID: traceA}, 2}, {"service", models.AuditFilter{Service: "payment-service"}, 2}, {"warning", models.AuditFilter{Severity: models.EventWarning}, 1}, {"error", models.AuditFilter{Severity: models.EventError}, 2}, {"combined", models.AuditFilter{CorrelationID: &corr, TraceID: traceA, Service: "payment-service", Severity: models.EventError}, 1}, {"time", models.AuditFilter{Since: &since, Until: &until}, 2}, {"no_match", models.AuditFilter{TraceID: strings.Repeat("c", 32)}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, e := repo.QueryAuditLogs(ctx, tc.f)
			if e != nil || len(rows) != tc.want {
				t.Fatalf("want %d got %d: %v", tc.want, len(rows), e)
			}
		})
	}
	first, e := repo.QueryAuditLogs(ctx, models.AuditFilter{Limit: 1})
	if e != nil || first[0].ID != ids[3] {
		t.Fatal("order/limit not stable")
	}
	second, e := repo.QueryAuditLogs(ctx, models.AuditFilter{Limit: 1, Offset: 1})
	if e != nil || second[0].ID != ids[2] {
		t.Fatal("offset not applied")
	}
	for _, f := range []models.AuditFilter{{TraceID: "' OR TRUE --"}, {Service: "payment-service' OR TRUE --"}, {Severity: "DEBUG"}, {Limit: 101}, {Offset: -1}, {TraceID: strings.Repeat("0", 32)}} {
		if _, e = repo.QueryAuditLogs(ctx, f); e == nil {
			t.Fatal("invalid filter accepted")
		}
	}
	var indexes int
	if e = db.Get(&indexes, `SELECT count(*) FROM pg_indexes WHERE schemaname=$1 AND indexname IN ('idx_audit_trace_time','idx_audit_service_level_time','idx_audit_correlation_time')`, schema); e != nil || indexes != 3 {
		t.Fatal("query indexes missing")
	}
}
