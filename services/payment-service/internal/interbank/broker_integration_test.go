package interbank

import (
	"context"
	"encoding/json"
	"github.com/Proyecto-SA-B-3/payment-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Requires an isolated broker; CI provisions one. No bank credentials or real accounts.
func TestPaymentBrokerHTTPPipeline(t *testing.T) {
	broker := os.Getenv("TEST_AMQP_URL")
	if broker == "" {
		t.Skip("TEST_AMQP_URL required: isolated broker only")
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{9}, SpanID: trace.SpanID{8}, TraceFlags: trace.FlagsSampled})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(context.Background(), propagation.HeaderCarrier(r.Header))
		if trace.SpanContextFromContext(ctx).TraceID() != sc.TraceID() {
			t.Error("AMQP to HTTP lost Trace ID")
		}
		if r.Header.Get("X-Api-Key") != "local-pipeline-only-key" {
			t.Error("partner key missing")
		}
		if r.Method == "POST" && r.URL.Path == "/interbank/v1/transfers" {
			if r.Header.Get("Idempotency-Key") != testID {
				t.Error("stable idempotency key lost")
			}
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`{"status":"PROCESSING"}`))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer server.Close()
	transport := &Transport{BaseURL: server.URL + "/interbank/v1", Key: "local-pipeline-only-key", Client: &http.Client{Transport: telemetry.Transport{Base: server.Client().Transport}, Timeout: 5 * time.Second}}
	queue := "test.payment.interbank." + uuid.NewString()
	conn, err := amqp.Dial(broker)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	defer ch.QueueDelete(queue, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); bankrpc.Run(ctx, broker, queue, transport.Handle) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("consumer did not stop")
		}
	}()
	ready := false
	for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
		response, e := bankrpc.Call(ctx, broker, queue, Request{Method: "GET", Path: "/health"})
		if e == nil && response.Status == 400 {
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatal("payment consumer not ready")
	}
	for _, r := range []Request{
		{Method: "POST", Path: "/transfers", Key: testID, Body: json.RawMessage(`{"interbankTransferId":"` + testID + `"}`)},
		{Method: "POST", Path: "/transfers/" + testID + "/result", Body: json.RawMessage("{}")},
		{Method: "POST", Path: "/transfers/" + testID + "/cancel", Body: json.RawMessage("{}")},
		{Method: "GET", Path: "/transfers/" + testID},
		{Method: "POST", Path: "/reversals", Body: json.RawMessage("{}")},
		{Method: "GET", Path: "/reversals/" + testID},
		{Method: "GET", Path: "/reconciliation?date=2026-10-22&page=1&pageSize=100"},
	} {
		response, e := bankrpc.Call(trace.ContextWithSpanContext(ctx, sc), broker, queue, r)
		if e != nil {
			t.Fatal(e)
		}
		var result Result
		if e = json.Unmarshal(response.Body, &result); e != nil || response.Status != 200 || result.Error != "" {
			t.Fatalf("%+v %v", response, e)
		}
		expected := 200
		if r.Path == "/transfers" {
			expected = 202
		}
		if result.Status != expected {
			t.Fatal(result)
		}
	}
}
