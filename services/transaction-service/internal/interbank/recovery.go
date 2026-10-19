package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"time"
)

func incident(ctx context.Context, tx pgx.Tx, id, kind string) error {
	tag, e := tx.Exec(ctx, `INSERT INTO incidentes_interbancarios(transferencia,tipo) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, kind)
	if e != nil || tag.RowsAffected() == 0 {
		return e
	}
	body, _ := json.Marshal(map[string]string{"interbankTransferId": id, "codigo": kind, "nivel": "ERROR"})
	_, e = tx.Exec(ctx, `INSERT INTO mensajes_salida(id_mensaje,tipo_evento,contenido,id_correlacion,es_comando,trace_context) VALUES($1,'interbancaria.incidente',$2,$3,false,$4)`, uuid.New(), body, id, telemetry.Serialize(ctx))
	if e == nil {
		slog.ErrorContext(ctx, "interbank incident", "interbankTransferId", id, "code", kind)
	}
	return e
}

func (s *Service) cancel(ctx context.Context, r Request) bankrpc.Response {
	var c Cancel
	if id, e := uuid.Parse(r.ID); e != nil || id == uuid.Nil || len(r.ID) != 36 || json.Unmarshal(r.Body, &c) != nil || (c.Reason != "TIMEOUT" && c.Reason != "CLIENT_REQUEST") {
		return failure(400, "VALIDATION_ERROR")
	}
	if r.Peer != s.Config.Partner || c.Origin != r.Peer {
		return failure(401, "UNAUTHORIZED")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	defer tx.Rollback(ctx)
	// Same lock/order as receiving the original request, including unknown ids.
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, r.ID); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	row, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1 FOR UPDATE`, r.ID))
	found := e == nil
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	outcome := "CANCELLED"
	if found {
		before := row.State
		if row.Direction != "IN" || row.Transfer.Origin != r.Peer {
			return failure(401, "UNAUTHORIZED")
		}
		switch row.State {
		case "ACCEPTED":
			outcome = "ALREADY_ACCEPTED"
		case "REJECTED":
			outcome = "ALREADY_REJECTED"
		case "PROCESSING":
			n, _ := Cents(row.Transfer.Amount)
			if n >= 1 && n <= 1000000 {
				receipt, err := s.account(ctx, AccountCommand{ID: r.ID, Action: "cancel_credit", Account: row.Transfer.DestinationAccount, Cents: n, Expires: row.Transfer.Expires})
				if err != nil {
					return failure(503, "SERVICE_UNAVAILABLE")
				}
				// The account may have committed before a coordinator crash. Never
				// acknowledge CANCELLED while a durable credit already exists.
				if receipt.State == "CREDITED" {
					outcome = "ALREADY_ACCEPTED"
					row.State = "ACCEPTED"
					row.Reason = ""
					row.Result = &Result{ID: r.ID, Destination: s.Config.Bank, Status: "ACCEPTED", Processed: receipt.Processed.UTC()}
				} else if receipt.State != "REJECTED" {
					return failure(503, "SERVICE_UNAVAILABLE")
				}
			}
		}
		if outcome == "CANCELLED" {
			row.State = "CANCELLED"
			row.Reason = "CANCELLED"
			row.Result = &Result{ID: r.ID, Destination: s.Config.Bank, Status: "REJECTED", Reason: "CANCELLED", Processed: s.now()}
		}
		if row.Result != nil && row.State != before {
			result, _ := json.Marshal(row.Result)
			_, e = tx.Exec(ctx, `UPDATE depositos_interbancarios SET estado=$2,motivo=$3,resultado=$4,actualizado=clock_timestamp(),proximo=clock_timestamp() WHERE id=$1`, r.ID, row.State, row.Reason, result)
			if e != nil {
				return failure(503, "SERVICE_UNAVAILABLE")
			}
		}
	}
	if outcome == "CANCELLED" {
		_, e = tx.Exec(ctx, `INSERT INTO cancelaciones_interbancarias(id,banco) VALUES($1,$2) ON CONFLICT DO NOTHING`, r.ID, r.Peer)
		if e != nil {
			return failure(503, "SERVICE_UNAVAILABLE")
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(200, map[string]string{"interbankTransferId": r.ID, "result": outcome})
}

func (s *Service) resolveTimeout(ctx context.Context, tx pgx.Tx, row record) (*Result, error) {
	t := row.Transfer
	resolved := func(status, reason string) *Result {
		return &Result{ID: t.ID, Destination: s.Config.Partner, Status: status, Reason: reason, Processed: s.now()}
	}
	code, b, err := s.get(ctx, "/transfers/"+t.ID)
	queryUnavailable := err != nil || code >= 500 || code == 0
	if err == nil && code == 200 {
		var response struct {
			ID     string `json:"interbankTransferId"`
			Status string `json:"status"`
			Reason string `json:"reasonCode"`
		}
		if json.Unmarshal(b, &response) == nil && response.ID == t.ID {
			switch response.Status {
			case "ACCEPTED":
				return resolved("ACCEPTED", ""), nil
			case "REJECTED", "CANCELLED":
				return resolved("REJECTED", response.Reason), nil
			case "PROCESSING":
			default:
				queryUnavailable = true
			}
		} else {
			queryUnavailable = true
		}
	}
	// Try cancellation even when the status endpoint is unavailable. A valid
	// cancellation is definitive; a network failure is never an immediate refund.
	cancelCode, cancelBody, cancelErr := s.Partner.Post(ctx, "/transfers/"+t.ID+"/cancel", "", Cancel{Origin: s.Config.Bank, Reason: "TIMEOUT"})
	cancelUnavailable := cancelErr != nil || cancelCode >= 500 || cancelCode == 0
	if cancelErr == nil && cancelCode == 200 {
		var response struct {
			ID     string `json:"interbankTransferId"`
			Result string `json:"result"`
		}
		if json.Unmarshal(cancelBody, &response) == nil && response.ID == t.ID {
			switch response.Result {
			case "ALREADY_ACCEPTED":
				return resolved("ACCEPTED", ""), nil
			case "CANCELLED", "ALREADY_REJECTED":
				return resolved("REJECTED", "CANCELLED"), nil
			default:
				cancelUnavailable = true
			}
		} else {
			cancelUnavailable = true
		}
	}
	if !s.now().Before(t.Expires.Add(300 * time.Second)) {
		kind := "TIMEOUT_REQUIRES_REVIEW"
		if queryUnavailable && cancelUnavailable {
			kind = "TIMEOUT_MARGIN_RELEASE"
		}
		if e := incident(ctx, tx, t.ID, kind); e != nil {
			return nil, e
		}
		if queryUnavailable && cancelUnavailable {
			return resolved("REJECTED", "PROCESSING_ERROR"), nil
		}
	}
	return nil, nil
}

func (s *Service) needReversal(ctx context.Context, tx pgx.Tx, row record) error {
	t := row.Transfer
	if row.RefundOf != "" {
		return incident(ctx, tx, t.ID, "LATE_ACCEPTED_REFUND_REQUIRES_REVIEW")
	}
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("bank-usac-reversal:"+t.ID)).String()
	request := Reversal{ID: id, Original: t.ID, Origin: s.Config.Bank, Amount: t.Amount, Currency: t.Currency}
	body, _ := json.Marshal(request)
	_, e := tx.Exec(ctx, `INSERT INTO reversos_interbancarios(id,original,direccion,solicitud,trace_context) VALUES($1,$2,'OUT',$3,$4) ON CONFLICT(original) DO NOTHING`, id, t.ID, body, telemetry.Serialize(ctx))
	if e != nil {
		return e
	}
	return incident(ctx, tx, t.ID, "LATE_ACCEPTED_AFTER_RELEASE")
}
