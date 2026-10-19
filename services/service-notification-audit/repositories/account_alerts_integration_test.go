package repositories

import (
	"bank-usac/service-notification-audit/models"
	"context"
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

func TestAccountAlertInboxPersistsWarningDetailsAndFilters(t *testing.T) {
	address := os.Getenv("TEST_AUDIT_DATABASE_URL")
	if address == "" {
		t.Skip("isolated TEST_AUDIT_DATABASE_URL required")
	}
	admin, err := sqlx.Connect("postgres", address)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "inbox_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	q.Set("search_path", schema+",public")
	parsed.RawQuery = q.Encode()
	db, err := sqlx.Connect("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, _ := filepath.Glob("../database/migrations/*.up.sql")
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(body)); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewNotificationRepository(db)
	account := uuid.New()
	ctx := context.Background()
	warning := &models.NotificationLog{ID: uuid.New(), CorrelationID: uuid.New(), Recipient: "test@example.test", NotificationType: "ACCOUNT_ANOMALY_DETECTED", Subject: "Alerta", BodySummary: "Alta frecuencia", Severity: models.EventWarning, AccountID: &account, Details: []byte(`{"operaciones":11,"maxOperaciones":10,"ventanaMinutos":5}`), Status: models.NotificationFailed, SentAt: time.Now()}
	if err = repo.SaveNotificationLog(ctx, warning); err != nil {
		t.Fatal(err)
	}
	if err = repo.SaveNotificationLog(ctx, warning); err != nil {
		t.Fatal(err)
	}
	info := &models.NotificationLog{ID: uuid.New(), CorrelationID: uuid.New(), Recipient: "test@example.test", NotificationType: "ACCOUNT_CREDITED", Subject: "Credito", BodySummary: "Aplicado", Status: models.NotificationSent, SentAt: time.Now()}
	if err = repo.SaveNotificationLog(ctx, info); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.GetNotificationLogs(ctx, models.NotificationFilter{Type: "ACCOUNT_ANOMALY_DETECTED", Severity: models.EventWarning, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].AccountID == nil || *rows[0].AccountID != account || len(rows[0].Details) == 0 || rows[0].Status != models.NotificationFailed {
		t.Fatal("inbox lost details or suppressed the alert on SMTP failure")
	}
}
