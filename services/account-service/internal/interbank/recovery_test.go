package interbank

import (
	"context"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func TestCancelVersusCreditDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ledger := Ledger{db}
	_, e := db.Exec(ctx, `INSERT INTO cuentas(id_cliente,numero_cuenta,tipo_cuenta) VALUES($1,'CANCEL-TEST','AHORRO')`, uuid.New())
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 10; i++ {
		c := Command{ID: uuid.NewString(), Action: "credit", Account: "CANCEL-TEST", Cents: 100, Expires: time.Now().UTC().Truncate(time.Microsecond).Add(time.Minute)}
		cancel := c
		cancel.Action = "cancel_credit"
		var credited, cancelled Result
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			r, e := ledger.Apply(ctx, c)
			if e != nil {
				t.Error(e)
			}
			credited = r
		}()
		go func() {
			defer wg.Done()
			r, e := ledger.Apply(ctx, cancel)
			if e != nil {
				t.Error(e)
			}
			cancelled = r
		}()
		wg.Wait()
		if credited.State != cancelled.State {
			t.Fatalf("cancel/credit disagree: %+v %+v", credited, cancelled)
		}
		if credited.State != "CREDITED" && credited.Reason != "CANCELLED" {
			t.Fatal(credited)
		}
	}
	var balance, count int64
	_ = db.QueryRow(ctx, `SELECT saldo_centavos FROM cuentas WHERE numero_cuenta='CANCEL-TEST'`).Scan(&balance)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM operaciones_interbancarias WHERE estado='CREDITED'`).Scan(&count)
	if balance != count*100 {
		t.Fatalf("balance=%d committed credits=%d", balance, count)
	}
}

func TestRefundConservesMoneyDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ledger := Ledger{db}
	owner := uuid.NewString()
	_, e := db.Exec(ctx, `INSERT INTO cuentas(id_cliente,numero_cuenta,tipo_cuenta,saldo_centavos,saldo_minimo_centavos,comision_transaccion_centavos) VALUES($1,'RECEIVER','AHORRO',0,2000,25),($1,'SENDER','MONETARIA',10000,1000,25)`, owner)
	if e != nil {
		t.Fatal(e)
	}
	apply := func(c Command) Result {
		t.Helper()
		r, e := ledger.Apply(ctx, c)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	expires := time.Now().UTC().Truncate(time.Microsecond).Add(time.Minute)
	original := Command{ID: uuid.NewString(), Action: "credit", Account: "RECEIVER", Cents: 1000, Expires: expires}
	if r := apply(original); r.State != "CREDITED" {
		t.Fatal(r)
	}
	refund := Command{ID: uuid.NewString(), Original: original.ID, Action: "refund_reserve", Account: "RECEIVER", Cents: 1000, Expires: expires}
	if r := apply(refund); r.State != "RESERVED" {
		t.Fatal(r)
	}
	refund.Action = "settle"
	if r := apply(refund); r.State != "DEBITED" {
		t.Fatal(r)
	}
	apply(refund)
	duplicate := refund
	duplicate.ID = uuid.NewString()
	duplicate.Action = "refund_reserve"
	if r := apply(duplicate); r.State != "REJECTED" || r.Reason != "DUPLICATE_REFUND" {
		t.Fatal(r)
	}
	var receiver, held int64
	_ = db.QueryRow(ctx, `SELECT saldo_centavos,saldo_reservado_centavos FROM cuentas WHERE numero_cuenta='RECEIVER'`).Scan(&receiver, &held)
	if receiver != 0 || held != 0 {
		t.Fatalf("refund charged a fee, double-debited, or ignored original: %d/%d", receiver, held)
	}
	// Sender had already released its original hold. The bank-to-bank return
	// settles that incident; it is not a second deposit owed to the customer.
	outgoing := Command{ID: uuid.NewString(), Action: "reserve", Account: "SENDER", Customer: owner, Cents: 1000, Expires: expires}
	apply(outgoing)
	outgoing.Action = "release"
	apply(outgoing)
	incoming := Command{ID: uuid.NewString(), Original: outgoing.ID, Action: "refund_received", Account: "SENDER", Customer: owner, Cents: 1000, Expires: expires}
	if r := apply(incoming); r.State != "CREDITED" {
		t.Fatal(r)
	}
	apply(incoming)
	duplicate = incoming
	duplicate.ID = uuid.NewString()
	if r := apply(duplicate); r.Reason != "DUPLICATE_REFUND" {
		t.Fatal(r)
	}
	var sender int64
	_ = db.QueryRow(ctx, `SELECT saldo_centavos,saldo_reservado_centavos FROM cuentas WHERE numero_cuenta='SENDER'`).Scan(&sender, &held)
	if sender != 10000 || held != 0 {
		t.Fatalf("money duplicated after refund: %d/%d", sender, held)
	}
	var movements int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM movimientos_cuenta`).Scan(&movements)
	if movements != 2 {
		t.Fatalf("unexpected customer movements=%d", movements)
	}
}
