package interbank

import (
	"bank-usac/service-customer/internal/bankrpc"
	"bank-usac/service-customer/internal/telemetry"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type KYC struct{ DB *sqlx.DB }

func (k KYC) Handle(ctx context.Context, raw json.RawMessage) bankrpc.Response {
	var r struct {
		Customer string `json:"customer"`
		ID       string `json:"id"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return bankrpc.Reply(400, nil)
	}
	if id, e := uuid.Parse(r.Customer); e != nil || id == uuid.Nil {
		return bankrpc.Reply(400, nil)
	}
	ctx = telemetry.WithCorrelation(ctx, r.ID)
	ctx, finish := telemetry.Database(ctx, "InterbankValidateKYC")
	defer finish()
	var active, kyc string
	err := k.DB.QueryRowContext(ctx, "SELECT status,kyc_status FROM customers WHERE customer_id=$1", r.Customer).Scan(&active, &kyc)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return bankrpc.Reply(503, nil)
	}
	return bankrpc.Reply(200, map[string]any{"verified": err == nil && active == "ACTIVO" && kyc == "VERIFIED"})
}
