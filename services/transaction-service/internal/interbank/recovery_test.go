package interbank

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

type recoveryPartner struct {
	get  func(string) (int, []byte, error)
	post func(string, any) (int, []byte, error)
}

func (p recoveryPartner) Get(ctx context.Context, path string) (int, []byte, error) {
	if p.get == nil {
		return 503, nil, nil
	}
	return p.get(path)
}
func (p recoveryPartner) Post(ctx context.Context, path, key string, body any) (int, []byte, error) {
	if p.post == nil {
		return 503, nil, nil
	}
	return p.post(path, body)
}
func response(status int, body any) (int, []byte, error) {
	b, _ := json.Marshal(body)
	return status, b, nil
}

func TestCancellationTombstoneAndOrphanCreditDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}}
	tr := sample()
	c, _ := json.Marshal(Cancel{Origin: "BANK14", Reason: "TIMEOUT"})
	request := Request{Action: "cancel", ID: tr.ID, Peer: "BANK14", Body: c}
	if r := invoke(s, request); r.Status != 200 {
		t.Fatal(r)
	}
	tr.Created = tr.Created.Add(-time.Hour)
	tr.Expires = tr.Created.Add(180 * time.Second)
	r := invoke(s, Request{Action: "receive", Key: tr.ID, Peer: "BANK14", Body: payload(tr)})
	var result map[string]any
	_ = json.Unmarshal(r.Body, &result)
	if r.Status != 200 || result["status"] != "CANCELLED" {
		t.Fatalf("cancelled original arrived later: %d %s", r.Status, r.Body)
	}
	tr = sample()
	if r := invoke(s, Request{Action: "receive", Key: tr.ID, Peer: "BANK14", Body: payload(tr)}); r.Status != 202 {
		t.Fatal(r)
	}
	processed := time.Now().UTC()
	s.Account = func(context.Context, AccountCommand) (AccountResult, error) {
		return AccountResult{State: "CREDITED", Processed: processed}, nil
	}
	request.ID = tr.ID
	r = invoke(s, request)
	_ = json.Unmarshal(r.Body, &result)
	if r.Status != 200 || result["result"] != "ALREADY_ACCEPTED" {
		t.Fatalf("orphan credit cancelled: %d %s", r.Status, r.Body)
	}
	var updated time.Time
	_ = db.QueryRow(ctx, `SELECT actualizado FROM depositos_interbancarios WHERE id=$1`, tr.ID).Scan(&updated)
	invoke(s, request)
	var repeated time.Time
	_ = db.QueryRow(ctx, `SELECT actualizado FROM depositos_interbancarios WHERE id=$1`, tr.ID).Scan(&repeated)
	if !updated.Equal(repeated) {
		t.Fatal("duplicate cancellation reset callback retry window")
	}
}

func TestTimeoutProtocolDatabase(t *testing.T) {
	for _, test := range []struct {
		name, status, cancel string
		queryCode            int
		age                  time.Duration
		want                 string
		incident             bool
	}{
		{"query accepted", "ACCEPTED", "", 200, time.Second, "ACCEPTED", false},
		{"query rejected", "REJECTED", "", 200, time.Second, "REVERSED", false},
		{"cancel absent", "", "CANCELLED", 404, time.Second, "REVERSED", false},
		{"cancel already accepted", "PROCESSING", "ALREADY_ACCEPTED", 200, time.Second, "ACCEPTED", false},
		{"cancel already rejected", "PROCESSING", "ALREADY_REJECTED", 200, time.Second, "REVERSED", false},
		{"margin not elapsed", "", "", 503, 299 * time.Second, "TIMEOUT", false},
		{"margin elapsed", "", "", 503, 301 * time.Second, "REVERSED", true},
		{"auth failure never releases blindly", "", "", 401, 301 * time.Second, "TIMEOUT", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := testDB(t)
			ctx := context.Background()
			now := time.Now().UTC()
			tr := sample()
			tr.Origin = "BANK03"
			tr.Destination = "BANK14"
			s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Now: func() time.Time { return now }, Account: func(ctx context.Context, c AccountCommand) (AccountResult, error) {
				state := "RESERVED"
				if c.Action == "settle" {
					state = "DEBITED"
				}
				if c.Action == "release" {
					state = "RELEASED"
				}
				return AccountResult{State: state}, nil
			}}
			s.Partner = recoveryPartner{get: func(path string) (int, []byte, error) {
				return response(test.queryCode, map[string]any{"interbankTransferId": tr.ID, "status": test.status})
			}, post: func(path string, body any) (int, []byte, error) {
				if test.cancel != "" {
					return response(200, map[string]any{"interbankTransferId": tr.ID, "result": test.cancel})
				}
				return 503, nil, nil
			}}
			if r := s.create(ctx, tr, "OUT", uuid.NewString()); r.Status != 202 {
				t.Fatal(r)
			}
			tr.Expires = now.Add(-test.age)
			tr.Created = tr.Expires.Add(-180 * time.Second)
			_, e := db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='TIMEOUT',solicitud=$2,proximo=clock_timestamp() WHERE id=$1`, tr.ID, payload(tr))
			if e != nil {
				t.Fatal(e)
			}
			for i := 0; i < 2; i++ {
				_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp() WHERE id=$1`, tr.ID)
				if _, e = s.Step(ctx); e != nil {
					t.Fatal(e)
				}
			}
			row, e := scan(db.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, tr.ID))
			if e != nil || row.State != test.want {
				t.Fatalf("got %+v %v want %s", row, e, test.want)
			}
			var incidents int
			_ = db.QueryRow(ctx, `SELECT count(*) FROM incidentes_interbancarios`).Scan(&incidents)
			if (incidents > 0) != test.incident {
				t.Fatalf("incidents=%d", incidents)
			}
		})
	}
}

func TestLateCallbackCreatesOneReversalDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}}
	tr := sample()
	tr.Origin = "BANK03"
	tr.Destination = "BANK14"
	if r := s.create(ctx, tr, "OUT", uuid.NewString()); r.Status != 202 {
		t.Fatal(r)
	}
	_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='REVERSED' WHERE id=$1`, tr.ID)
	b, _ := json.Marshal(Result{ID: tr.ID, Destination: "BANK14", Status: "ACCEPTED", Processed: tr.Expires.Add(time.Minute)})
	for i := 0; i < 3; i++ {
		r := invoke(s, Request{Action: "result", ID: tr.ID, Peer: "BANK14", Body: b})
		if r.Status != 200 {
			t.Fatal(r)
		}
	}
	var reversals, incidents int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM reversos_interbancarios`).Scan(&reversals)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM incidentes_interbancarios`).Scan(&incidents)
	if reversals != 1 || incidents != 1 {
		t.Fatalf("reversals=%d incidents=%d", reversals, incidents)
	}
}
