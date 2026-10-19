package interbank

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required: isolated PostgreSQL only")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	// Keep the shared extension in public; dropping one test schema must not
	// remove it while another module is installing or using its migrations.
	extension, e := admin.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = extension.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('bank-test-pgcrypto',0))"); e != nil {
		t.Fatal(e)
	}
	if _, e = extension.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public"); e != nil {
		t.Fatal(e)
	}
	if e = extension.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	schema := "interbank_" + uuid.NewString()[:8]
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close(); _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	files, e := filepath.Glob("../../database/migrations/*.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(files)
	for _, file := range files {
		sql, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(ctx, string(sql)); e != nil {
			t.Fatalf("%s: %v", file, e)
		}
	}
	// The deployment migrations must also be safe to execute again.
	latest, e := os.ReadFile(files[len(files)-1])
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, string(latest)); e != nil {
		t.Fatal(e)
	}
	return db
}
