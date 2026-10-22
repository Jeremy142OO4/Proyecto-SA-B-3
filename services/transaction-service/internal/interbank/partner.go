package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
)

// Partner transport is exclusively AMQP. Payment owns the external HTTP client.
type BrokerPartner struct {
	Config Config
	Call   func(context.Context, string, string, any) (bankrpc.Response, error)
}
type transportRequest struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Key    string          `json:"key,omitempty"`
	Body   json.RawMessage `json:"body,omitempty"`
}
type transportResult struct {
	Status int    `json:"status"`
	Body   []byte `json:"body"`
	Error  string `json:"error,omitempty"`
}

func NewPartner(c Config) *BrokerPartner { return &BrokerPartner{Config: c, Call: bankrpc.Call} }
func (p *BrokerPartner) Ready() bool {
	return p.Config.PartnerURL != "" && len(p.Config.PartnerKey) >= 16
}
func (p *BrokerPartner) Post(ctx context.Context, path, key string, body any) (int, []byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	return p.request(ctx, transportRequest{Method: "POST", Path: path, Key: key, Body: b})
}
func (p *BrokerPartner) Get(ctx context.Context, path string) (int, []byte, error) {
	return p.request(ctx, transportRequest{Method: "GET", Path: path})
}
func (p *BrokerPartner) request(ctx context.Context, request transportRequest) (int, []byte, error) {
	if !p.Ready() {
		return 0, nil, ErrPartnerNotConfigured
	}
	response, err := p.Call(ctx, p.Config.RabbitURL, "pagos.interbancarios.salida", request)
	if err != nil {
		return 0, nil, err
	}
	if response.Status != 200 {
		return 0, nil, errors.New("payment interbank transport unavailable")
	}
	var result transportResult
	if err = json.Unmarshal(response.Body, &result); err != nil {
		return 0, nil, err
	}
	if result.Error == "not_configured" {
		return 0, nil, ErrPartnerNotConfigured
	}
	if result.Error != "" {
		return 0, nil, errors.New("payment interbank HTTP request failed")
	}
	return result.Status, result.Body, nil
}
