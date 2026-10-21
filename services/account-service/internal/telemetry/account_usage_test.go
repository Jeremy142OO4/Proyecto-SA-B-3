package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAccountMovementsByAccountTypeAndDirection(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { _ = provider.Shutdown(ctx); otel.SetMeterProvider(previous) })
	initializeAccountUsage(ctx)
	var baseline metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &baseline); err != nil {
		t.Fatal(err)
	}
	baselineSeries := 0
	for _, scope := range baseline.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			if instrument.Name != "bank_account_movements" {
				continue
			}
			for _, point := range instrument.Data.(metricdata.Sum[int64]).DataPoints {
				baselineSeries++
				if point.Value != 0 {
					t.Fatalf("baseline must be zero, got %d", point.Value)
				}
			}
		}
	}
	if baselineSeries != 6 {
		t.Fatalf("expected six initialized series, got %d", baselineSeries)
	}
	for _, accountType := range []string{"AHORRO", "CORRIENTE"} {
		for _, direction := range []string{"DEBITO", "CREDITO"} {
			AccountMovementCommitted(ctx, accountType, direction, "synthetic-account", false)
		}
	}
	// A compensation must not inflate the debit/credit operation counter.
	AccountMovementCommitted(ctx, "AHORRO", "COMPENSACION", "synthetic-account", false)
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int64{}
	for _, scope := range collected.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			if instrument.Name != "bank_account_movements" {
				continue
			}
			for _, point := range instrument.Data.(metricdata.Sum[int64]).DataPoints {
				accountType, _ := point.Attributes.Value("account_type")
				direction, _ := point.Attributes.Value("direction")
				seen[accountType.AsString()+"/"+direction.AsString()] += point.Value
			}
		}
	}
	if len(seen) != 6 {
		t.Fatalf("expected six series including legacy zero series, got %v", seen)
	}
	for _, key := range []string{"AHORRO/DEBITO", "AHORRO/CREDITO", "CORRIENTE/DEBITO", "CORRIENTE/CREDITO"} {
		if seen[key] != 1 {
			t.Errorf("%s: expected 1, got %d", key, seen[key])
		}
	}
}
