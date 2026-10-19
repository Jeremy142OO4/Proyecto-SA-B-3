package telemetry

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestCustomerLifecycleResumesRegistrationAndLinksNewRequest(t *testing.T) {
	tp, exporter := tracing(t)
	id := uuid.New()
	ctx, created := StartCustomerLifecycle(context.Background(), "", "customer.lifecycle.created", id)
	saved, expected := Serialize(ctx), created.SpanContext().TraceID()
	created.End()
	request, requestSpan := tp.Tracer("test").Start(context.Background(), "later KYC request")
	defer requestSpan.End()
	resumed, verified := StartCustomerLifecycle(request, saved, "customer.lifecycle.kyc", id)
	CustomerLifecycleCommitted(resumed, uuid.New(), "cliente.kyc.estado.actualizado", "KYC", "PENDING", "VERIFIED", nil)
	verified.End()
	spans := exporter.GetSpans()
	last := spans[len(spans)-1]
	if last.SpanContext.TraceID() != expected || len(last.Links) != 1 || !last.Links[0].SpanContext.Equal(requestSpan.SpanContext()) {
		t.Fatal("registration trace or request link lost")
	}
	if len(last.Events) != 1 {
		t.Fatal("missing committed lifecycle event")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	resumed, span := StartCustomerLifecycle(cancelled, saved, "cancelled request", id)
	defer span.End()
	if resumed.Err() != context.Canceled {
		t.Fatal("request cancellation was lost")
	}
}

func TestCustomerLifecycleMetricUnitsAndPrivacy(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	old := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()); otel.SetMeterProvider(old) })
	var output bytes.Buffer
	oldLogger, oldWriter, oldFlags := slog.Default(), log.Writer(), log.Flags()
	t.Cleanup(func() { slog.SetDefault(oldLogger); log.SetOutput(oldWriter); log.SetFlags(oldFlags) })
	ConfigureLogger(&output, "customer-service", "test", "test")
	ctx := WithCorrelation(context.Background(), uuid.NewString())
	duration := 2 * time.Hour
	CustomerLifecycleCommitted(ctx, uuid.New(), "cliente.kyc.estado.actualizado", "KYC", "PENDING", "VERIFIED", &duration)
	CustomerLifecycleCommitted(ctx, uuid.New(), "cliente.kyc.estado.actualizado", "KYC", "PENDING", "REJECTED", nil)
	CustomerLifecycleCommitted(ctx, uuid.New(), "cliente.estado.actualizado", "ACCESS", "ACTIVO", "BLOQUEADO", nil)
	slog.InfoContext(ctx, "redaction test", "correo", "private-email", "documento", "private-document",
		"fotografiaDocumento", "private-photo", "jwt", "private-jwt", "dpi", "private-dpi",
		slog.Group("nested", "documentPhotoUrl", "private-nested-photo"))
	if strings.Contains(output.String(), "private-") {
		t.Fatal("customer data leaked into logs")
	}
	if !strings.Contains(output.String(), `"level":"WARNING"`) {
		t.Fatal("rejection warning missing")
	}
	var result metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &result); err != nil {
		t.Fatal(err)
	}
	var decisions int64
	var observations uint64
	for _, scope := range result.ScopeMetrics {
		for _, m := range scope.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				if m.Name != "bank_kyc_decisions" {
					continue
				}
				for _, point := range data.DataPoints {
					decisions += point.Value
					if strings.Contains(point.Attributes.Encoded(nil), "customerId") {
						t.Fatal("high cardinality labels")
					}
				}
			case metricdata.Histogram[float64]:
				if m.Name != "bank_kyc_duration" {
					continue
				}
				if m.Unit != "s" {
					t.Fatal("histogram unit must be seconds")
				}
				for _, point := range data.DataPoints {
					observations += point.Count
					if point.Sum != 7200 {
						t.Fatalf("incorrect duration: %v", point.Sum)
					}
				}
			}
		}
	}
	if decisions != 2 || observations != 1 {
		t.Fatalf("decisions=%d observations=%d", decisions, observations)
	}
}
