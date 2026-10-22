package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testID = "11111111-1111-4111-8111-111111111111"

func invoke(p *Transport, r Request) (int, Result) {
	raw, _ := json.Marshal(r)
	reply := p.Handle(context.Background(), raw)
	var result Result
	_ = json.Unmarshal(reply.Body, &result)
	return reply.Status, result
}
func TestAllowedOperations(t *testing.T) {
	for _, r := range []Request{
		{Method: "POST", Path: "/transfers", Key: testID, Body: json.RawMessage("{}")},
		{Method: "GET", Path: "/transfers/" + testID},
		{Method: "POST", Path: "/transfers/" + testID + "/result", Body: json.RawMessage("{}")},
		{Method: "POST", Path: "/transfers/" + testID + "/cancel", Body: json.RawMessage("{}")},
		{Method: "POST", Path: "/reversals", Body: json.RawMessage("{}")},
		{Method: "GET", Path: "/reversals/" + testID},
		{Method: "GET", Path: "/reconciliation?date=2026-10-22&page=1&pageSize=100"},
	} {
		t.Run(r.Method+r.Path, func(t *testing.T) {
			if !valid(r) {
				t.Fatal("required contract operation rejected")
			}
		})
	}
	for _, path := range []string{"https://evil.test/reconciliation", "//evil.test/reconciliation", "/transfers/../../health", "/reconciliation?url=https://evil.test", "/health", "/transfers/" + testID + "/result?secret=x"} {
		if valid(Request{Method: "GET", Path: path}) {
			t.Fatal("unsafe path accepted", path)
		}
	}
}
func TestHTTPSConfiguration(t *testing.T) {
	t.Setenv("INTERBANK_PARTNER_API_KEY", "test-only-partner-key")
	for _, raw := range []string{"http://bank14.test/interbank/v1", "https://u:p@bank14.test/interbank/v1", "https://bank14.test/wrong", "https://bank14.test/interbank/v1?key=x", "https://bank14.test/interbank/v1#frag"} {
		t.Setenv("INTERBANK_PARTNER_BASE_URL", raw)
		if _, err := New(); err == nil {
			t.Fatal(raw)
		}
	}
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "https://bank14.test/interbank/v1")
	if _, err := New(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INTERBANK_TEST_MODE", "true")
	t.Setenv("ENTORNO", "interbank-sandbox")
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "http://127.0.0.1:8099/interbank/v1")
	if _, err := New(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENTORNO", "dev")
	if _, err := New(); err == nil {
		t.Fatal("HTTP allowed outside isolated laboratory")
	}
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "")
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}
	_, r := invoke(p, Request{Method: "GET", Path: "/transfers/" + testID})
	if r.Error != "not_configured" {
		t.Fatal(r)
	}
}
func TestHTTPContractTraceAndNoRedirect(t *testing.T) {
	calls := 0
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect leaked request") }))
	defer target.Close()
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Api-Key") != "test-only-partner-key" || r.Header.Get("Idempotency-Key") != testID {
			t.Error("lost authentication or idempotency")
		}
		extracted := propagation.TraceContext{}.Extract(context.Background(), propagation.HeaderCarrier(r.Header))
		if trace.SpanContextFromContext(extracted).TraceID() != sc.TraceID() {
			t.Error("distributed Trace ID lost")
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"interbankTransferId":"`+testID+`","amount":"100.00"}` {
			t.Error("external body changed")
		}
		if calls == 1 {
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`{"status":"PROCESSING"}`))
			return
		}
		http.Redirect(w, r, target.URL, 307)
	}))
	defer server.Close()
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "https://bank14.test/interbank/v1")
	t.Setenv("INTERBANK_PARTNER_API_KEY", "test-only-partner-key")
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}
	p.BaseURL = server.URL + "/interbank/v1"
	// Keep the production instrumented transport, using only the test CA.
	p.Client.Transport = telemetry.Transport{Base: server.Client().Transport}
	raw := json.RawMessage(`{"method":"POST","path":"/transfers","key":"` + testID + `","body":{"interbankTransferId":"` + testID + `","amount":"100.00"}}`)
	for _, want := range []int{202, 307} {
		reply := p.Handle(trace.ContextWithSpanContext(context.Background(), sc), raw)
		var result Result
		_ = json.Unmarshal(reply.Body, &result)
		if reply.Status != 200 || result.Status != want || result.Error != "" {
			t.Fatal(reply, string(reply.Body))
		}
	}
	if calls != 2 {
		t.Fatal("HTTP transport retried unexpectedly")
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("sensitive upstream failure")
}
func TestFailuresAreBoundedAndSanitized(t *testing.T) {
	p := &Transport{BaseURL: "https://bank14.test/interbank/v1", Key: "test-only-partner-key", Client: &http.Client{Transport: failingTransport{}}}
	_, r := invoke(p, Request{Method: "GET", Path: "/transfers/" + testID})
	if r.Error != "request_failed" || strings.Contains(r.Error, "sensitive") {
		t.Fatal(r)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(strings.Repeat("x", 65537))) }))
	defer server.Close()
	p.Client = server.Client()
	p.BaseURL = server.URL
	_, r = invoke(p, Request{Method: "GET", Path: "/transfers/" + testID})
	if r.Error != "response_failed" {
		t.Fatal("oversized response accepted", r)
	}
	reply := p.Handle(context.Background(), json.RawMessage(`{"method":"GET","path":"/health","url":"https://evil.test"}`))
	if reply.Status != 400 {
		t.Fatal("invalid command accepted")
	}
}
