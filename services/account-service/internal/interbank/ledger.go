package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/account-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/account-service/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"math"
	"time"
)

type Command struct {
	Original string    `json:"original,omitempty"`
	ID       string    `json:"id"`
	Action   string    `json:"action"`
	Account  string    `json:"account"`
	Customer string    `json:"customer"`
	Cents    int64     `json:"cents"`
	Expires  time.Time `json:"expires"`
}
type Result struct {
	State     string    `json:"state"`
	Reason    string    `json:"reason,omitempty"`
	Processed time.Time `json:"processedAt"`
}
type Ledger struct{ DB *pgxpool.Pool }

func (l Ledger) Handle(ctx context.Context, raw json.RawMessage) bankrpc.Response {
	var cmd Command
	if json.Unmarshal(raw, &cmd) != nil {
		return bankrpc.Reply(400, Result{Reason: "INVALID_COMMAND"})
	}
	result, err := l.Apply(ctx, cmd)
	if err != nil {
		return bankrpc.Reply(503, Result{Reason: "SERVICE_UNAVAILABLE"})
	}
	return bankrpc.Reply(200, result)
}

// Apply serializes each transfer before locking its account. Ledger, balance,
// movement and audit Outbox are committed together, including rejected credits.
func (l Ledger) Apply(ctx context.Context, c Command) (Result, error) {
	ctx = telemetry.WithCorrelation(ctx, c.ID)
	ctx, finish := telemetry.Database(ctx, "InterbankLedger")
	defer finish()
	c.Expires = c.Expires.UTC().Truncate(time.Microsecond)
	if _, err := uuid.Parse(c.ID); err != nil {
		return Result{Reason: "INVALID_ID"}, nil
	}
	if c.Cents < 1 || c.Cents > 1000000 || c.Account == "" || c.Expires.IsZero() {
		return Result{Reason: "AMOUNT_LIMIT_EXCEEDED"}, nil
	}
	if c.Action != "reserve" && c.Action != "credit" && c.Action != "settle" && c.Action != "release" && c.Action != "cancel_credit" && c.Action != "refund_reserve" && c.Action != "refund_received" {
		return Result{Reason: "INVALID_ACTION"}, nil
	}
	if c.Action == "reserve" {
		if _, err := uuid.Parse(c.Customer); err != nil {
			return Result{Reason: "OWNER_MISMATCH"}, nil
		}
	}
	tx, err := l.DB.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	refund := c.Action == "refund_reserve" || c.Action == "refund_received"
	if refund {
		if id, e := uuid.Parse(c.Original); e != nil || id == uuid.Nil || c.Original == c.ID {
			return Result{State: "REJECTED", Reason: "INVALID_ORIGINAL"}, nil
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, c.Original); err != nil {
			return Result{}, err
		}
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, c.ID); err != nil {
		return Result{}, err
	}
	var old Command
	var result Result
	var fee int64
	err = tx.QueryRow(ctx, `SELECT cuenta,cliente,monto,comision,estado,motivo,vence,actualizado,COALESCE(original::text,'') FROM operaciones_interbancarias WHERE id=$1`, c.ID).Scan(&old.Account, &old.Customer, &old.Cents, &fee, &result.State, &result.Reason, &old.Expires, &result.Processed, &old.Original)
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Result{}, err
	}
	if exists {
		if c.Action == "cancel_credit" && old.Account == c.Account && old.Cents == c.Cents && old.Expires.Equal(c.Expires) {
			return result, nil
		}
		if old.Account != c.Account || old.Customer != c.Customer || old.Cents != c.Cents || !old.Expires.Equal(c.Expires) || old.Original != c.Original {
			return Result{Reason: "IDEMPOTENCY_CONFLICT"}, nil
		}
		if c.Action == "reserve" || c.Action == "credit" || refund || result.State != "RESERVED" {
			return result, nil
		}
	} else if c.Action == "settle" || c.Action == "release" {
		return Result{Reason: "NOT_FOUND"}, nil
	}
	var accountID, owner, state string
	var balance, held, minimum, commission int64
	err = tx.QueryRow(ctx, `SELECT id_cuenta::text,id_cliente::text,estado,saldo_centavos,saldo_reservado_centavos,saldo_minimo_centavos,comision_transaccion_centavos FROM cuentas WHERE numero_cuenta=$1 FOR UPDATE`, c.Account).Scan(&accountID, &owner, &state, &balance, &held, &minimum, &commission)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Result{}, err
	}
	reason := ""
	if !exists {
		if refund {
			var originalState, originalAccount, originalCustomer, originalAction string
			var originalCents int64
			if e := tx.QueryRow(ctx, `SELECT estado,cuenta,cliente,monto,accion FROM operaciones_interbancarias WHERE id=$1`, c.Original).Scan(&originalState, &originalAccount, &originalCustomer, &originalCents, &originalAction); e != nil {
				return Result{State: "REJECTED", Reason: "ORIGINAL_NOT_FOUND"}, nil
			}
			wanted := "CREDITED"
			if c.Action == "refund_received" {
				wanted = "RELEASED"
			}
			if originalState != wanted || originalAccount != c.Account || originalCents != c.Cents || originalAction == "refund_received" || (c.Action == "refund_received" && originalCustomer != c.Customer) {
				return Result{State: "REJECTED", Reason: "INVALID_ORIGINAL"}, nil
			}
			var duplicate bool
			if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM operaciones_interbancarias WHERE original=$1)`, c.Original).Scan(&duplicate); e != nil {
				return Result{}, e
			}
			if duplicate {
				return Result{State: "REJECTED", Reason: "DUPLICATE_REFUND"}, nil
			}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			reason = "ACCOUNT_NOT_FOUND"
		} else if state != "ACTIVA" {
			reason = "ACCOUNT_INACTIVE"
		}
		if refund && accountID != "" {
			reason = ""
		}
		if c.Action == "reserve" && reason == "" && owner != c.Customer {
			reason = "OWNER_MISMATCH"
		}
		// clock_timestamp is evaluated AFTER acquiring the account lock, not at receipt.
		var now time.Time
		if e := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
			return Result{}, e
		}
		if !now.Before(c.Expires) {
			reason = "EXPIRED"
		}
		if c.Action == "reserve" && reason == "" {
			fee = commission
			if fee > math.MaxInt64-c.Cents || balance-held-minimum < c.Cents+fee {
				reason = "INSUFFICIENT_FUNDS"
			}
		}
		if c.Action == "refund_reserve" && reason == "" && balance-held < c.Cents {
			reason = "INSUFFICIENT_FUNDS"
		}
		if c.Action == "cancel_credit" {
			reason = "CANCELLED"
		}
		if c.Action == "credit" && reason == "" && balance > math.MaxInt64-c.Cents {
			reason = "PROCESSING_ERROR"
		}
		result = Result{State: "REJECTED", Reason: reason, Processed: now}
		if reason == "" {
			if c.Action == "credit" || c.Action == "refund_received" {
				result.State = "CREDITED"
			} else {
				result.State = "RESERVED"
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO operaciones_interbancarias(id,cuenta,cliente,monto,comision,estado,motivo,vence,actualizado,original,accion) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,$11)`, c.ID, c.Account, c.Customer, c.Cents, fee, result.State, result.Reason, c.Expires, now, c.Original, c.Action)
		if err != nil {
			return Result{}, err
		}
	}
	before := balance
	movement := ""
	amount := c.Cents
	if result.Reason == "" {
		switch c.Action {
		case "reserve", "refund_reserve":
			held += c.Cents + fee
		case "credit":
			balance += c.Cents
			movement = "CREDITO"
		case "settle":
			held -= c.Cents + fee
			balance -= c.Cents + fee
			amount += fee
			movement = "DEBITO"
			result.State = "DEBITED"
		case "release":
			held -= c.Cents + fee
			result.State = "RELEASED"
		}
		tag, e := tx.Exec(ctx, `UPDATE cuentas SET saldo_centavos=$2,saldo_reservado_centavos=$3,version=version+1,fecha_actualizacion=clock_timestamp(),ultima_actividad=clock_timestamp() WHERE id_cuenta=$1 AND ($4 NOT IN ('credit','refund_received') OR clock_timestamp()<$5)`, accountID, balance, held, c.Action, c.Expires)
		err = e
		if err != nil {
			return Result{}, err
		}
		if tag.RowsAffected() != 1 {
			return Result{}, errors.New("credit window expired; retry to persist rejection")
		}
		if movement != "" {
			_, err = tx.Exec(ctx, `INSERT INTO movimientos_cuenta(id_cuenta,id_operacion,id_correlacion,tipo_movimiento,monto_centavos,saldo_anterior_centavos,saldo_nuevo_centavos,descripcion) VALUES($1,$2,$2,$3,$4,$5,$6,'Deposito interbancario')`, accountID, c.ID, movement, amount, before, balance)
			if err != nil {
				return Result{}, err
			}
		}
	}
	if exists {
		_, err = tx.Exec(ctx, `UPDATE operaciones_interbancarias SET estado=$2,actualizado=clock_timestamp() WHERE id=$1`, c.ID, result.State)
		if err != nil {
			return Result{}, err
		}
	}
	payload, _ := json.Marshal(map[string]string{"interbankTransferId": c.ID, "status": result.State, "reasonCode": result.Reason})
	_, err = tx.Exec(ctx, `INSERT INTO mensajes_salida(tipo_evento,contenido,id_correlacion,trace_context) VALUES('cuenta.interbancaria.actualizada',$1,$2,$3)`, payload, c.ID, telemetry.Serialize(ctx))
	if err != nil {
		return Result{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return result, nil
}
