package interbank

// Test-only direct adapter isolates coordinator/database tests. Production uses BrokerPartner.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"io"
	"net/http"
	"time"
)

func (p *HTTPPartner) Ready() bool { return p.BaseURL != "" && len(p.Key) >= 16 }

type HTTPPartner struct {
	BaseURL, Key string
	Client       *http.Client
}

func newHTTPTestPartner(c Config) *HTTPPartner {
	return &HTTPPartner{BaseURL: c.PartnerURL, Key: c.PartnerKey, Client: &http.Client{Timeout: 5 * time.Second, Transport: telemetry.Transport{}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (p *HTTPPartner) Post(ctx context.Context, path, key string, body any) (int, []byte, error) {
	b, e := json.Marshal(body)
	if e != nil {
		return 0, nil, e
	}
	return p.request(ctx, "POST", path, key, b)
}
func (p *HTTPPartner) Get(ctx context.Context, path string) (int, []byte, error) {
	return p.request(ctx, "GET", path, "", nil)
}
func (p *HTTPPartner) request(ctx context.Context, method, path, key string, b []byte) (int, []byte, error) {
	if !p.Ready() {
		return 0, nil, ErrPartnerNotConfigured
	}
	req, e := http.NewRequestWithContext(ctx, method, p.BaseURL+path, bytes.NewReader(b))
	if e != nil {
		return 0, nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", p.Key)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, e := p.Client.Do(req)
	if e != nil {
		return 0, nil, e
	}
	defer resp.Body.Close()
	data, e := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if len(data) > 65536 {
		return 0, nil, errors.New("partner response too large")
	}
	return resp.StatusCode, data, e
}
