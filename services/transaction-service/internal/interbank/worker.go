package interbank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/telemetry"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Partner interface {
	Post(context.Context, string, string, any) (int, []byte, error)
}
type reader interface {
	Get(context.Context, string) (int, []byte, error)
}

func (s *Service) get(ctx context.Context, path string) (int, []byte, error) {
	if p, ok := s.Partner.(reader); ok {
		return p.Get(ctx, path)
	}
	return 0, nil, errors.New("partner reader unavailable")
}

type HTTPPartner struct {
	BaseURL, Key string
	Client       *http.Client
}

var ErrPartnerNotConfigured = errors.New("interbank outbound partner not configured")

func (p *HTTPPartner) Ready() bool { return p.BaseURL != "" && len(p.Key) >= 16 }

func (s *Service) partnerReady() bool {
	if s.Partner == nil {
		return false
	}
	if p, ok := s.Partner.(interface{ Ready() bool }); ok {
		return p.Ready()
	}
	return true // Alternate transports used by isolated tests.
}

func NewPartner(c Config) *HTTPPartner {
	return &HTTPPartner{BaseURL: c.PartnerURL, Key: c.PartnerKey, Client: &http.Client{Timeout: 5 * time.Second, Transport: telemetry.Transport{}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (p *HTTPPartner) Post(ctx context.Context, path, key string, body any) (int, []byte, error) {
	b, e := json.Marshal(body)
	if e != nil {
		return 0, nil, e
	}
	return p.request(ctx, "POST", path, key, b)
}
func (p *HTTPPartner) Get(ctx context.Context, path string) (int, []byte, error) {
	return p.request(ctx, "GET", path, "", nil)
}
func (p *HTTPPartner) request(ctx context.Context, method, path, key string, b []byte) (int, []byte, error) {
	if !p.Ready() {
		return 0, nil, ErrPartnerNotConfigured
	}
	req, e := http.NewRequestWithContext(ctx, method, p.BaseURL+path, bytes.NewReader(b))
	if e != nil {
		return 0, nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", p.Key)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, e := p.Client.Do(req)
	if e != nil {
		return 0, nil, e
	}
	defer resp.Body.Close()
	data, e := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if len(data) > 65536 {
		return 0, nil, errors.New("partner response too large")
	}
	return resp.StatusCode, data, e
}
func (s *Service) account(ctx context.Context, c AccountCommand) (AccountResult, error) {
	if s.Account != nil {
		return s.Account(ctx, c)
	}
	r, e := bankrpc.Call(ctx, s.Config.RabbitURL, "cuentas.interbancarias", c)
	if e != nil {
		return AccountResult{}, e
	}
	if r.Status != 200 {
		return AccountResult{}, errors.New("account service unavailable")
	}
	var result AccountResult
	e = json.Unmarshal(r.Body, &result)
	return result, e
}
func (s *Service) Run(ctx context.Context) {
	for ctx.Err() == nil {
		worked, e := s.Step(ctx)
		if e != nil {
			slog.ErrorContext(ctx, "interbank processing failed", "error", e)
		}
		if w, err := s.StepReversal(ctx); err != nil {
			slog.ErrorContext(ctx, "interbank reversal processing failed", "error", err)
		} else {
			worked = worked || w
		}
		if w, err := s.StepReconciliation(ctx); err != nil {
			slog.ErrorContext(ctx, "interbank reconciliation failed", "error", err)
		} else {
			worked = worked || w
		}
		if !worked || e != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}

// Step locks a durable work row (SKIP LOCKED across replicas). External effects
// use the same transfer id on every retry, including after a process crash.
func (s *Service) Step(parent context.Context) (worked bool, failureErr error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	row, e := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE proximo<=clock_timestamp() AND (
 (direccion='IN' AND (estado='PROCESSING' OR (NOT callback_entregado AND actualizado>clock_timestamp()-interval '24 hours')))
 OR (direccion='OUT' AND estado IN ('PENDING_RESERVATION','RESERVED','PROCESSING','TIMEOUT')))
 ORDER BY proximo FOR UPDATE SKIP LOCKED LIMIT 1`))
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	ctx = telemetry.WithCorrelation(telemetry.Restore(ctx, row.Trace), row.Transfer.ID)
	ctx, span := otel.Tracer("bank-usac").Start(ctx, "interbank process "+row.Direction)
	defer span.End()
	// A failing peer/account must not starve other transfers at the head of
	// the work queue. Roll back first, then schedule this durable row again.
	defer func() {
		if failureErr != nil {
			_ = tx.Rollback(ctx)
			retryctx, stop := context.WithTimeout(parent, 2*time.Second)
			defer stop()
			_, _ = s.DB.Exec(retryctx, `UPDATE depositos_interbancarios SET proximo=clock_timestamp()+interval '5 seconds' WHERE id=$1 AND intentos=$2`, row.Transfer.ID, row.Attempts)
		}
	}()
	t := row.Transfer
	n, _ := Cents(t.Amount)
	cmd := AccountCommand{ID: t.ID, Cents: n, Expires: t.Expires}
	before := row.State
	delay := time.Second
	delivered := false
	if row.Direction == "IN" {
		if row.State == "PROCESSING" {
			row.Reason = ""
			row.State = "REJECTED"
			processed := time.Now().UTC()
			if n < 1 || n > 1000000 {
				row.Reason = "AMOUNT_LIMIT_EXCEEDED"
				if !time.Now().Before(t.Expires) {
					row.Reason = "EXPIRED"
				}
			} else {
				// Always consult the durable account receipt, even after expiry.
				// A credit committed before a coordinator crash must not later
				// be misreported as REJECTED/EXPIRED after money was credited.
				cmd.Action = "credit"
				cmd.Account = t.DestinationAccount
				if err := s.refundReceipt(ctx, tx, row, &cmd); err != nil {
					return true, err
				}
				a, err := s.account(ctx, cmd)
				if err != nil {
					return true, err
				}
				if !a.Processed.IsZero() {
					processed = a.Processed.UTC()
				}
				if a.State == "CREDITED" {
					row.State = "ACCEPTED"
				} else {
					row.Reason = a.Reason
					if cmd.Action == "cancel_credit" {
						row.Reason = "EXPIRED"
					}
					if row.Reason == "" {
						return true, errors.New("unexpected account credit state")
					}
				}
			}
			// Callback uses the timestamp of the actual credit commit, not the later
			// callback delivery attempt. Account rejects credit after expiry under lock.
			row.Result = &Result{ID: t.ID, Destination: s.Config.Bank, Status: row.State, Reason: row.Reason, Processed: processed}
		} else {
			if row.Result == nil {
				return true, errors.New("missing durable callback result")
			}
			code, b, err := 0, []byte(nil), ErrPartnerNotConfigured
			if s.partnerReady() {
				code, b, err = s.Partner.Post(ctx, "/transfers/"+t.ID+"/result", "", row.Result)
			}
			var ack struct {
				ID       string `json:"interbankTransferId"`
				Received bool   `json:"received"`
			}
			delivered = err == nil && code == 200 && json.Unmarshal(b, &ack) == nil && ack.ID == t.ID && ack.Received
			delay = retryDelay(row.Attempts)
			if errors.Is(err, ErrPartnerNotConfigured) {
				delay = time.Minute
			}
		}
	} else {
		cmd.Customer = row.Customer
		cmd.Account = t.OriginAccount
		cmd.Original = row.RefundOf
		switch {
		case row.Result != nil:
			cmd.Action = "release"
			if row.Result.Status == "ACCEPTED" {
				cmd.Action = "settle"
			}
			a, err := s.account(ctx, cmd)
			if err != nil {
				return true, err
			}
			if a.State == "DEBITED" {
				row.State = "ACCEPTED"
			} else if a.State == "RELEASED" {
				row.State = "REVERSED"
				row.Reason = row.Result.Reason
				if row.Result.Status == "ACCEPTED" {
					if err = s.needReversal(ctx, tx, row); err != nil {
						return true, err
					}
				}
			} else {
				return true, errors.New("unexpected reservation settlement state")
			}
		case !s.partnerReady():
			// A local missing URL is not a remote outage. Never reserve new funds
			// or release existing holds via timeout recovery for this reason.
			delay = time.Minute
		case row.State == "PENDING_RESERVATION":
			cmd.Action = "reserve"
			if row.RefundOf != "" {
				cmd.Action = "refund_reserve"
			}
			a, err := s.account(ctx, cmd)
			if err != nil {
				return true, err
			}
			if a.State == "RESERVED" {
				row.State = "RESERVED"
			} else if a.State == "REJECTED" {
				row.State = "REJECTED"
				row.Reason = a.Reason
			} else {
				return true, errors.New("unexpected reservation state")
			}
		case !s.now().Before(t.Expires):
			row.State = "TIMEOUT"
			if before != "TIMEOUT" {
				slog.WarnContext(ctx, "interbank timeout: funds remain reserved", "interbankTransferId", t.ID)
			}
			resolved, err := s.resolveTimeout(ctx, tx, row)
			if err != nil {
				return true, err
			}
			row.Result = resolved
			delay = retryDelay(row.Attempts)
		default:
			code, b, err := s.Partner.Post(ctx, "/transfers", t.ID, t)
			if err == nil && (code == 202 || code == 200) {
				var ack struct {
					ID     string `json:"interbankTransferId"`
					Status string `json:"status"`
				}
				if json.Unmarshal(b, &ack) != nil || ack.ID != t.ID {
					return true, errors.New("invalid partner acknowledgement")
				}
				if code == 202 && ack.Status != "PROCESSING" {
					return true, errors.New("invalid partner acknowledgement status")
				}
				if ack.Status == "ACCEPTED" || ack.Status == "REJECTED" || ack.Status == "CANCELLED" {
					status := ack.Status
					if status == "CANCELLED" {
						status = "REJECTED"
					}
					row.Result = &Result{ID: t.ID, Destination: s.Config.Partner, Status: status, Processed: time.Now().UTC()}
				} else if ack.Status != "PROCESSING" {
					return true, errors.New("invalid partner transfer status")
				}
				row.State = "PROCESSING"
			} else if err == nil && code == 422 {
				row.Result = &Result{ID: t.ID, Destination: s.Config.Partner, Status: "REJECTED", Reason: "EXPIRED", Processed: time.Now().UTC()}
			}
			// Retry request ambiguity with the SAME id and original validity window.
			delay = retryDelay(row.Attempts)
		}
	}
	var result []byte
	if row.Result != nil {
		result, _ = json.Marshal(row.Result)
	}
	_, e = tx.Exec(ctx, `UPDATE depositos_interbancarios SET estado=$2,motivo=$3,resultado=$4,
 actualizado=CASE WHEN estado<>$2 THEN clock_timestamp() ELSE actualizado END,
 proximo=clock_timestamp()+$5*interval '1 millisecond',intentos=intentos+1,callback_entregado=callback_entregado OR $6 WHERE id=$1`, t.ID, row.State, row.Reason, result, delay.Milliseconds(), delivered)
	if e != nil {
		return true, e
	}
	if before != row.State {
		if e = stateEvent(ctx, tx, t.ID, row.State); e != nil {
			return true, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return true, e
	}
	if before != row.State {
		counter, _ := otel.Meter("bank-usac").Int64Counter("bank_interbank_transitions")
		if counter != nil {
			counter.Add(ctx, 1, metric.WithAttributes(attribute.String("direction", row.Direction), attribute.String("status", row.State)))
		}
		slog.InfoContext(ctx, "interbank state changed", "interbankTransferId", t.ID, "direction", row.Direction, "status", row.State)
	}
	return true, nil
}
func retryDelay(attempt int) time.Duration {
	if attempt > 7 {
		attempt = 7
	}
	return time.Duration(2<<attempt) * time.Second
}
