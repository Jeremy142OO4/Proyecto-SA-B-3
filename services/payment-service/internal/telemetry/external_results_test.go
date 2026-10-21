package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"testing"
)

func TestExternalResultsBaselineAndFirstTimeout(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { _ = provider.Shutdown(ctx); otel.SetMeterProvider(previous) })
	initializeExternalResults(ctx)
	collect := func() map[string]int64 {
		t.Helper()
		var data metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &data); err != nil {
			t.Fatal(err)
		}
		results := map[string]int64{}
		for _, scope := range data.ScopeMetrics {
			for _, instrument := range scope.Metrics {
				if instrument.Name != "bank_payment_external_results" {
					continue
				}
				for _, point := range instrument.Data.(metricdata.Sum[int64]).DataPoints {
					outcome, _ := point.Attributes.Value("outcome")
					results[outcome.AsString()] += point.Value
				}
			}
		}
		return results
	}
	baseline := collect()
	if len(baseline) != 3 {
		t.Fatalf("expected three baseline outcomes: %v", baseline)
	}
	for _, v := range baseline {
		if v != 0 {
			t.Fatal("nonzero baseline")
		}
	}
	ProveedorConfirmado(ctx, "TIMEOUT", 1)
	after := collect()
	if after["timeout"] != 1 || after["failure"] != 0 || after["success"] != 0 {
		t.Fatalf("incorrect first timeout: %v", after)
	}
	ProveedorConfirmado(ctx, "FALLO", 0.1)
	ProveedorConfirmado(ctx, "EXITO", 0.1)
	after = collect()
	if after["timeout"] != 1 || after["failure"] != 1 || after["success"] != 1 {
		t.Fatalf("incorrect results: %v", after)
	}
}
