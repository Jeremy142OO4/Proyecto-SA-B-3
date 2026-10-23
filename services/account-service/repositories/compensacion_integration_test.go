package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Proyecto-SA-B-3/account-service/events"
	"github.com/Proyecto-SA-B-3/account-service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCompensacionCompletaDatabase(t *testing.T) {
	ctx := context.Background()
	pool := baseMonitoreo(t)
	repo := NuevoRepositorioCuentasPostgres(pool)
	account := cuentaMonitoreo(t, repo)
	if _, err := pool.Exec(ctx, `UPDATE cuentas SET comision_transaccion_centavos=1000 WHERE id_cuenta=$1`, account.IDCuenta); err != nil {
		t.Fatal(err)
	}
	debit := solicitudMonitoreo(account.IDCuenta, models.TipoMovimientoDebito)
	debit.MontoCentavos = 10000
	if result, err := repo.ProcesarMovimiento(ctx, debit); err != nil || result.Movimiento.MontoCentavos != 11000 {
		t.Fatalf("debit=%+v err=%v", result, err)
	}
	// A later rule change or account block must not alter the historical refund.
	if _, err := pool.Exec(ctx, `UPDATE cuentas SET comision_transaccion_centavos=7000,estado='BLOQUEADA' WHERE id_cuenta=$1`, account.IDCuenta); err != nil {
		t.Fatal(err)
	}
	comp := solicitudMonitoreo(account.IDCuenta, models.TipoMovimientoCompensacion)
	comp.IDOperacion = debit.IDOperacion
	comp.MontoCentavos = 10000
	result, err := repo.ProcesarMovimiento(ctx, comp)
	if err != nil || result.Duplicado || result.Movimiento.MontoCentavos != 11000 || result.Movimiento.SaldoNuevoCentavos != 100000 {
		t.Fatalf("refund=%+v err=%v", result, err)
	}
	for i := 0; i < 3; i++ {
		if i > 0 {
			comp.IDMensaje = uuid.New()
			comp.MontoCentavos = 999999
		}
		replay, e := repo.ProcesarMovimiento(ctx, comp)
		if e != nil || !replay.Duplicado || replay.Movimiento.IDMovimiento != result.Movimiento.IDMovimiento {
			t.Fatalf("replay=%+v err=%v", replay, e)
		}
	}
	stored, e := repo.BuscarPorID(ctx, account.IDCuenta)
	if e != nil || stored.SaldoCentavos != 100000 {
		t.Fatalf("balance=%+v err=%v", stored, e)
	}
	var count int
	if e = pool.QueryRow(ctx, `SELECT count(*) FROM movimientos_cuenta WHERE id_operacion=$1 AND tipo_movimiento='COMPENSACION'`, debit.IDOperacion).Scan(&count); e != nil || count != 1 {
		t.Fatalf("refund count=%d err=%v", count, e)
	}
	var raw []byte
	if e = pool.QueryRow(ctx, `SELECT contenido FROM mensajes_salida WHERE tipo_evento=$1 AND contenido->>'idOperacion'=$2`, events.EventoCuentaCompensada, debit.IDOperacion.String()).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var movement models.MovimientoCuenta
	if e = json.Unmarshal(raw, &movement); e != nil || movement.MontoCentavos != 11000 {
		t.Fatalf("outbox=%s err=%v", raw, e)
	}
	other := cuentaMonitoreo(t, repo)
	comp.IDMensaje = uuid.New()
	comp.IDCuenta = other.IDCuenta
	if _, e = repo.ProcesarMovimiento(ctx, comp); !errors.Is(e, ErrMovimientoNoEncontrado) {
		t.Fatalf("another account received refund: %v", e)
	}
	comp.IDCuenta = account.IDCuenta
	comp.IDOperacion = uuid.New()
	comp.IDMensaje = uuid.New()
	if _, e = repo.ProcesarMovimiento(ctx, comp); !errors.Is(e, ErrMovimientoNoEncontrado) {
		t.Fatalf("missing debit accepted: %v", e)
	}
}

func TestCompensacionConcurrenteDatabase(t *testing.T) {
	ctx := context.Background()
	pool := baseMonitoreo(t)
	repo := NuevoRepositorioCuentasPostgres(pool)
	account := cuentaMonitoreo(t, repo)
	if _, err := pool.Exec(ctx, `UPDATE cuentas SET comision_transaccion_centavos=1000 WHERE id_cuenta=$1`, account.IDCuenta); err != nil {
		t.Fatal(err)
	}
	debit := solicitudMonitoreo(account.IDCuenta, models.TipoMovimientoDebito)
	debit.MontoCentavos = 10000
	if _, err := repo.ProcesarMovimiento(ctx, debit); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			comp := solicitudMonitoreo(account.IDCuenta, models.TipoMovimientoCompensacion)
			comp.IDOperacion = debit.IDOperacion
			comp.MontoCentavos = 10000
			for attempt := 0; attempt < 15; attempt++ {
				_, err := repo.ProcesarMovimiento(ctx, comp)
				if err == nil {
					return
				}
				var pgerr *pgconn.PgError
				if !errors.As(err, &pgerr) || pgerr.Code != "40001" {
					t.Errorf("concurrent compensation: %v", err)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Error("serialization retry exhausted")
		}()
	}
	wg.Wait()
	stored, err := repo.BuscarPorID(ctx, account.IDCuenta)
	if err != nil || stored.SaldoCentavos != 100000 {
		t.Fatalf("duplicate refund balance=%+v err=%v", stored, err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM movimientos_cuenta WHERE id_operacion=$1 AND tipo_movimiento='COMPENSACION'`, debit.IDOperacion).Scan(&count); err != nil || count != 1 {
		t.Fatalf("refunds=%d err=%v", count, err)
	}
}
