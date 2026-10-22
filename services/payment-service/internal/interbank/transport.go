// Package interbank owns all outbound bank HTTP traffic. Transaction coordinates
// durable retries and financial state; this service never retries or changes money.
package interbank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Proyecto-SA-B-3/payment-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const Queue = "pagos.interbancarios.salida"

type Request struct {
	Scenario    string          `json:"scenario,omitempty"`
	Correlation string          `json:"idCorrelacion,omitempty"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Key         string          `json:"key,omitempty"`
	Body        json.RawMessage `json:"body,omitempty"`
}
type Result struct {
	Status int    `json:"status"`
	Body   []byte `json:"body"`
	Error  string `json:"error,omitempty"`
}
type Transport struct {
	BaseURL, Key string
	Client       *http.Client
	Mock         *Service
}

func New() (*Transport, error) {
	raw := strings.TrimRight(os.Getenv("INTERBANK_PARTNER_BASE_URL"), "/")
	key := os.Getenv("INTERBANK_PARTNER_API_KEY")
	if raw != "" {
		u, err := url.Parse(raw)
		sandbox := os.Getenv("INTERBANK_TEST_MODE") == "true" && os.Getenv("ENTORNO") == "interbank-sandbox"
		loopback := err == nil && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")
		if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/interbank/v1" || strings.HasSuffix(u.Hostname(), "example.com") || !(u.Scheme == "https" || (sandbox && loopback && u.Scheme == "http")) {
			return nil, errors.New("invalid interbank partner HTTPS URL")
		}
	}
	if key != "" && len(key) < 16 {
		return nil, errors.New("invalid interbank partner secret")
	}
	return &Transport{BaseURL: raw, Key: key, Client: &http.Client{Timeout: 5 * time.Second, Transport: telemetry.Transport{}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

var transferPath = regexp.MustCompile(`^/transfers/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(/result|/cancel)?$`)
var reversalPath = regexp.MustCompile("^/reversals/[0-9a-fA-F-]{36}$")

func valid(r Request) bool {
	if r.Method == "POST" && r.Path == "/transfers" {
		return len(r.Body) > 0 && r.Key != ""
	}
	if r.Method == "POST" && r.Path == "/reversals" {
		return len(r.Body) > 0
	}
	if r.Method == "GET" && reversalPath.MatchString(r.Path) {
		return true
	}
	if transferPath.MatchString(r.Path) {
		return (r.Method == "GET" && !strings.HasSuffix(r.Path, "/result") && !strings.HasSuffix(r.Path, "/cancel")) ||
			(r.Method == "POST" && (strings.HasSuffix(r.Path, "/result") || strings.HasSuffix(r.Path, "/cancel")))
	}
	if r.Method == "GET" {
		u, e := url.Parse(r.Path)
		if e != nil || u.Path != "/reconciliation" || u.Fragment != "" || u.Host != "" || u.Scheme != "" {
			return false
		}
		for k := range u.Query() {
			if k != "date" && k != "page" && k != "pageSize" {
				return false
			}
		}
		return true
	}
	return false
}
func (p *Transport) Handle(ctx context.Context, raw json.RawMessage) bankrpc.Response {
	var r Request
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&r) != nil || decoder.Decode(new(any)) != io.EOF || !valid(r) {
		return bankrpc.Reply(400, map[string]string{"error": "invalid interbank transport command"})
	}
	if r.Scenario != "" {
		if os.Getenv("ENTORNO") != "dev" || p.Mock == nil {
			return bankrpc.Reply(403, nil)
		}
		switch r.Scenario {
		case "reject", "unavailable", "no_callback", "delayed":
		default:
			return bankrpc.Reply(400, nil)
		}
		response := p.Mock.mock(ctx, r)
		return bankrpc.Reply(200, Result{Status: response.Status, Body: response.Body})
	}
	if p.BaseURL == "" || len(p.Key) < 16 {
		return bankrpc.Reply(200, Result{Error: "not_configured"})
	}
	ctx = telemetry.WithCorrelation(ctx, r.Correlation)
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "interbank partner HTTP")
	span.SetAttributes(attribute.String("interbank.transport.owner", "payment-service"))
	// Copy only the opaque transfer ID into telemetry, never account numbers or keys.
	id := ""
	if match := transferPath.FindStringSubmatch(r.Path); len(match) > 1 {
		id = match[1]
	}
	if id == "" && len(r.Body) > 0 {
		var body struct {
			ID string `json:"interbankTransferId"`
		}
		_ = json.Unmarshal(r.Body, &body)
		id = body.ID
	}
	if id != "" {
		span.SetAttributes(attribute.String("transaction.id", id))
	}
	defer span.End()
	req, err := http.NewRequestWithContext(ctx, r.Method, p.BaseURL+r.Path, bytes.NewReader(r.Body))
	if err != nil {
		return bankrpc.Reply(200, Result{Error: "request_failed"})
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", p.Key)
	if r.Key != "" {
		req.Header.Set("Idempotency-Key", r.Key)
	}
	response, err := p.Client.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "interbank partner HTTP failed", "interbankTransferId", id, "method", r.Method)
		return bankrpc.Reply(200, Result{Error: "request_failed"})
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(body) > 65536 {
		return bankrpc.Reply(200, Result{Error: "response_failed"})
	}
	slog.InfoContext(ctx, "interbank partner HTTP completed", "interbankTransferId", id, "method", r.Method, "httpStatus", response.StatusCode)
	return bankrpc.Reply(200, Result{Status: response.StatusCode, Body: body})
}
