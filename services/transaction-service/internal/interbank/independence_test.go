package interbank

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"testing"
	"time"
)

func TestLoadWithoutPartnerURL(t *testing.T) {
	t.Setenv("INTERBANK_BANK_CODE", "BANK03")
	t.Setenv("INTERBANK_PARTNER_BANK_CODE", "BANK14")
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "")
	t.Setenv("INTERBANK_PARTNER_API_KEY", "")
	c, err := Load("amqp://local-test")
	if err != nil {
		t.Fatal(err)
	}
	p := NewPartner(c)
	if p.Ready() {
		t.Fatal("outbound unexpectedly configured")
	}
	if _, _, err = p.Get(context.Background(), "/health"); !errors.Is(err, ErrPartnerNotConfigured) {
		t.Fatal(err)
	}
	t.Setenv("INTERBANK_PARTNER_API_KEY", "local-only-partner-key")
	if _, err = Load(""); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"http://bank14.test/interbank/v1", "https://bank14.test/wrong", "https://user:pass@bank14.test/interbank/v1", "https://bank14.test/interbank/v1?key=secret"} {
		t.Setenv("INTERBANK_PARTNER_BASE_URL", raw)
		if _, err = Load(""); err == nil {
			t.Fatalf("unsafe URL accepted: %s", raw)
		}
	}
	t.Setenv("INTERBANK_PARTNER_BASE_URL", "https://bank14.test/interbank/v1")
	c, err = Load("")
	if err != nil || !NewPartner(c).Ready() {
		t.Fatal("valid outbound config rejected", err)
	}
}

func TestSendWithoutPartnerDoesNotTouchDatabase(t *testing.T) {
	s := &Service{Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: NewPartner(Config{})}
	r := invoke(s, Request{Action: "send", Customer: uuid.NewString(), Body: payload(sample())})
	if r.Status != http.StatusServiceUnavailable {
		t.Fatal(r)
	}
	if worked, err := s.StepReconciliation(context.Background()); worked || err != nil {
		t.Fatal(worked, err)
	}
	if worked, err := s.StepReversal(context.Background()); worked || err != nil {
		t.Fatal(worked, err)
	}
}

func TestReceivingIndependentOfPartnerDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	count := 0
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: NewPartner(Config{}), Account: func(_ context.Context, c AccountCommand) (AccountResult, error) {
		count++
		if c.Action != "credit" {
			t.Fatalf("unexpected money operation: %s", c.Action)
		}
		return AccountResult{State: "CREDITED", Processed: time.Now().UTC()}, nil
	}}
	tr := sample()
	if r := invoke(s, Request{Action: "receive", Peer: "BANK14", Key: tr.ID, Body: payload(tr)}); r.Status != 202 {
		t.Fatal(r)
	}
	if _, err := s.Step(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp() WHERE id=$1`, tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Step(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	var delivered bool
	var result []byte
	if err = db.QueryRow(ctx, `SELECT estado,callback_entregado,resultado FROM depositos_interbancarios WHERE id=$1`, tr.ID).Scan(&state, &delivered, &result); err != nil {
		t.Fatal(err)
	}
	if state != "ACCEPTED" || delivered || len(result) == 0 || count != 1 {
		t.Fatalf("state=%s delivered=%v credits=%d result=%s", state, delivered, count, result)
	}
	// Once configured, the durable callback is retried without crediting again.
	partner := &fakePartner{}
	s.Partner = partner
	_, _ = db.Exec(ctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp() WHERE id=$1`, tr.ID)
	if _, err = s.Step(ctx); err != nil {
		t.Fatal(err)
	}
	if partner.callbackID != tr.ID || count != 1 {
		t.Fatal("durable callback lost or double credit")
	}
}

func TestMissingLocalConfigurationDoesNotReleaseExistingHoldDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	calls := 0
	s := &Service{DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Partner: NewPartner(Config{}), Account: func(_ context.Context, c AccountCommand) (AccountResult, error) {
		calls++
		if c.Action != "settle" {
			t.Fatalf("missing local config triggered %s", c.Action)
		}
		return AccountResult{State: "DEBITED"}, nil
	}}
	tr := sample()
	tr.Origin, tr.Destination = "BANK03", "BANK14"
	if r := s.create(ctx, tr, "OUT", uuid.NewString()); r.Status != 202 {
		t.Fatal(r)
	}
	tr.Created = tr.Created.Add(-time.Hour)
	tr.Expires = tr.Created.Add(180 * time.Second)
	if _, err := db.Exec(ctx, `UPDATE depositos_interbancarios SET estado='RESERVED',solicitud=$2,proximo=clock_timestamp() WHERE id=$1`, tr.ID, payload(tr)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Step(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := db.QueryRow(ctx, `SELECT estado FROM depositos_interbancarios WHERE id=$1`, tr.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || state != "RESERVED" {
		t.Fatalf("hold changed without outbound config: %s calls=%d", state, calls)
	}
	// An authenticated final callback still settles a preexisting reservation.
	result := Result{ID: tr.ID, Destination: "BANK14", Status: "ACCEPTED", Processed: tr.Created.Add(time.Second)}
	if r := s.callback(ctx, result); r.Status != 200 {
		t.Fatal(r)
	}
	if _, err := s.Step(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("incoming result was blocked by missing outbound config")
	}
}
