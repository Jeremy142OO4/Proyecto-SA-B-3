package repositories

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type debitoCompensacionTx struct {
	pgx.Tx
	t                  *testing.T
	account, operation uuid.UUID
	amount             int64
	err                error
}
type debitoCompensacionRow struct {
	amount int64
	err    error
}

func (r debitoCompensacionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*int64) = r.amount
	return nil
}
func (tx debitoCompensacionTx) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	if !strings.Contains(query, "id_cuenta=$2") || !strings.Contains(query, "tipo_movimiento='DEBITO'") || len(args) != 2 || args[0] != tx.operation || args[1] != tx.account {
		tx.t.Fatalf("compensation must read the debit of the same account and operation: %s %v", query, args)
	}
	return debitoCompensacionRow{tx.amount, tx.err}
}
func TestCompensacionDevuelveDebitoOriginalConComision(t *testing.T) {
	account, operation := uuid.New(), uuid.New()
	for _, requested := range []int64{1, 10000, 999999} {
		c := models.Cuenta{IDCuenta: account, SaldoCentavos: 89000, ComisionTransaccionCentavos: 7000}
		s := SolicitudMovimientoCuenta{IDCuenta: account, IDOperacion: operation, TipoMovimiento: models.TipoMovimientoCompensacion, MontoCentavos: requested}
		tx := debitoCompensacionTx{t: t, account: account, operation: operation, amount: 11000}
		balance, err := calcularSaldoNuevo(context.Background(), tx, c, s)
		if err != nil || balance != 100000 {
			t.Fatalf("requested=%d balance=%d err=%v", requested, balance, err)
		}
	}
}
func TestCompensacionRechazaDebitoAusenteYDesbordamiento(t *testing.T) {
	account, operation := uuid.New(), uuid.New()
	s := SolicitudMovimientoCuenta{IDCuenta: account, IDOperacion: operation, TipoMovimiento: models.TipoMovimientoCompensacion, MontoCentavos: 100}
	tx := debitoCompensacionTx{t: t, account: account, operation: operation, err: pgx.ErrNoRows}
	if _, err := calcularSaldoNuevo(context.Background(), tx, models.Cuenta{}, s); !errors.Is(err, ErrMovimientoNoEncontrado) {
		t.Fatal(err)
	}
	tx.err = nil
	tx.amount = 100
	if _, err := calcularSaldoNuevo(context.Background(), tx, models.Cuenta{SaldoCentavos: math.MaxInt64 - 50}, s); err == nil {
		t.Fatal("overflow must never credit a wrapped balance")
	}
}
