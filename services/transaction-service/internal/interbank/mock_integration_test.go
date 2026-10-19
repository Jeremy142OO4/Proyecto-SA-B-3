package interbank

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This test runs the original Python contract mock over real loopback HTTP,
// including its schema validation and asynchronous callback delivery.
func TestPythonContractMockDatabase(t *testing.T) {
	python := os.Getenv("INTERBANK_MOCK_PYTHON")
	if python == "" {
		t.Skip("INTERBANK_MOCK_PYTHON required")
	}
	db := testDB(t)
	ctx := context.Background()
	calls := map[string]int{}
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
		calls[c.Action]++
		state := "RESERVED"
		if c.Action == "settle" {
			state = "DEBITED"
		}
		if c.Action == "release" {
			state = "RELEASED"
		}
		if c.Action == "credit" || c.Action == "refund_received" {
			state = "CREDITED"
		}
		return AccountResult{State: state, Processed: time.Now().UTC()}, nil
	}}
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "local-callback-test-key" {
			w.WriteHeader(401)
			return
		}
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 65536))
		requestctx := propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(r.Header))
		if r.URL.Path == "/interbank/v1/transfers" {
			request, _ := json.Marshal(Request{Action: "receive", Peer: "BANK14", Key: r.Header.Get("Idempotency-Key"), Body: raw})
			reply := s.Handle(requestctx, request)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(reply.Status)
			_, _ = w.Write(reply.Body)
			return
		}
		var result Result
		if json.Unmarshal(raw, &result) != nil {
			w.WriteHeader(400)
			return
		}
		reply := s.callback(requestctx, result)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.Status)
		_, _ = w.Write(reply.Body)
	}))
	defer callback.Close()
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	root, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(python, "-m", "tools.interbank_mock.app", "--port", strconv.Itoa(port), "--database", filepath.Join(t.TempDir(), "mock.sqlite3"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "MOCK_API_KEY=local-mock-only-test-key", "MOCK_PARTNER_BASE_URL="+callback.URL+"/interbank/v1", "MOCK_PARTNER_API_KEY=local-callback-test-key")
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	base := "http://127.0.0.1:" + strconv.Itoa(port) + "/interbank/v1"
	client := &http.Client{Timeout: time.Second}
	ready := false
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		r, e := client.Get(base + "/health")
		if e == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				ready = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatal("contract mock did not start; verify Python/PyYAML")
	}
	s.Partner = NewPartner(Config{PartnerURL: base, PartnerKey: "local-mock-only-test-key"})
	tid, _ := trace.TraceIDFromHex(strings.Repeat("a", 32))
	sid, _ := trace.SpanIDFromHex(strings.Repeat("b", 16))
	tracectx := trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled}))
	for _, destination := range []string{"TEST-BANK14-001", "MISSING-ACCOUNT"} {
		tr := sample()
		tr.Origin = "BANK03"
		tr.Destination = "BANK14"
		tr.DestinationAccount = destination
		if r := s.create(tracectx, tr, "OUT", uuid.NewString()); r.Status != 202 {
			t.Fatal(r)
		}
		done := false
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
			_, e = s.Step(ctx)
			if e != nil {
				t.Fatal(e)
			}
			var state string
			_ = db.QueryRow(ctx, `SELECT estado FROM depositos_interbancarios WHERE id=$1`, tr.ID).Scan(&state)
			if state == "ACCEPTED" || state == "REVERSED" {
				want := "ACCEPTED"
				if destination == "MISSING-ACCOUNT" {
					want = "REVERSED"
				}
				if state != want {
					t.Fatalf("got %s want %s", state, want)
				}
				done = true
				break
			}
			time.Sleep(40 * time.Millisecond)
		}
		if !done {
			t.Fatal("real mock callback/settlement did not finish")
		}
	}
	if calls["reserve"] != 2 || calls["settle"] != 1 || calls["release"] != 1 {
		t.Fatalf("effects=%v", calls)
	}
	// Simulate the original sender having released its hold before a delayed
	// ACCEPTED arrives. Both banks now communicate through the real HTTP mock
	// for reversal POST/GET and the new return transfer, not synthetic callbacks.
	original := sample()
	original.Origin = "BANK03"
	original.Destination = "BANK14"
	original.DestinationAccount = "TEST-BANK14-001"
	if r := s.create(tracectx, original, "OUT", uuid.NewString()); r.Status != 202 {
		t.Fatal(r)
	}
	if _, e = db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='REVERSED' WHERE id=$1`, original.ID); e != nil {
		t.Fatal(e)
	}
	if code, _, e := s.Partner.Post(tracectx, "/transfers", original.ID, original); e != nil || code != 202 {
		t.Fatalf("late original: %d %v", code, e)
	}
	completed := false
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		if _, e = s.Step(ctx); e != nil {
			t.Fatal(e)
		}
		if _, e = s.StepReversal(ctx); e != nil {
			t.Fatal(e)
		}
		var state string
		_ = db.QueryRow(ctx, `SELECT estado FROM reversos_interbancarios WHERE original=$1`, original.ID).Scan(&state)
		if state == "COMPLETED" {
			completed = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !completed || calls["refund_received"] != 1 || calls["credit"] != 0 {
		t.Fatalf("HTTP reversal/settlement incomplete or duplicated customer funds: completed=%v effects=%v", completed, calls)
	}
}
