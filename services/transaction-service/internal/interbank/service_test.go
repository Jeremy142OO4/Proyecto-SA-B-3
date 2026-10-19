package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type fakePartner struct {
	calls      int
	callbackID string
	fail       bool
}

func (p *fakePartner) Post(ctx context.Context, path, key string, body any) (int, []byte, error) {
	p.calls++
	if p.fail {
		return 0, nil, errors.New("network timeout")
	}
	if key != "" {
		t := body.(Transfer)
		b, _ := json.Marshal(map[string]string{"interbankTransferId": t.ID, "status": "PROCESSING"})
		return 202, b, nil
	}
	r := body.(*Result)
	p.callbackID = r.ID
	b, _ := json.Marshal(map[string]any{"interbankTransferId": r.ID, "received": true})
	return 200, b, nil
}
func sample() Transfer {
	now := time.Now().UTC().Truncate(time.Second)
	return Transfer{ID: uuid.NewString(), Origin: "BANK14", Destination: "BANK03", OriginAccount: "OTHER-001", DestinationAccount: "LOCAL-001", Amount: "10.00", Currency: "GTQ", Created: now, Expires: now.Add(180 * time.Second)}
}
func invoke(s *Service, r Request) bankrpc.Response {
	b, _ := json.Marshal(r)
	return s.Handle(context.Background(), b)
}
func payload(t Transfer) json.RawMessage { b, _ := json.Marshal(t); return b }
func TestReceiveDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	partner := &fakePartner{}
	credits := map[string]AccountResult{}
	creditCount := 0
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: partner, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
		if r, ok := credits[c.ID]; ok {
			return r, nil
		}
		creditCount++
		r := AccountResult{State: "CREDITED", Processed: time.Now().UTC()}
		credits[c.ID] = r
		return r, nil
	}}
	tr := sample()
	req := Request{Action: "receive", Peer: "BANK14", Key: tr.ID, Body: payload(tr)}
	var wg sync.WaitGroup
	accepted := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); accepted <- invoke(s, req).Status }()
	}
	wg.Wait()
	close(accepted)
	count := 0
	for code := range accepted {
		if code == 202 {
			count++
		} else if code != 200 {
			t.Fatal(code)
		}
	}
	if count != 1 {
		t.Fatalf("new receipts=%d", count)
	}
	changed := tr
	changed.Amount = "11.00"
	conflict := req
	conflict.Body = payload(changed)
	if r := invoke(s, conflict); r.Status != 409 {
		t.Fatal(r)
	}
	if _, e := s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	row, e := scan(db.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, tr.ID))
	if e != nil || row.State != "ACCEPTED" || creditCount != 1 {
		t.Fatalf("%+v %v credit=%d", row, e, creditCount)
	}
	// Simulate crash AFTER account commit but BEFORE coordinator commit.
	_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='PROCESSING',resultado=NULL,proximo=clock_timestamp() WHERE id=$1`, tr.ID)
	if _, e = s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	if creditCount != 1 {
		t.Fatal("duplicate credit after restart")
	}
	partner.fail = true
	_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp() WHERE id=$1`, tr.ID)
	if _, e = s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	var delivered bool
	_ = db.QueryRow(ctx, `SELECT callback_entregado FROM depositos_interbancarios WHERE id=$1`, tr.ID).Scan(&delivered)
	if delivered {
		t.Fatal("failed callback marked delivered")
	}
	partner.fail = false
	_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp() WHERE id=$1`, tr.ID)
	if _, e = s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	if partner.callbackID != tr.ID {
		t.Fatal("callback not delivered")
	}
	late := sample()
	late.Created = late.Created.Add(-time.Hour)
	late.Expires = late.Created.Add(180 * time.Second)
	if r := invoke(s, Request{Action: "receive", Peer: "BANK14", Key: late.ID, Body: payload(late)}); r.Status != 422 {
		t.Fatal(r)
	}
	if r := invoke(s, Request{Action: "get", ID: tr.ID, Peer: "BANK99"}); r.Status != 401 {
		t.Fatal(r)
	}
}
func TestOutgoingDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	partner := &fakePartner{}
	states := map[string]string{}
	effects := map[string]int{}
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: partner, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
		state := states[c.ID]
		switch c.Action {
		case "reserve":
			if state == "" {
				state = "RESERVED"
				effects[c.ID+"reserve"]++
			}
		case "settle":
			if state == "RESERVED" {
				state = "DEBITED"
				effects[c.ID+"debit"]++
			}
		case "release":
			if state == "RESERVED" {
				state = "RELEASED"
				effects[c.ID+"release"]++
			}
		}
		states[c.ID] = state
		return AccountResult{State: state}, nil
	}}
	owner := uuid.NewString()
	for _, outcome := range []string{"ACCEPTED", "REJECTED"} {
		t.Run(outcome, func(t *testing.T) {
			tr := sample()
			tr.Origin = "BANK03"
			tr.Destination = "BANK14"
			r := Request{Action: "send", Customer: owner, Key: tr.ID, Body: payload(tr)}
			if reply := invoke(s, r); reply.Status != 202 {
				t.Fatal(reply)
			}
			if reply := invoke(s, r); reply.Status != 200 {
				t.Fatal(reply)
			}
			if _, e := s.Step(ctx); e != nil {
				t.Fatal(e)
			} // Reserve.
			_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp() WHERE id=$1`, tr.ID)
			if _, e := s.Step(ctx); e != nil {
				t.Fatal(e)
			} // Send actual HTTP via Partner interface.
			result := Result{ID: tr.ID, Destination: "BANK14", Status: outcome, Processed: time.Now().UTC()}
			if outcome == "REJECTED" {
				result.Reason = "ACCOUNT_NOT_FOUND"
			}
			b, _ := json.Marshal(result)
			callback := Request{Action: "result", ID: tr.ID, Peer: "BANK14", Body: b}
			for i := 0; i < 2; i++ {
				reply := invoke(s, callback)
				if reply.Status != 200 {
					t.Fatalf("callback %d %s", reply.Status, reply.Body)
				}
			}
			if _, e := s.Step(ctx); e != nil {
				t.Fatal(e)
			}
			wanted := "DEBITED"
			suffix := "debit"
			if outcome == "REJECTED" {
				wanted = "RELEASED"
				suffix = "release"
			}
			if states[tr.ID] != wanted || effects[tr.ID+suffix] != 1 || effects[tr.ID+"reserve"] != 1 {
				t.Fatalf("state=%s effects=%v", states[tr.ID], effects)
			}
			if r := invoke(s, Request{Action: "client_get", ID: tr.ID, Customer: uuid.NewString()}); r.Status != 401 {
				t.Fatal(r)
			}
		})
	}
	// Unknown result cannot create an outgoing operation.
	unknown := Result{ID: uuid.NewString(), Destination: "BANK14", Status: "ACCEPTED", Processed: time.Now().UTC()}
	b, _ := json.Marshal(unknown)
	if r := invoke(s, Request{Action: "result", ID: unknown.ID, Peer: "BANK14", Body: b}); r.Status != 404 {
		t.Fatal(r)
	}
}
func TestTimeoutRetainsReservationDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	released := false
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: &fakePartner{fail: true}, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
		if c.Action == "release" {
			released = true
		}
		return AccountResult{State: "RESERVED"}, nil
	}}
	tr := sample()
	tr.Origin = "BANK03"
	tr.Destination = "BANK14"
	if r := invoke(s, Request{Action: "send", Customer: uuid.NewString(), Key: tr.ID, Body: payload(tr)}); r.Status != 202 {
		t.Fatal(r)
	}
	if _, e := s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	tr.Created = tr.Created.Add(-time.Hour)
	tr.Expires = tr.Created.Add(180 * time.Second)
	_, e := db.Exec(ctx, `UPDATE depositos_interbancarios SET solicitud=$2,proximo=clock_timestamp() WHERE id=$1`, tr.ID, payload(tr))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	row, e := scan(db.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, tr.ID))
	if e != nil || row.State != "TIMEOUT" || released {
		t.Fatalf("unsafe timeout %+v %v released=%v", row, e, released)
	}
}

func TestCreditReceiptRecoveredAfterExpiryDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	processed := time.Now().UTC()
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: &fakePartner{}, Account: func(context.Context, AccountCommand) (AccountResult, error) {
		return AccountResult{State: "CREDITED", Processed: processed}, nil
	}}
	tr := sample()
	tr.Expires = time.Now().UTC().Add(150 * time.Millisecond)
	tr.Created = tr.Expires.Add(-180 * time.Second)
	if r := s.create(ctx, tr, "IN", ""); r.Status != 202 {
		t.Fatal(r)
	}
	if _, e := s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	time.Sleep(200 * time.Millisecond)
	_, e := db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='PROCESSING',resultado=NULL,proximo=clock_timestamp() WHERE id=$1`, tr.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Step(ctx); e != nil {
		t.Fatal(e)
	}
	row, e := scan(db.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, tr.ID))
	if e != nil || row.State != "ACCEPTED" || row.Result == nil || !row.Result.Processed.Equal(processed) {
		t.Fatalf("lost committed credit after expiry: %+v %v", row, e)
	}
}
func TestHTTPPartnerNoRedirectOrKeyLeak(t *testing.T) {
	leaked := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "local-only-partner-key" {
			t.Error("missing key")
		}
		http.Redirect(w, r, target.URL, 307)
	}))
	defer server.Close()
	p := NewPartner(Config{PartnerURL: server.URL, PartnerKey: "local-only-partner-key"})
	code, _, e := p.Post(context.Background(), "/transfers", "test", map[string]string{})
	if e != nil || code != 307 || leaked {
		t.Fatalf("redirect leaked credentials code=%d err=%v leaked=%v", code, e, leaked)
	}
}
func TestContractValidation(t *testing.T) {
	tr := sample()
	if !tr.Valid() {
		t.Fatal("valid request rejected")
	}
	for _, v := range []string{"1.0", "1e2", "-1.00", "100000.00", "NaN"} {
		if _, e := Cents(v); e == nil {
			t.Fatal(v)
		}
	}
	tr.Expires = tr.Expires.Add(time.Second)
	if tr.Valid() {
		t.Fatal("wrong validity window")
	}
	if strict([]byte(`{"amount":"1.00","extra":true}`), &Transfer{}) == nil {
		t.Fatal("unknown field")
	}
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "http://127.0.0.1/interbank/v1")
	t.Setenv("INTERBANK_PARTNER_API_KEY", "local-test-only-123456")
	if _, e := Load("amqp://local"); e == nil {
		t.Fatal("real config accepts plaintext HTTP")
	}
}
