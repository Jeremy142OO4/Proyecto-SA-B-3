package repositories_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"bank-usac/service-customer/config"
	"bank-usac/service-customer/controllers"
	"bank-usac/service-customer/events"
	"bank-usac/service-customer/internal/telemetry"
	"bank-usac/service-customer/models"
	"bank-usac/service-customer/repositories"
	"bank-usac/service-customer/routes"
	"bank-usac/service-customer/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"net/http/httptest"
	"net/url"
)

func lifecycleDB(t *testing.T) *sqlx.DB {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("isolated TEST_DATABASE_URL required")
	}
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	schema := "customer_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = db.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	db, err = sqlx.Connect("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { _, _ = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); _ = db.Close() })
	migrations, err := filepath.Glob("../database/migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range migrations {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(body)); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	return db
}

func registerLifecycleCustomer(t *testing.T, db *sqlx.DB) (*models.Customer, uuid.UUID) {
	t.Helper()
	id := uuid.New()
	svc := services.NewCustomerService(repositories.NewCustomerRepository(db), &config.Config{ActivationLinkBase: "https://example.test/activate"})
	customer, err := svc.RegisterCustomer(context.Background(), services.RegisterRequest{
		FirstName: "Private", LastName: "Customer", DocumentID: uuid.NewString(), Email: uuid.NewString() + "@private.test",
		BirthDate: "1990-01-01", Password: "private-password", DocumentPhotoURL: "https://private.test/photo", Address: "private-address",
	}, id)
	if err != nil {
		t.Fatal(err)
	}
	return customer, id
}

func TestCustomerLifecycleAtomicDecisionsRestartAndMetrics(t *testing.T) {
	db := lifecycleDB(t)
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	oldTP, oldMP := otel.GetTracerProvider(), otel.GetMeterProvider()
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
		otel.SetTracerProvider(oldTP)
		otel.SetMeterProvider(oldMP)
	})
	var output bytes.Buffer
	oldLogger, oldWriter, oldFlags := slog.Default(), log.Writer(), log.Flags()
	telemetry.ConfigureLogger(&output, "customer-service", "test", "test")
	t.Cleanup(func() { slog.SetDefault(oldLogger); log.SetOutput(oldWriter); log.SetFlags(oldFlags) })
	customer, correlationID := registerLifecycleCustomer(t, db)
	if _, err := db.Exec(`UPDATE customers SET created_at=clock_timestamp()-interval '2 hours' WHERE customer_id=$1`, customer.CustomerID); err != nil {
		t.Fatal(err)
	}
	// Simulate a fresh process/request: only the database context survives.
	svc := services.NewCustomerService(repositories.NewCustomerRepository(db), &config.Config{})
	request, requestSpan := tp.Tracer("test").Start(context.Background(), "admin KYC request")
	verified, err := svc.UpdateCustomerKYCStatus(request, customer.CustomerID, "VERIFIED", uuid.New())
	requestSpan.End()
	if err != nil {
		t.Fatal(err)
	}
	if verified.KYCVerifiedAt == nil || verified.KYCUpdatedAt == nil {
		t.Fatal("KYC timestamps missing")
	}
	first := *verified.KYCVerifiedAt
	for _, status := range []string{"VERIFIED", "REJECTED", "REJECTED", "PENDING", "VERIFIED"} {
		result, err := svc.UpdateCustomerKYCStatus(context.Background(), customer.CustomerID, status, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if !result.KYCVerifiedAt.Equal(first) {
			t.Fatal("first verification timestamp changed")
		}
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM customer_lifecycle_events WHERE customer_id=$1`, customer.CustomerID); err != nil || count != 5 {
		t.Fatalf("history count=%d err=%v", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM outbox_messages WHERE event_type=$1`, events.EventoKYCActualizado); err != nil || count != 4 {
		t.Fatalf("outbox count=%d err=%v", count, err)
	}
	var correlations []uuid.UUID
	if err := db.Select(&correlations, `SELECT DISTINCT correlation_id FROM customer_lifecycle_events WHERE customer_id=$1`, customer.CustomerID); err != nil || len(correlations) != 1 || correlations[0] != correlationID {
		t.Fatalf("lifecycle correlation changed: %v %v", correlations, err)
	}
	var messages []models.OutboxMessage
	if err := db.Select(&messages, `SELECT * FROM outbox_messages`); err != nil {
		t.Fatal(err)
	}
	expected := trace.SpanContextFromContext(telemetry.Restore(context.Background(), customer.LifecycleTraceContext)).TraceID()
	for _, message := range messages {
		var envelope events.EventEnvelope
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.CorrelationID != correlationID || message.CorrelationID != correlationID {
			t.Fatal("outbox/envelope correlation mismatch")
		}
		if trace.SpanContextFromContext(telemetry.Restore(context.Background(), message.TraceContext)).TraceID() != expected {
			t.Fatal("outbox trace split")
		}
	}
	for _, span := range exporter.GetSpans() {
		encoded, _ := json.Marshal(span.Attributes)
		if strings.Contains(string(encoded), "private") {
			t.Fatal("sensitive span attributes")
		}
	}
	if strings.Contains(output.String(), "private") || strings.Contains(output.String(), customer.DocumentID) {
		t.Fatal("sensitive data in logs")
	}
	// Repeated Saga checks must not become new KYC decisions or durations.
	for i := 0; i < 3; i++ {
		if err := repositories.NewCustomerRepository(db).RegistrarValidacionKYC(context.Background(), uuid.New(), uuid.New(), uuid.New(), customer.CustomerID); err != nil {
			t.Fatal(err)
		}
	}
	stop, err := repositories.RegisterCustomerKYCMetric(db)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	var decisions int64
	var observations uint64
	var verifiedCount int64
	for _, scope := range metrics.ScopeMetrics {
		for _, m := range scope.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				if m.Name == "bank_kyc_decisions" {
					for _, point := range data.DataPoints {
						decisions += point.Value
					}
				}
			case metricdata.Histogram[float64]:
				if m.Name == "bank_kyc_duration" {
					for _, point := range data.DataPoints {
						observations += point.Count
						if point.Sum < 7199 || point.Sum > 7205 {
							t.Fatalf("duration=%f", point.Sum)
						}
					}
				}
			case metricdata.Gauge[int64]:
				if m.Name == "bank_kyc_customers" {
					for _, point := range data.DataPoints {
						for _, attr := range point.Attributes.ToSlice() {
							if attr.Key == "status" && attr.Value.AsString() == "VERIFIED" {
								verifiedCount = point.Value
							}
						}
					}
				}
			}
		}
	}
	if decisions != 3 || observations != 1 || verifiedCount != 1 {
		t.Fatalf("decisions=%d durations=%d verified=%d", decisions, observations, verifiedCount)
	}
}

func TestCustomerLifecycleRollbackAndConcurrentRetries(t *testing.T) {
	db := lifecycleDB(t)
	customer, _ := registerLifecycleCustomer(t, db)
	repo := repositories.NewCustomerRepository(db)
	// A duplicate outbox primary key forces rollback after the state/history write.
	var duplicate uuid.UUID
	if err := db.Get(&duplicate, `SELECT id FROM outbox_messages LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	envelope, _ := events.NewEnvelope(events.EventoKYCActualizado, uuid.New(), nil, events.CustomerKYCUpdatedPayload{CustomerID: customer.CustomerID, EstadoKYC: "VERIFIED"})
	payload, _ := json.Marshal(envelope)
	_, err := repo.UpdateKYCStatusWithOutbox(context.Background(), customer.CustomerID, models.KYCVerified, &models.OutboxMessage{ID: duplicate, Payload: payload, EventType: events.EventoKYCActualizado, CreatedAt: time.Now()})
	if err == nil {
		t.Fatal("expected outbox conflict")
	}
	stored, err := repo.GetByID(context.Background(), customer.CustomerID)
	if err != nil || stored.KYCStatus != models.KYCPending || stored.KYCVerifiedAt != nil {
		t.Fatal("state was not rolled back")
	}
	var count int
	if err = db.Get(&count, `SELECT count(*) FROM customer_lifecycle_events`); err != nil || count != 1 {
		t.Fatal("history was not rolled back")
	}
	svc := services.NewCustomerService(repo, &config.Config{})
	var group sync.WaitGroup
	errorsFound := make(chan error, 8)
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := svc.UpdateCustomerKYCStatus(context.Background(), customer.CustomerID, "VERIFIED", uuid.New())
			errorsFound <- err
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = db.Get(&count, `SELECT count(*) FROM customer_lifecycle_events`); err != nil || count != 2 {
		t.Fatalf("duplicate decision events: %d %v", count, err)
	}
}

func TestCustomerLifecycleActivationAndLegacyCustomer(t *testing.T) {
	db := lifecycleDB(t)
	customer, correlation := registerLifecycleCustomer(t, db)
	repo := repositories.NewCustomerRepository(db)
	var tokenID uuid.UUID
	if err := db.Get(&tokenID, `SELECT id FROM activation_tokens WHERE customer_id=$1`, customer.CustomerID); err != nil {
		t.Fatal(err)
	}
	if err := repo.ActivateCustomer(context.Background(), customer.CustomerID, tokenID, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.ActivateCustomer(context.Background(), customer.CustomerID, tokenID, nil); err == nil {
		t.Fatal("activation token reused")
	}
	blocked, err := repo.UpdateStatusWithOutbox(context.Background(), customer.CustomerID, models.StatusBlocked, nil)
	if err != nil || blocked.LifecycleCorrelationID == nil || *blocked.LifecycleCorrelationID != correlation {
		t.Fatal("access change lost correlation")
	}
	// Legacy verified users have no trustworthy date of first verification.
	if _, err = db.Exec(`UPDATE customers SET lifecycle_correlation_id=NULL,lifecycle_trace_context='',lifecycle_observed_from_creation=FALSE,kyc_status='VERIFIED' WHERE customer_id=$1`, customer.CustomerID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateKYCStatusWithOutbox(context.Background(), customer.CustomerID, models.KYCRejected, nil); err != nil {
		t.Fatal(err)
	}
	legacy, err := repo.UpdateKYCStatusWithOutbox(context.Background(), customer.CustomerID, models.KYCVerified, nil)
	if err != nil || legacy.KYCVerifiedAt != nil {
		t.Fatal("fabricated legacy verification timestamp")
	}
}

func TestCustomerLifecycleHTTPUsesExistingProtectedKYCContract(t *testing.T) {
	db := lifecycleDB(t)
	customer, correlation := registerLifecycleCustomer(t, db)
	cfg := &config.Config{JWTSecret: "isolated-test-secret"}
	svc := services.NewCustomerService(repositories.NewCustomerRepository(db), cfg)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(telemetry.HTTP())
	routes.SetupRoutes(app, controllers.NewCustomerController(svc), cfg, func() bool { return true })
	path := "/api/v1/customers/" + customer.CustomerID.String() + "/kyc-status"
	req := httptest.NewRequest("PATCH", path, strings.NewReader(`{"kycStatus":"VERIFIED"}`))
	req.Header.Set("Content-Type", "application/json")
	unauthorized, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != 401 {
		t.Fatal("KYC endpoint must require authorization")
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "ADMIN", "sub": uuid.NewString(), "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("PATCH", path, strings.NewReader(`{"kycStatus":"VERIFIED"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Correlation-ID", uuid.NewString())
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body map[string]any
	if err = json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || body["kycStatus"] != "VERIFIED" || body["kycVerifiedAt"] == nil {
		t.Fatalf("unexpected KYC response: status=%d", res.StatusCode)
	}
	for _, key := range []string{"passwordHash", "lifecycleTraceContext", "lifecycleCorrelationId"} {
		if _, ok := body[key]; ok {
			t.Fatalf("internal field exposed: %s", key)
		}
	}
	var storedCorrelation uuid.UUID
	if err = db.Get(&storedCorrelation, `SELECT correlation_id FROM outbox_messages WHERE event_type=$1`, events.EventoKYCActualizado); err != nil || storedCorrelation != correlation {
		t.Fatal("HTTP request replaced registration correlation")
	}
}
