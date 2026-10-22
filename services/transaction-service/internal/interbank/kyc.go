package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"os"
)

func simulationEnabled() bool {
	return os.Getenv("ENTORNO") == "dev" || os.Getenv("ENTORNO") == "interbank-sandbox"
}
func (s *Service) verifyKYC(ctx context.Context, id, customer string) (bool, error) {
	if s.KYC != nil {
		return s.KYC(ctx, id, customer)
	}
	r, e := bankrpc.Call(ctx, s.Config.RabbitURL, "clientes.interbancarias.kyc", map[string]string{"customer": customer, "id": id})
	if e != nil {
		return false, e
	}
	if r.Status != 200 {
		return false, errors.New("KYC unavailable")
	}
	var result struct {
		Verified bool `json:"verified"`
	}
	e = json.Unmarshal(r.Body, &result)
	return result.Verified, e
}
