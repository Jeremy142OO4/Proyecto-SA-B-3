package interbank

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
)

func TestKYCRejectedDoesNotReserveDatabase(t *testing.T) {
	db := testDB(t)
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: &fakePartner{}, KYC: func(context.Context, string, string) (bool, error) { return false, nil }, Account: func(context.Context, AccountCommand) (AccountResult, error) {
		t.Fatal("KYC rejected must not reserve funds")
		return AccountResult{}, nil
	}}
	tr := sample()
	r := invoke(s, Request{Action: "send", Key: tr.ID, Customer: uuid.NewString(), Body: payload(tr)})
	if r.Status != 202 {
		t.Fatal(r)
	}
	if _, err := s.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	err := db.QueryRow(context.Background(), "SELECT estado,motivo FROM depositos_interbancarios WHERE id=$1", tr.ID).Scan(&state, &reason)
	if err != nil || state != "REJECTED" || reason != "KYC_NOT_VERIFIED" {
		t.Fatal(state, reason, err)
	}
	r = invoke(s, Request{Action: "client_list", Customer: uuid.NewString()})
	var rows []any
	if json.Unmarshal(r.Body, &rows) != nil || len(rows) != 0 {
		t.Fatal("history leaked another customer's deposits")
	}
}
