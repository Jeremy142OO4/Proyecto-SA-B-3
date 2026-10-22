package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"strings"
	"testing"
)

func TestBrokerPartnerPreservesWireContract(t *testing.T) {
	c := Config{PartnerURL: "https://bank14.test/interbank/v1", PartnerKey: "test-only-partner-key", RabbitURL: "amqp://isolated"}
	p := NewPartner(c)
	response := []byte(`{"status":"PROCESSING"}`)
	calls := 0
	p.Call = func(ctx context.Context, url, queue string, body any) (bankrpc.Response, error) {
		calls++
		if url != c.RabbitURL || queue != "pagos.interbancarios.salida" {
			t.Fatal("request bypassed payment queue")
		}
		raw, _ := json.Marshal(body)
		if len(raw) == 0 || strings.Contains(string(raw), c.PartnerKey) {
			t.Fatal("invalid body")
		}
		var r transportRequest
		_ = json.Unmarshal(raw, &r)
		if r.Correlation != "11111111-1111-4111-8111-111111111111" {
			t.Fatal("operation correlation lost over AMQP")
		}
		if r.Method != "POST" || r.Path != "/transfers" || r.Key != "stable-id" || string(r.Body) != `{"amount":"100.00"}` {
			t.Fatalf("%+v", r)
		}
		return bankrpc.Reply(200, transportResult{Status: 202, Body: response}), nil
	}
	for i := 0; i < 2; i++ {
		code, body, err := p.Post(telemetry.WithCorrelation(context.Background(), "11111111-1111-4111-8111-111111111111"), "/transfers", "stable-id", map[string]string{"amount": "100.00"})
		if err != nil || code != 202 || string(body) != string(response) {
			t.Fatalf("%d %s %v", code, body, err)
		}
	}
	if calls != 2 {
		t.Fatal("transport retried or dropped a command")
	}
}
func TestBrokerPartnerErrorsAndReads(t *testing.T) {
	p := NewPartner(Config{PartnerURL: "https://bank14.test/interbank/v1", PartnerKey: "test-only-partner-key"})
	p.Call = func(_ context.Context, _, _ string, body any) (bankrpc.Response, error) {
		r := body.(transportRequest)
		if r.Method != "GET" || r.Path != "/reconciliation?date=2026-10-22&page=1&pageSize=100" {
			t.Fatal(r)
		}
		return bankrpc.Reply(200, transportResult{Error: "not_configured"}), nil
	}
	_, _, err := p.Get(context.Background(), "/reconciliation?date=2026-10-22&page=1&pageSize=100")
	if !errors.Is(err, ErrPartnerNotConfigured) {
		t.Fatal(err)
	}
	p.Call = func(context.Context, string, string, any) (bankrpc.Response, error) {
		return bankrpc.Response{}, errors.New("broker unavailable")
	}
	if _, _, err = p.Get(context.Background(), "/transfers/id"); err == nil {
		t.Fatal("broker outage must not look like successful HTTP")
	}
}
