package interbank

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestRequestedRefundNormalTransferDatabase(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "returned", true: "insufficient funds"}[failure], func(t *testing.T) {
			db := testDB(t)
			ctx := context.Background()
			effects := map[string]int{}
			receipts := map[string]string{}
			s := &Service{KYC: func(context.Context, string, string) (bool, error) { return true, nil }, DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
				state := receipts[c.ID]
				switch c.Action {
				case "credit":
					if state == "" {
						state = "CREDITED"
						effects["credit"]++
					}
				case "refund_reserve":
					if state == "" {
						state = "RESERVED"
						if failure {
							state = "REJECTED"
						}
						effects["refund_reserve"]++
					}
				case "settle":
					if state == "RESERVED" {
						state = "DEBITED"
						effects["debit"]++
					}
				}
				receipts[c.ID] = state
				r := AccountResult{State: state, Processed: time.Now().UTC()}
				if state == "REJECTED" {
					r.Reason = "INSUFFICIENT_FUNDS"
				}
				return r, nil
			}}
			s.Partner = recoveryPartner{post: func(path string, body any) (int, []byte, error) {
				if tr, ok := body.(Transfer); ok {
					return response(200, map[string]any{"interbankTransferId": tr.ID, "status": "ACCEPTED"})
				}
				r := body.(*Result)
				return response(200, map[string]any{"interbankTransferId": r.ID, "received": true})
			}}
			tr := sample()
			if r := s.create(ctx, tr, "IN", ""); r.Status != 202 {
				t.Fatal(r)
			}
			if _, e := s.Step(ctx); e != nil {
				t.Fatal(e)
			}
			request := Reversal{ID: uuid.NewString(), Original: tr.ID, Origin: "BANK14", Amount: tr.Amount, Currency: "GTQ"}
			b, _ := json.Marshal(request)
			r := Request{Action: "reversal", Peer: "BANK14", Body: b}
			first := invoke(s, r)
			if first.Status != 202 {
				t.Fatalf("%d %s", first.Status, first.Body)
			}
			if duplicate := invoke(s, r); duplicate.Status != 202 {
				t.Fatal(duplicate)
			}
			var ack struct {
				Refund string `json:"refundTransferId"`
			}
			_ = json.Unmarshal(first.Body, &ack)
			if ack.Refund == "" || ack.Refund == tr.ID {
				t.Fatal("refund id not available while processing")
			}
			changed := request
			changed.ID = uuid.NewString()
			b, _ = json.Marshal(changed)
			if reply := invoke(s, Request{Action: "reversal", Peer: "BANK14", Body: b}); reply.Status != 400 {
				t.Fatal("two refund requests for same original accepted")
			}
			for i := 0; i < 8; i++ {
				_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp()`)
				_, _ = db.Exec(ctx, `UPDATE reversos_interbancarios SET proximo=clock_timestamp()`)
				if _, e := s.StepReversal(ctx); e != nil {
					t.Fatal(e)
				}
				if _, e := s.Step(ctx); e != nil {
					t.Fatal(e)
				}
			}
			row, e := scanReversal(db.QueryRow(ctx, `SELECT `+reversalColumns+` FROM reversos_interbancarios WHERE id=$1`, request.ID))
			if e != nil {
				t.Fatal(e)
			}
			want := "COMPLETED"
			debits := 1
			if failure {
				want = "FAILED"
				debits = 0
			}
			if row.State != want || effects["refund_reserve"] != 1 || effects["debit"] != debits {
				t.Fatalf("state=%s effects=%v", row.State, effects)
			}
			refund, e := scan(db.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, ack.Refund))
			if e != nil || refund.RefundOf != tr.ID || refund.Transfer.OriginAccount != tr.DestinationAccount || refund.Transfer.DestinationAccount != tr.OriginAccount {
				t.Fatalf("refund not a normal return deposit: %+v %v", refund, e)
			}
		})
	}
}

func TestExpectedRefundIsSettlementNotSecondClientCreditDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	refundID := uuid.NewString()
	effects := map[string]int{}
	s := &Service{KYC: func(context.Context, string, string) (bool, error) { return true, nil }, DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
		effects[c.Action]++
		return AccountResult{State: "CREDITED", Processed: time.Now().UTC()}, nil
	}}
	original := sample()
	original.Origin = "BANK03"
	original.Destination = "BANK14"
	if r := s.create(ctx, original, "OUT", uuid.NewString()); r.Status != 202 {
		t.Fatal(r)
	}
	_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='REVERSED' WHERE id=$1`, original.ID)
	if r := s.callback(ctx, Result{ID: original.ID, Destination: "BANK14", Status: "ACCEPTED", Processed: time.Now().UTC()}); r.Status != 200 {
		t.Fatal(r)
	}
	var reversalID string
	_ = db.QueryRow(ctx, `SELECT id::text FROM reversos_interbancarios`).Scan(&reversalID)
	s.Partner = recoveryPartner{post: func(path string, body any) (int, []byte, error) {
		return response(202, map[string]any{"reversalId": reversalID, "status": "PROCESSING", "refundTransferId": refundID})
	}, get: func(path string) (int, []byte, error) {
		return response(200, map[string]any{"reversalId": reversalID, "status": "COMPLETED", "refundTransferId": refundID})
	}}
	if _, e := s.StepReversal(ctx); e != nil {
		t.Fatal(e)
	}
	var state string
	_ = db.QueryRow(ctx, `SELECT estado FROM reversos_interbancarios`).Scan(&state)
	if state != "PROCESSING" {
		t.Fatal("partner COMPLETED trusted before local settlement receipt")
	}
	incoming := sample()
	incoming.ID = refundID
	incoming.OriginAccount = original.DestinationAccount
	incoming.DestinationAccount = original.OriginAccount
	incoming.Amount = original.Amount
	if r := s.create(ctx, incoming, "IN", ""); r.Status != 202 {
		t.Fatal(r)
	}
	if _, e := s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	_, _ = db.Exec(ctx, `UPDATE reversos_interbancarios SET proximo=clock_timestamp()`)
	if _, e := s.StepReversal(ctx); e != nil {
		t.Fatal(e)
	}
	_ = db.QueryRow(ctx, `SELECT estado FROM reversos_interbancarios`).Scan(&state)
	if state != "COMPLETED" || effects["refund_received"] != 1 || effects["credit"] != 0 {
		t.Fatalf("settlement duplicated customer money: %s %v", state, effects)
	}
}
