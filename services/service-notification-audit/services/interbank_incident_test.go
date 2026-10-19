package services

import (
	"bank-usac/service-notification-audit/models"
	"encoding/json"
	"testing"
)

func TestInterbankFinancialIncidentsAlwaysError(t *testing.T) {
	for _, code := range []string{"LATE_ACCEPTED_AFTER_RELEASE", "RECONCILIATION_AMOUNT_MISMATCH", "REFUND_FAILED", "TIMEOUT_MARGIN_RELEASE"} {
		payload, _ := json.Marshal(map[string]string{"codigo": code, "nivel": "ERROR"})
		if got := ClassifyEventWithPayload("interbancaria.incidente", payload); got != models.EventError {
			t.Fatalf("%s classified %s", code, got)
		}
	}
}
