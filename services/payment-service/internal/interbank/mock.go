package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/payment-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/payment-service/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"strings"
	"time"
)

type Service struct {
	DB                  *pgxpool.Pool
	Rabbit, PartnerBank string
}
type mockTransfer struct {
	ID          string    `json:"interbankTransferId"`
	Amount      string    `json:"amount"`
	Destination string    `json:"destinationBankCode"`
	Expires     time.Time `json:"expiresAt"`
}

func (s *Service) mock(ctx context.Context, r Request) bankrpc.Response {
	// Each deposit snapshots its own scenario: concurrent clients cannot change each other's tests.
	if strings.HasSuffix(r.Path, "/resume") && r.Method == "POST" {
		id := strings.TrimSuffix(strings.TrimPrefix(r.Path, "/transfers/"), "/resume")
		var t mockTransfer
		if json.Unmarshal(r.Body, &t) != nil || t.ID != id {
			return bankrpc.Reply(400, nil)
		}
		_, err := s.DB.Exec(ctx, `INSERT INTO mock_interbank(id,body,scenario,due,expires,trace_context,recovered) VALUES($1,$2,$3,clock_timestamp(),$4,$5,true) ON CONFLICT(id) DO UPDATE SET recovered=true,due=clock_timestamp(),callback_delivered=false WHERE mock_interbank.scenario=excluded.scenario`, id, r.Body, r.Scenario, t.Expires, telemetry.Serialize(ctx))
		if err != nil {
			return bankrpc.Reply(503, nil)
		}
		return bankrpc.Reply(200, map[string]bool{"restored": true})
	}
	if r.Path == "/transfers" && r.Method == "POST" {
		var t mockTransfer
		if json.Unmarshal(r.Body, &t) != nil || t.ID != r.Key || t.Expires.IsZero() {
			return bankrpc.Reply(400, nil)
		}
		if _, e := uuid.Parse(t.ID); e != nil {
			return bankrpc.Reply(400, nil)
		}
		ctx = telemetry.WithCorrelation(ctx, t.ID)
		var recovered bool
		_ = s.DB.QueryRow(ctx, `SELECT recovered FROM mock_interbank WHERE id=$1`, t.ID).Scan(&recovered)
		if r.Scenario == "unavailable" && !recovered {
			slog.ErrorContext(ctx, "simulated interbank external failure", "interbankTransferId", t.ID)
			return bankrpc.Reply(503, map[string]string{"code": "SERVICE_UNAVAILABLE"})
		}
		delay := time.Second
		if r.Scenario == "delayed" {
			delay = 181 * time.Second
		}
		tag, e := s.DB.Exec(ctx, `INSERT INTO mock_interbank(id,body,scenario,due,expires,trace_context) VALUES($1,$2,$3,clock_timestamp()+$4*interval '1 millisecond',$5,$6) ON CONFLICT DO NOTHING`, t.ID, r.Body, r.Scenario, delay.Milliseconds(), t.Expires, telemetry.Serialize(ctx))
		if e != nil {
			return bankrpc.Reply(503, nil)
		}
		var stored json.RawMessage
		var scenario, state, reason string
		var updated time.Time
		e = s.DB.QueryRow(ctx, `SELECT body,scenario,status,reason,updated FROM mock_interbank WHERE id=$1`, t.ID).Scan(&stored, &scenario, &state, &reason, &updated)
		var a, b any
		_ = json.Unmarshal(stored, &a)
		_ = json.Unmarshal(r.Body, &b)
		canonicalA, _ := json.Marshal(a)
		canonicalB, _ := json.Marshal(b)
		if e != nil {
			return bankrpc.Reply(503, nil)
		}
		if string(canonicalA) != string(canonicalB) || scenario != r.Scenario {
			return bankrpc.Reply(409, map[string]string{"code": "IDEMPOTENCY_CONFLICT"})
		}
		code := 200
		if tag.RowsAffected() == 1 {
			code = 202
		}
		return bankrpc.Reply(code, map[string]any{"interbankTransferId": t.ID, "status": state, "reasonCode": reason, "receivedAt": updated, "updatedAt": updated})
	}
	parts := strings.Split(strings.Trim(r.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "transfers" {
		return bankrpc.Reply(400, nil)
	}
	id, e := uuid.Parse(parts[1])
	if e != nil {
		return bankrpc.Reply(400, nil)
	}
	ctx = telemetry.WithCorrelation(ctx, id.String())
	if len(parts) == 3 && parts[2] == "cancel" && r.Method == "POST" {
		// A cancellation tombstone prevents any delayed/retried request from being accepted later.
		var empty = json.RawMessage(`{}`)
		_, e = s.DB.Exec(ctx, `INSERT INTO mock_interbank(id,body,scenario,status,reason,due,expires,trace_context) VALUES($1,$2,$3,'CANCELLED','CANCELLED',clock_timestamp(),clock_timestamp(),$4) ON CONFLICT(id) DO UPDATE SET status=CASE WHEN mock_interbank.status='PROCESSING' THEN 'CANCELLED' ELSE mock_interbank.status END,reason=CASE WHEN mock_interbank.status='PROCESSING' THEN 'CANCELLED' ELSE mock_interbank.reason END`, id, empty, r.Scenario, telemetry.Serialize(ctx))
		if e != nil {
			return bankrpc.Reply(503, nil)
		}
	}
	var state, reason, scenario string
	var updated time.Time
	e = s.DB.QueryRow(ctx, `SELECT status,reason,scenario,updated FROM mock_interbank WHERE id=$1`, id).Scan(&state, &reason, &scenario, &updated)
	if errors.Is(e, pgx.ErrNoRows) {
		return bankrpc.Reply(404, map[string]string{"code": "NOT_FOUND"})
	}
	if e != nil {
		return bankrpc.Reply(503, nil)
	}
	if scenario != r.Scenario {
		return bankrpc.Reply(409, nil)
	}
	if len(parts) == 3 && parts[2] == "cancel" {
		result := "CANCELLED"
		if state == "ACCEPTED" {
			result = "ALREADY_ACCEPTED"
		}
		if state == "REJECTED" {
			result = "ALREADY_REJECTED"
		}
		return bankrpc.Reply(200, map[string]string{"interbankTransferId": id.String(), "result": result})
	}
	if r.Method != "GET" || len(parts) != 2 {
		return bankrpc.Reply(400, nil)
	}
	return bankrpc.Reply(200, map[string]any{"interbankTransferId": id.String(), "status": state, "reasonCode": reason, "updatedAt": updated})
}
func (s *Service) RunMock(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e := s.StepMock(ctx); e != nil {
				slog.ErrorContext(ctx, "interbank mock processing failed")
			}
		}
	}
}
func (s *Service) StepMock(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var id, scenario, state, reason, traceContext string
	var expires, updated time.Time
	e = tx.QueryRow(ctx, `SELECT id::text,scenario,status,reason,trace_context,expires,updated FROM mock_interbank WHERE NOT callback_delivered AND due<=clock_timestamp() ORDER BY due FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&id, &scenario, &state, &reason, &traceContext, &expires, &updated)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	ctx = telemetry.WithCorrelation(telemetry.Restore(ctx, traceContext), id)
	if state == "PROCESSING" {
		state = "ACCEPTED"
		if scenario == "reject" {
			state = "REJECTED"
			reason = "PROCESSING_ERROR"
		}
		if !time.Now().Before(expires) {
			state = "REJECTED"
			reason = "EXPIRED"
		}
		updated = time.Now().UTC()
		_, e = tx.Exec(ctx, `UPDATE mock_interbank SET status=$2,reason=$3,updated=$4,due=clock_timestamp()+interval '2 seconds' WHERE id=$1`, id, state, reason, updated)
		if e != nil {
			return e
		}
		slog.InfoContext(ctx, "simulated interbank state changed", "interbankTransferId", id, "status", state)
		// Commit the durable result before callback; a failed callback never reprocesses money.
		return tx.Commit(ctx)
	}
	var recovered bool
	if e = tx.QueryRow(ctx, `SELECT recovered FROM mock_interbank WHERE id=$1`, id).Scan(&recovered); e != nil {
		return e
	}
	delivered := (scenario == "no_callback" && !recovered) || state == "CANCELLED"
	if !delivered {
		result := map[string]any{"interbankTransferId": id, "destinationBankCode": s.PartnerBank, "status": state, "reasonCode": reason, "processedAt": updated}
		r, err := bankrpc.Call(ctx, s.Rabbit, "transacciones.interbancarias", map[string]any{"action": "result", "id": id, "peer": s.PartnerBank, "body": result})
		delivered = err == nil && r.Status == 200
	}
	_, e = tx.Exec(ctx, `UPDATE mock_interbank SET callback_delivered=$2,due=clock_timestamp()+interval '5 seconds' WHERE id=$1`, id, delivered)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
