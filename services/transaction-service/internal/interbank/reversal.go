package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type reversalRow struct {
	Request                                 Reversal
	Direction, State, Refund, Detail, Trace string
	Attempts                                int
}

const reversalColumns = `solicitud,direccion,estado,COALESCE(refund_id::text,''),detalle,trace_context,intentos`

func scanReversal(row pgx.Row) (reversalRow, error) {
	var r reversalRow
	var body []byte
	e := row.Scan(&body, &r.Direction, &r.State, &r.Refund, &r.Detail, &r.Trace, &r.Attempts)
	if e == nil {
		e = json.Unmarshal(body, &r.Request)
	}
	return r, e
}
func reversalBody(r reversalRow) map[string]any {
	b := map[string]any{"reversalId": r.Request.ID, "status": r.State}
	if r.Refund != "" {
		b["refundTransferId"] = r.Refund
	}
	if r.Detail != "" {
		b["reasonMessage"] = r.Detail
	}
	return b
}

func (s *Service) requestReversal(ctx context.Context, r Request) bankrpc.Response {
	var request Reversal
	if json.Unmarshal(r.Body, &request) != nil || !request.Valid() {
		return failure(400, "VALIDATION_ERROR")
	}
	if request.Origin != s.Config.Partner || r.Peer != s.Config.Partner {
		return failure(401, "UNAUTHORIZED")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, request.Original); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	old, e := scanReversal(tx.QueryRow(ctx, `SELECT `+reversalColumns+` FROM reversos_interbancarios WHERE id=$1 OR original=$2`, request.ID, request.Original))
	if e == nil {
		if old.Direction != "IN" || old.Request != request {
			return failure(400, "VALIDATION_ERROR")
		}
		return bankrpc.Reply(202, reversalBody(old))
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	original, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, request.Original))
	if errors.Is(e, pgx.ErrNoRows) {
		return failure(404, "NOT_FOUND")
	}
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if original.Direction != "IN" || original.Transfer.Origin != r.Peer {
		return failure(401, "UNAUTHORIZED")
	}
	if original.State != "ACCEPTED" {
		return failure(404, "NOT_FOUND")
	}
	if original.Transfer.Amount != request.Amount || original.Transfer.Currency != request.Currency {
		return failure(400, "VALIDATION_ERROR")
	}
	// The id is fixed and published while PROCESSING. The requesting bank can
	// authenticate the refund deposit before acknowledging it, without guessing
	// from description, amount or account number alone.
	refund := uuid.NewSHA1(uuid.NameSpaceOID, []byte("bank-usac-refund:"+request.ID)).String()
	body, _ := json.Marshal(request)
	_, e = tx.Exec(ctx, `INSERT INTO reversos_interbancarios(id,original,direccion,solicitud,refund_id,trace_context) VALUES($1,$2,'IN',$3,$4,$5)`, request.ID, request.Original, body, refund, telemetry.Serialize(ctx))
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if e = tx.Commit(ctx); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(202, map[string]any{"reversalId": request.ID, "status": "PROCESSING", "refundTransferId": refund})
}
func (s *Service) reversalStatus(ctx context.Context, r Request) bankrpc.Response {
	if _, e := uuid.Parse(r.ID); e != nil {
		return failure(400, "VALIDATION_ERROR")
	}
	if r.Peer != s.Config.Partner {
		return failure(401, "UNAUTHORIZED")
	}
	row, e := scanReversal(s.DB.QueryRow(ctx, `SELECT `+reversalColumns+` FROM reversos_interbancarios WHERE id=$1 AND direccion='IN'`, r.ID))
	if errors.Is(e, pgx.ErrNoRows) {
		return failure(404, "NOT_FOUND")
	}
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(200, reversalBody(row))
}

func (s *Service) StepReversal(parent context.Context) (worked bool, resultErr error) {
	if !s.partnerReady() {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	row, e := scanReversal(tx.QueryRow(ctx, `SELECT `+reversalColumns+` FROM reversos_interbancarios WHERE estado='PROCESSING' AND proximo<=clock_timestamp() ORDER BY proximo FOR UPDATE SKIP LOCKED LIMIT 1`))
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	ctx = telemetry.WithCorrelation(telemetry.Restore(ctx, row.Trace), row.Request.Original)
	defer func() {
		if resultErr != nil {
			_ = tx.Rollback(ctx)
			retryctx, stop := context.WithTimeout(parent, 2*time.Second)
			defer stop()
			_, _ = s.DB.Exec(retryctx, `UPDATE reversos_interbancarios SET proximo=clock_timestamp()+interval '5 seconds' WHERE id=$1 AND intentos=$2`, row.Request.ID, row.Attempts)
		}
	}()
	original, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, row.Request.Original))
	if e != nil {
		return true, e
	}
	if row.Direction == "IN" {
		refund, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, row.Refund))
		if errors.Is(e, pgx.ErrNoRows) {
			now := s.now().Truncate(time.Second)
			transfer := Transfer{ID: row.Refund, Origin: s.Config.Bank, Destination: s.Config.Partner, OriginAccount: original.Transfer.DestinationAccount, DestinationAccount: original.Transfer.OriginAccount, Amount: row.Request.Amount, Currency: "GTQ", Description: "Reversal " + row.Request.ID, Created: now, Expires: now.Add(180 * time.Second)}
			body, _ := json.Marshal(transfer)
			_, e = tx.Exec(ctx, `INSERT INTO depositos_interbancarios(id,direccion,solicitud,huella,estado,trace_context,refund_of) VALUES($1,'OUT',$2,'','PENDING_RESERVATION',$3,$4)`, transfer.ID, body, telemetry.Serialize(ctx), row.Request.Original)
			if e != nil {
				return true, e
			}
		} else if e != nil {
			return true, e
		} else {
			switch refund.State {
			case "ACCEPTED":
				row.State = "COMPLETED"
			case "REVERSED", "REJECTED":
				row.State = "FAILED"
				row.Detail = "Refund was not accepted; manual review required"
			}
		}
	} else {
		// POST is itself idempotent. Repeating it after an ambiguous response is
		// safe, and lets us rediscover the refund id after a restart.
		code, body, err := s.Partner.Post(ctx, "/reversals", "", row.Request)
		if err == nil && (code == 202 || code == 200) {
			if err = learnReversal(&row, body); err != nil {
				return true, err
			}
		} else if err == nil && (code == 400 || code == 401 || code == 404) {
			if e = incident(ctx, tx, row.Request.Original, "REVERSAL_REQUIRES_REVIEW"); e != nil {
				return true, e
			}
		}
		code, body, err = s.get(ctx, "/reversals/"+row.Request.ID)
		if err == nil && code == 200 {
			if err = learnReversal(&row, body); err != nil {
				return true, err
			}
		}
		if row.State == "COMPLETED" {
			// A partner's COMPLETED is not evidence that our settlement receipt was
			// committed. Wait for that local receipt; never credit the customer twice.
			var accepted bool
			e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM depositos_interbancarios WHERE id=NULLIF($1,'')::uuid AND direccion='IN' AND estado='ACCEPTED' AND refund_of=$2)`, row.Refund, row.Request.Original).Scan(&accepted)
			if e != nil {
				return true, e
			}
			if !accepted {
				row.State = "PROCESSING"
			}
		}
	}
	if row.State == "FAILED" {
		if e = incident(ctx, tx, row.Request.Original, "REFUND_FAILED"); e != nil {
			return true, e
		}
	}
	_, e = tx.Exec(ctx, `UPDATE reversos_interbancarios SET estado=$2,refund_id=NULLIF($3,'')::uuid,detalle=$4,intentos=intentos+1,proximo=clock_timestamp()+$5*interval '1 millisecond',actualizado=clock_timestamp() WHERE id=$1`, row.Request.ID, row.State, row.Refund, row.Detail, retryDelay(row.Attempts).Milliseconds())
	if e != nil {
		return true, e
	}
	return true, tx.Commit(ctx)
}

func learnReversal(row *reversalRow, body []byte) error {
	var response struct {
		ID     string `json:"reversalId"`
		Status string `json:"status"`
		Refund string `json:"refundTransferId"`
	}
	if json.Unmarshal(body, &response) != nil || response.ID != row.Request.ID {
		return errors.New("invalid reversal response")
	}
	switch response.Status {
	case "PROCESSING", "COMPLETED", "FAILED":
	default:
		return errors.New("invalid reversal state")
	}
	if response.Refund != "" {
		if id, e := uuid.Parse(response.Refund); e != nil || id == uuid.Nil || len(response.Refund) != 36 {
			return errors.New("invalid refund id")
		}
		if row.Refund != "" && row.Refund != response.Refund {
			return errors.New("partner changed refund id")
		}
		row.Refund = response.Refund
	}
	if response.Status == "COMPLETED" && row.Refund == "" {
		return errors.New("completed reversal without refund id")
	}
	row.State = response.Status
	return nil
}

// refundReceipt distinguishes an expected bank-to-bank settlement return from
// a new client deposit. The original hold was already released: no second
// credit to that client is owed. Identification requires the partner's declared
// refundTransferId, not a caller-controlled description or an amount match.
func (s *Service) refundReceipt(ctx context.Context, tx pgx.Tx, in record, cmd *AccountCommand) error {
	unresolved := func(cause error) error {
		if s.now().Before(in.Transfer.Expires) {
			return cause
		}
		if e := incident(ctx, tx, in.Transfer.ID, "REFUND_IDENTITY_UNRESOLVED"); e != nil {
			return e
		}
		cmd.Action = "cancel_credit"
		return nil
	}
	rows, e := tx.Query(ctx, `SELECT `+reversalColumns+` FROM reversos_interbancarios r WHERE direccion='OUT' AND (refund_id=$1 OR original IN (SELECT id FROM depositos_interbancarios WHERE solicitud->>'originAccount'=$2 AND solicitud->>'amount'=$3 AND solicitud->>'destinationAccount'=$4))`, in.Transfer.ID, in.Transfer.DestinationAccount, in.Transfer.Amount, in.Transfer.OriginAccount)
	if e != nil {
		return e
	}
	var candidates []reversalRow
	for rows.Next() {
		r, e := scanReversal(rows)
		if e != nil {
			rows.Close()
			return e
		}
		candidates = append(candidates, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, r := range candidates {
		if r.Refund == "" {
			code, body, err := s.get(ctx, "/reversals/"+r.Request.ID)
			if err != nil || code != 200 {
				return unresolved(errors.New("refund identity unresolved"))
			}
			if e = learnReversal(&r, body); e != nil {
				return e
			}
			if r.Refund == "" {
				return unresolved(errors.New("partner must publish refundTransferId while PROCESSING"))
			}
			_, e = tx.Exec(ctx, `UPDATE reversos_interbancarios SET refund_id=$2 WHERE id=$1 AND (refund_id IS NULL OR refund_id=$2)`, r.Request.ID, r.Refund)
			if e != nil {
				return e
			}
		}
		if r.Refund == in.Transfer.ID {
			original, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, r.Request.Original))
			if e != nil {
				return e
			}
			if original.State != "REVERSED" || in.Transfer.Origin != s.Config.Partner || in.Transfer.DestinationAccount != original.Transfer.OriginAccount || in.Transfer.OriginAccount != original.Transfer.DestinationAccount || in.Transfer.Amount != original.Transfer.Amount {
				return errors.New("invalid refund transfer content")
			}
			cmd.Action = "refund_received"
			cmd.Original = r.Request.Original
			cmd.Customer = original.Customer
			_, e = tx.Exec(ctx, `UPDATE depositos_interbancarios SET refund_of=$2 WHERE id=$1`, in.Transfer.ID, r.Request.Original)
			return e
		}
	}
	return nil
}
