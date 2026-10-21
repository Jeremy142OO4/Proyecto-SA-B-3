package telemetry

import (
	"context"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"testing"
)

func TestTransferResultsExportBaselineBeforeFirstSuccess(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	old := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { _ = provider.Shutdown(ctx); otel.SetMeterProvider(old) })
	initializeTransferResults(ctx)
	collect := func() map[string]int64 {
		t.Helper()
		var data metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &data); err != nil {
			t.Fatal(err)
		}
		counts := map[string]int64{}
		for _, scope := range data.ScopeMetrics {
			for _, instrument := range scope.Metrics {
				if instrument.Name != "bank_transfer_results" {
					continue
				}
				for _, point := range instrument.Data.(metricdata.Sum[int64]).DataPoints {
					outcome, _ := point.Attributes.Value("outcome")
					counts[outcome.AsString()] += point.Value
				}
			}
		}
		return counts
	}
	baseline := collect()
	if len(baseline) != 2 || baseline["success"] != 0 || baseline["failure"] != 0 {
		t.Fatalf("missing zero baseline: %v", baseline)
	}
	TransferCommitted(ctx, "synthetic-operation", "00000000-0000-4000-8000-000000000001", "", "PROCESANDO", 0)
	after := collect()
	if after["success"] != 0 || after["failure"] != 0 {
		t.Fatal("pending transfer counted as terminal")
	}
	TransferCommitted(ctx, "synthetic-operation", "00000000-0000-4000-8000-000000000001", "", "COMPLETADA", 0)
	after = collect()
	if after["success"] != 1 || after["failure"] != 0 {
		t.Fatalf("first success lost: %v", after)
	}
}
