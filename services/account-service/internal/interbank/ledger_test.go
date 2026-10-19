package interbank

import (
	"context"
	"errors"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/Proyecto-SA-B-3/account-service/repositories"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func TestLedgerDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ledger := Ledger{db}
	owner := uuid.NewString()
	accountID := uuid.New()
	_, e := db.Exec(ctx, `INSERT INTO cuentas(id_cuenta,id_cliente,numero_cuenta,tipo_cuenta,saldo_centavos,saldo_minimo_centavos,comision_transaccion_centavos) VALUES($1,$2,'BANK03-TEST','MONETARIA',10000,1000,25)`, accountID, owner)
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
	assertBalance := func(balance, held int64) {
		t.Helper()
		var b, h int64
		if e := db.QueryRow(ctx, `SELECT saldo_centavos,saldo_reservado_centavos FROM cuentas WHERE id_cuenta=$1`, accountID).Scan(&b, &h); e != nil {
			t.Fatal(e)
		}
		if b != balance || h != held {
			t.Fatalf("balance=%d held=%d want %d/%d", b, h, balance, held)
		}
	}
	command := Command{ID: uuid.NewString(), Action: "reserve", Account: "BANK03-TEST", Customer: owner, Cents: 5000, Expires: time.Now().UTC().Truncate(time.Microsecond).Add(time.Minute)}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := ledger.Apply(ctx, command)
			if e != nil || r.State != "RESERVED" {
				t.Errorf("reserve: %+v %v", r, e)
			}
		}()
	}
	wg.Wait()
	assertBalance(10000, 5025)
	other := command
	other.ID = uuid.NewString()
	if r := apply(other); r.Reason != "INSUFFICIENT_FUNDS" {
		t.Fatalf("overspend allowed %+v", r)
	}
	changed := command
	changed.Cents++
	if r := apply(changed); r.Reason != "IDEMPOTENCY_CONFLICT" {
		t.Fatal(r)
	}
	// Existing internal/payment debit paths cannot consume the held amount.
	repo := repositories.NuevoRepositorioCuentasPostgres(db)
	_, e = repo.ProcesarMovimiento(ctx, repositories.SolicitudMovimientoCuenta{IDMensaje: uuid.New(), IDCorrelacion: uuid.New(), IDCuenta: accountID, IDOperacion: uuid.New(), TipoMovimiento: models.TipoMovimientoDebito, MontoCentavos: 5000, NombreConsumidor: "test", TipoEventoExitoso: "test.debit"})
	if !errors.Is(e, repositories.ErrSaldoMinimo) {
		t.Fatalf("ordinary debit consumed held money: %v", e)
	}
	command.Action = "settle"
	if r := apply(command); r.State != "DEBITED" {
		t.Fatal(r)
	}
	apply(command)
	assertBalance(4975, 0)
	command.Action = "release"
	if r := apply(command); r.State != "DEBITED" {
		t.Fatal(r)
	}
	assertBalance(4975, 0)
	release := command
	release.ID = uuid.NewString()
	release.Action = "reserve"
	release.Cents = 1000
	if r := apply(release); r.State != "RESERVED" {
		t.Fatal(r)
	}
	release.Action = "release"
	apply(release)
	apply(release)
	assertBalance(4975, 0)
	credit := Command{ID: uuid.NewString(), Action: "credit", Account: "BANK03-TEST", Cents: 2000, Expires: command.Expires}
	first := apply(credit)
	second := apply(credit)
	if first.State != "CREDITED" || first.Processed.IsZero() || !first.Processed.Equal(second.Processed) {
		t.Fatalf("credit receipt not durable %+v %+v", first, second)
	}
	assertBalance(6975, 0)
	expired := credit
	expired.ID = uuid.NewString()
	expired.Expires = time.Now().Add(-time.Second)
	if r := apply(expired); r.Reason != "EXPIRED" {
		t.Fatal(r)
	}
	assertBalance(6975, 0)
	bad := credit
	bad.ID = uuid.NewString()
	bad.Account = "MISSING"
	if r := apply(bad); r.Reason != "ACCOUNT_NOT_FOUND" {
		t.Fatal(r)
	}
	wrongOwner := release
	wrongOwner.ID = uuid.NewString()
	wrongOwner.Action = "reserve"
	wrongOwner.Customer = uuid.NewString()
	if r := apply(wrongOwner); r.Reason != "OWNER_MISMATCH" {
		t.Fatal(r)
	}
	var count int
	if e = db.QueryRow(ctx, `SELECT count(*) FROM movimientos_cuenta`).Scan(&count); e != nil || count != 2 {
		t.Fatalf("movements=%d %v", count, e)
	}
}
func TestCreditExpiresWhileWaitingForAccountLockDatabase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ledger := Ledger{db}
	_, e := db.Exec(ctx, `INSERT INTO cuentas(id_cliente,numero_cuenta,tipo_cuenta) VALUES($1,'LOCKED','AHORRO')`, uuid.New())
	if e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT id_cuenta FROM cuentas WHERE numero_cuenta='LOCKED' FOR UPDATE`); e != nil {
		t.Fatal(e)
	}
	done := make(chan Result, 1)
	errc := make(chan error, 1)
	cmd := Command{ID: uuid.NewString(), Action: "credit", Account: "LOCKED", Cents: 100, Expires: time.Now().UTC().Add(150 * time.Millisecond)}
	go func() { r, e := ledger.Apply(ctx, cmd); done <- r; errc <- e }()
	time.Sleep(250 * time.Millisecond)
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	r := <-done
	if e = <-errc; e != nil || r.Reason != "EXPIRED" {
		t.Fatalf("late credit: %+v %v", r, e)
	}
	var balance int64
	_ = db.QueryRow(ctx, `SELECT saldo_centavos FROM cuentas WHERE numero_cuenta='LOCKED'`).Scan(&balance)
	if balance != 0 {
		t.Fatal("credited after expiry")
	}
}
