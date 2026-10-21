package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"testing"
)

func TestBusinessStateLogIncludesOriginAccount(t *testing.T) {
	old, writer, flags := slog.Default(), log.Writer(), log.Flags()
	defer func() { slog.SetDefault(old); log.SetOutput(writer); log.SetFlags(flags) }()
	var output bytes.Buffer
	ConfigureLogger(&output, "transaction-service", "test", "test")
	ctx := context.Background()
	accountID := "00000000-0000-4000-8000-000000000002"
	TransferCommitted(ctx, "synthetic-operation", "00000000-0000-4000-8000-000000000001", "", "COMPLETADA", 0, accountID)
	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["accountId"] != accountID {
		t.Fatalf("missing accountId: %v", record)
	}
	if record["state"] != "COMPLETADA" {
		t.Fatalf("incorrect state: %v", record)
	}
}
