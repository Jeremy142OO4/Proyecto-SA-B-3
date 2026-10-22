package interbank

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestReconciliationEndpointUTCAndPaginationDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	s := &Service{KYC: func(context.Context, string, string) (bool, error) { return true, nil }, DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}}
	day := "2026-10-18"
	for i := 0; i < 2; i++ {
		tr := sample()
		if r := s.create(ctx, tr, "IN", ""); r.Status != 202 {
			t.Fatal(r)
		}
		_, e := db.Exec(ctx, `UPDATE depositos_interbancarios SET recibido=$2 WHERE id=$1`, tr.ID, day+"T23:59:00Z")
		if e != nil {
			t.Fatal(e)
		}
	}
	for page := 1; page <= 2; page++ {
		r := invoke(s, Request{Action: "reconciliation", Peer: "BANK14", Date: day, Page: page, PageSize: 1})
		var result reconciliationPage
		if r.Status != 200 || json.Unmarshal(r.Body, &result) != nil || result.Total != 2 || len(result.Items) != 1 || result.Page != page {
			t.Fatalf("page %d: %d %s", page, r.Status, r.Body)
		}
	}
	if r := invoke(s, Request{Action: "reconciliation", Peer: "BANK99", Date: day}); r.Status != 401 {
		t.Fatal(r)
	}
	if r := invoke(s, Request{Action: "reconciliation", Peer: "BANK14", Date: "bad"}); r.Status != 400 {
		t.Fatal(r)
	}
	if r := invoke(s, Request{Action: "reconciliation", Peer: "BANK14", Date: day, PageSize: 501}); r.Status != 400 {
		t.Fatal(r)
	}
}

func TestDailyReconciliationFindsDiscrepanciesDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(24 * time.Hour).Add(12 * time.Hour)
	yesterday := now.Truncate(24 * time.Hour).Add(-24 * time.Hour)
	s := &Service{KYC: func(context.Context, string, string) (bool, error) { return true, nil }, DB: db, Config: Config{Bank: "BANK03", Partner: "BANK14"}, Now: func() time.Time { return now }}
	makeOut := func(state string) Transfer {
		t.Helper()
		tr := sample()
		tr.Origin = "BANK03"
		tr.Destination = "BANK14"
		if r := s.create(ctx, tr, "OUT", uuid.NewString()); r.Status != 202 {
			t.Fatal(r)
		}
		tr.Created = yesterday.Add(12 * time.Hour)
		tr.Expires = tr.Created.Add(180 * time.Second)
		_, e := db.Exec(ctx, `UPDATE depositos_interbancarios SET estado=$2,solicitud=$3 WHERE id=$1`, tr.ID, state, payload(tr))
		if e != nil {
			t.Fatal(e)
		}
		return tr
	}
	accepted := makeOut("TIMEOUT")
	mismatch := makeOut("TIMEOUT")
	_ = makeOut("ACCEPTED") // absent remotely
	date := yesterday.Format("2006-01-02")
	s.Partner = recoveryPartner{get: func(path string) (int, []byte, error) {
		return response(200, reconciliationPage{Date: date, Page: 1, Size: 100, Total: 3, Items: []reconciliationItem{
			{ID: accepted.ID, Amount: accepted.Amount, Currency: "GTQ", Status: "ACCEPTED", Updated: yesterday.Add(time.Hour)},
			{ID: mismatch.ID, Amount: "999.00", Currency: "GTQ", Status: "ACCEPTED", Updated: yesterday.Add(time.Hour)},
			{ID: uuid.NewString(), Amount: "10.00", Currency: "GTQ", Status: "ACCEPTED", Updated: yesterday.Add(time.Hour)},
		}})
	}}
	if _, e := s.StepReconciliation(ctx); e != nil {
		t.Fatal(e)
	}
	row, e := scan(db.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, accepted.ID))
	if e != nil || row.Result == nil || row.Result.Status != "ACCEPTED" {
		t.Fatalf("accepted result not recovered %+v %v", row, e)
	}
	var incidents, count int
	var state string
	_ = db.QueryRow(ctx, `SELECT count(*) FROM incidentes_interbancarios WHERE tipo LIKE 'RECONCILIATION_%'`).Scan(&incidents)
	_ = db.QueryRow(ctx, `SELECT estado,discrepancias FROM conciliaciones_interbancarias WHERE fecha=$1`, date).Scan(&state, &count)
	if incidents != 3 || state != "COMPLETED" || count != 3 {
		t.Fatalf("incidents=%d state=%s discrepancies=%d", incidents, state, count)
	}
	if worked, e := s.StepReconciliation(ctx); e != nil || worked {
		t.Fatalf("completed daily job ran twice: %v %v", worked, e)
	}
}
