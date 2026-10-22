package interbank

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Service struct {
	DB      *pgxpool.Pool
	Config  Config
	Account func(context.Context, AccountCommand) (AccountResult, error)
	Partner Partner
	Now     func() time.Time
	KYC     func(context.Context, string, string) (bool, error)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func failure(status int, code string) bankrpc.Response {
	return bankrpc.Reply(status, map[string]any{"code": code, "message": code, "timestamp": time.Now().UTC()})
}
func (s *Service) Handle(ctx context.Context, raw json.RawMessage) bankrpc.Response {
	var r Request
	if strict(raw, &r) != nil {
		return failure(400, "VALIDATION_ERROR")
	}
	switch r.Action {
	case "cancel":
		return s.cancel(ctx, r)
	case "reversal":
		return s.requestReversal(ctx, r)
	case "reversal_get":
		return s.reversalStatus(ctx, r)
	case "reconciliation":
		return s.reconciliation(ctx, r)
	case "receive":
		var t Transfer
		if strict(r.Body, &t) != nil || !t.Valid() || r.Key != t.ID || t.Origin != s.Config.Partner || r.Peer != s.Config.Partner || t.Destination != s.Config.Bank {
			return failure(400, "VALIDATION_ERROR")
		}
		return s.create(ctx, t, "IN", "")
	case "send", "simulate":
		if r.Action == "simulate" {
			if !simulationEnabled() {
				return failure(403, "SIMULATION_DISABLED")
			}
			switch r.Scenario {
			case "reject", "unavailable", "no_callback", "delayed":
			default:
				return failure(400, "VALIDATION_ERROR")
			}
		} else if r.Scenario != "" {
			return failure(400, "VALIDATION_ERROR")
		}
		if _, e := uuid.Parse(r.Customer); e != nil {
			return failure(401, "UNAUTHORIZED")
		}
		if r.Action != "simulate" && !s.partnerReady() {
			return failure(503, "PARTNER_NOT_CONFIGURED")
		}
		var t Transfer
		if strict(r.Body, &t) != nil {
			return failure(400, "VALIDATION_ERROR")
		}
		// Internal client endpoint accepts the same TransferRequest fields, but bank
		// codes and the validity window must be authored by this bank, not the caller.
		t.Origin = s.Config.Bank
		t.Destination = s.Config.Partner
		t.Created = time.Now().UTC().Truncate(time.Second)
		t.Expires = t.Created.Add(180 * time.Second)
		if !t.Valid() || r.Key != t.ID {
			return failure(400, "VALIDATION_ERROR")
		}
		n, _ := Cents(t.Amount)
		if n < 1 || n > 1000000 {
			return failure(400, "VALIDATION_ERROR")
		}
		return s.createScenario(ctx, t, "OUT", r.Customer, r.Scenario)
	case "result":
		var result Result
		if strict(r.Body, &result) != nil || !result.Valid() || result.ID != r.ID || result.Destination != s.Config.Partner || r.Peer != s.Config.Partner {
			return failure(400, "VALIDATION_ERROR")
		}
		return s.callback(ctx, result)
	case "client_list":
		return s.listClient(ctx, r)
	case "get", "client_get":
		return s.lookup(ctx, r)
	default:
		return failure(400, "VALIDATION_ERROR")
	}
}

type record struct {
	Transfer                                  Transfer
	Direction, Customer, State, Reason, Trace string
	Result                                    *Result
	Received, Updated                         time.Time
	Attempts                                  int
	RefundOf                                  string
	Scenario                                  string
}

func scan(row pgx.Row) (record, error) {
	var r record
	var body, result []byte
	e := row.Scan(&body, &r.Direction, &r.Customer, &r.State, &r.Reason, &result, &r.Trace, &r.Received, &r.Updated, &r.Attempts, &r.RefundOf, &r.Scenario)
	if e != nil {
		return r, e
	}
	e = json.Unmarshal(body, &r.Transfer)
	if e == nil && len(result) > 0 {
		e = json.Unmarshal(result, &r.Result)
	}
	return r, e
}

const columns = `solicitud,direccion,cliente,estado,motivo,resultado,trace_context,recibido,actualizado,intentos,COALESCE(refund_of::text,''),scenario`

func (s *Service) create(ctx context.Context, t Transfer, direction, customer string) bankrpc.Response {
	return s.createScenario(ctx, t, direction, customer, "")
}
func (s *Service) createScenario(ctx context.Context, t Transfer, direction, customer, scenario string) bankrpc.Response {
	ctx, finishTrace := telemetry.StartTransfer(ctx, t.ID, t.ID)
	defer finishTrace()
	// Timestamp normalization permits different ISO8601 offsets for the same instant.
	t.Created = t.Created.UTC().Truncate(time.Microsecond)
	t.Expires = t.Expires.UTC().Truncate(time.Microsecond)
	body, _ := json.Marshal(t)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, t.ID); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	existing, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, t.ID))
	if e == nil {
		a, b := existing.Transfer, t
		// Retries of an outgoing request keep the original server-generated window.
		if direction == "OUT" {
			b.Created = a.Created
			b.Expires = a.Expires
		}
		ab, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if string(ab) != string(bb) || existing.Customer != customer || existing.Direction != direction || existing.Scenario != scenario {
			return failure(409, "IDEMPOTENCY_CONFLICT")
		}
		return bankrpc.Reply(200, status(existing))
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if direction == "IN" {
		var cancelled bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cancelaciones_interbancarias WHERE id=$1 AND banco=$2)`, t.ID, t.Origin).Scan(&cancelled); e != nil {
			return failure(503, "SERVICE_UNAVAILABLE")
		}
		if cancelled {
			result, _ := json.Marshal(Result{ID: t.ID, Destination: s.Config.Bank, Status: "REJECTED", Reason: "CANCELLED", Processed: s.now()})
			_, e = tx.Exec(ctx, `INSERT INTO depositos_interbancarios(id,direccion,solicitud,huella,estado,motivo,resultado,trace_context) VALUES($1,'IN',$2,'','CANCELLED','CANCELLED',$3,$4)`, t.ID, body, result, telemetry.Serialize(ctx))
			if e != nil {
				return failure(503, "SERVICE_UNAVAILABLE")
			}
			if e = tx.Commit(ctx); e != nil {
				return failure(503, "SERVICE_UNAVAILABLE")
			}
			return bankrpc.Reply(200, map[string]any{"interbankTransferId": t.ID, "status": "CANCELLED", "reasonCode": "CANCELLED", "updatedAt": s.now()})
		}
	}
	if !time.Now().Before(t.Expires) {
		return failure(422, "EXPIRED")
	}
	state := "PROCESSING"
	if direction == "OUT" {
		state = "PENDING_RESERVATION"
	}
	digest := sha256.Sum256(body)
	_, e = tx.Exec(ctx, `INSERT INTO depositos_interbancarios(id,direccion,cliente,solicitud,huella,estado,trace_context,scenario) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, t.ID, direction, customer, body, hex.EncodeToString(digest[:]), state, telemetry.Serialize(ctx), scenario)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if e = stateEvent(ctx, tx, t.ID, state); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if e = tx.Commit(ctx); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(202, map[string]any{"interbankTransferId": t.ID, "status": "PROCESSING", "receivedAt": time.Now().UTC()})
}
func status(r record) map[string]any {
	b := map[string]any{"interbankTransferId": r.Transfer.ID, "status": r.State, "updatedAt": r.Updated.UTC()}
	b["simulation"] = r.Scenario != ""
	if r.Scenario != "" {
		b["scenario"] = r.Scenario
	}
	if r.Reason != "" {
		b["reasonCode"] = r.Reason
	}
	return b
}

func stateEvent(ctx context.Context, tx pgx.Tx, id, state string) error {
	body, _ := json.Marshal(map[string]string{"interbankTransferId": id, "status": state})
	_, e := tx.Exec(ctx, `INSERT INTO mensajes_salida(id_mensaje,tipo_evento,contenido,id_correlacion,es_comando,trace_context) VALUES($1,'interbancaria.estado.actualizado',$2,$3,false,$4)`, uuid.New(), body, id, telemetry.Serialize(ctx))
	return e
}
func (s *Service) lookup(ctx context.Context, r Request) bankrpc.Response {
	if _, e := uuid.Parse(r.ID); e != nil {
		return failure(400, "VALIDATION_ERROR")
	}
	row, e := scan(s.DB.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1`, r.ID))
	if errors.Is(e, pgx.ErrNoRows) {
		if r.Action == "get" && r.Peer == s.Config.Partner {
			var cancelled time.Time
			if err := s.DB.QueryRow(ctx, `SELECT creado FROM cancelaciones_interbancarias WHERE id=$1 AND banco=$2`, r.ID, r.Peer).Scan(&cancelled); err == nil {
				return bankrpc.Reply(200, map[string]any{"interbankTransferId": r.ID, "status": "CANCELLED", "updatedAt": cancelled.UTC()})
			}
		}
		return failure(404, "NOT_FOUND")
	}
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if r.Action == "get" && (r.Peer != s.Config.Partner || row.Direction != "IN") || r.Action == "client_get" && (row.Direction != "OUT" || row.Customer != r.Customer || r.Customer == "") {
		return failure(401, "UNAUTHORIZED")
	}
	return bankrpc.Reply(200, status(row))
}
func (s *Service) callback(ctx context.Context, result Result) bankrpc.Response {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	defer tx.Rollback(ctx)
	row, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1 FOR UPDATE`, result.ID))
	if errors.Is(e, pgx.ErrNoRows) {
		return failure(404, "NOT_FOUND")
	}
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if row.Direction != "OUT" {
		return failure(401, "UNAUTHORIZED")
	}
	if result.Status == "ACCEPTED" && row.State == "REVERSED" {
		if e = s.needReversal(ctx, tx, row); e != nil {
			return failure(503, "SERVICE_UNAVAILABLE")
		}
		if e = tx.Commit(ctx); e != nil {
			return failure(503, "SERVICE_UNAVAILABLE")
		}
		return bankrpc.Reply(200, map[string]any{"interbankTransferId": result.ID, "received": true})
	}
	if row.Result != nil {
		if row.Result.Status != result.Status || row.Result.Reason != result.Reason {
			if result.Status == "ACCEPTED" && row.State == "TIMEOUT" {
				b, _ := json.Marshal(result)
				_, e = tx.Exec(ctx, `UPDATE depositos_interbancarios SET resultado=$2,proximo=clock_timestamp() WHERE id=$1`, result.ID, b)
				if e != nil {
					return failure(503, "SERVICE_UNAVAILABLE")
				}
			} else {
				return failure(400, "VALIDATION_ERROR")
			}
		}
	} else {
		if row.State != "RESERVED" && row.State != "PROCESSING" && row.State != "TIMEOUT" {
			return failure(400, "VALIDATION_ERROR")
		}
		// ACCEPTED must refer to processing during the validity window.
		if result.Status == "ACCEPTED" && (result.Processed.Before(row.Transfer.Created) || result.Processed.After(row.Transfer.Expires)) {
			if e = incident(ctx, tx, result.ID, "PARTNER_CREDIT_OUTSIDE_WINDOW"); e != nil {
				return failure(503, "SERVICE_UNAVAILABLE")
			}
		}
		b, _ := json.Marshal(result)
		_, e = tx.Exec(ctx, `UPDATE depositos_interbancarios SET resultado=$2,proximo=clock_timestamp() WHERE id=$1`, result.ID, b)
		if e != nil {
			return failure(503, "SERVICE_UNAVAILABLE")
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(200, map[string]any{"interbankTransferId": result.ID, "received": true})
}

func (s *Service) listClient(ctx context.Context, r Request) bankrpc.Response {
	if id, e := uuid.Parse(r.Customer); e != nil || id == uuid.Nil {
		return failure(401, "UNAUTHORIZED")
	}
	rows, e := s.DB.Query(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE direccion='OUT' AND cliente=$1 ORDER BY recibido DESC LIMIT 50`, r.Customer)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		row, err := scan(rows)
		if err != nil {
			return failure(503, "SERVICE_UNAVAILABLE")
		}
		item := status(row)
		item["amount"] = row.Transfer.Amount
		item["destinationAccount"] = row.Transfer.DestinationAccount
		item["createdAt"] = row.Received
		items = append(items, item)
	}
	if rows.Err() != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(200, items)
}
