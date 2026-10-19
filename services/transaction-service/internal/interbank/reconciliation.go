package interbank

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Proyecto-SA-B-3/transaction-service/internal/bankrpc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/url"
	"time"
)

type reconciliationItem struct {
	ID       string    `json:"interbankTransferId"`
	Amount   string    `json:"amount"`
	Currency string    `json:"currency"`
	Status   string    `json:"status"`
	Updated  time.Time `json:"updatedAt"`
}
type reconciliationPage struct {
	Date  string               `json:"date"`
	Items []reconciliationItem `json:"items"`
	Page  int                  `json:"page"`
	Size  int                  `json:"pageSize"`
	Total int                  `json:"totalItems"`
}

func (s *Service) reconciliation(ctx context.Context, r Request) bankrpc.Response {
	if r.Peer != s.Config.Partner {
		return failure(401, "UNAUTHORIZED")
	}
	day, e := time.Parse("2006-01-02", r.Date)
	if e != nil {
		return failure(400, "VALIDATION_ERROR")
	}
	if r.Page == 0 {
		r.Page = 1
	}
	if r.PageSize == 0 {
		r.PageSize = 100
	}
	if r.Page < 1 || r.Page > 1000000 || r.PageSize < 1 || r.PageSize > 500 {
		return failure(400, "VALIDATION_ERROR")
	}
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	defer tx.Rollback(ctx)
	page := reconciliationPage{Date: r.Date, Items: []reconciliationItem{}, Page: r.Page, Size: r.PageSize}
	e = tx.QueryRow(ctx, `SELECT count(*) FROM depositos_interbancarios WHERE direccion='IN' AND solicitud->>'originBankCode'=$1 AND recibido>=$2 AND recibido<$3`, r.Peer, day, day.Add(24*time.Hour)).Scan(&page.Total)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	rows, e := tx.Query(ctx, `SELECT id::text,solicitud->>'amount',solicitud->>'currency',estado,actualizado FROM depositos_interbancarios WHERE direccion='IN' AND solicitud->>'originBankCode'=$1 AND recibido>=$2 AND recibido<$3 ORDER BY id LIMIT $4 OFFSET $5`, r.Peer, day, day.Add(24*time.Hour), r.PageSize, (r.Page-1)*r.PageSize)
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	for rows.Next() {
		var item reconciliationItem
		if e = rows.Scan(&item.ID, &item.Amount, &item.Currency, &item.Status, &item.Updated); e != nil {
			rows.Close()
			return failure(503, "SERVICE_UNAVAILABLE")
		}
		item.Updated = item.Updated.UTC()
		page.Items = append(page.Items, item)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	if e = tx.Commit(ctx); e != nil {
		return failure(503, "SERVICE_UNAVAILABLE")
	}
	return bankrpc.Reply(200, page)
}

// Daily reconciliation is a durable, paginated job. A unique date and SKIP
// LOCKED prevent duplicate runs across replicas. Nothing releases or credits
// money merely because an item is absent from a listing.
func (s *Service) StepReconciliation(parent context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	now := s.now()
	today := now.Truncate(24 * time.Hour)
	cutoff := today.Add(-24 * time.Hour)
	if now.Sub(today) < 10*time.Minute {
		cutoff = cutoff.Add(-24 * time.Hour)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	var start string
	e = tx.QueryRow(ctx, `SELECT COALESCE((MAX(fecha)+1)::text,(SELECT LEAST(COALESCE((MIN(recibido AT TIME ZONE 'UTC'))::date,$1::date),$1::date)::text FROM depositos_interbancarios)) FROM conciliaciones_interbancarias`, cutoff.Format("2006-01-02")).Scan(&start)
	if e != nil {
		return false, e
	}
	first, e := time.Parse("2006-01-02", start)
	if e != nil {
		return false, e
	}
	for i := 0; i < 31 && !first.After(cutoff); i++ {
		_, e = tx.Exec(ctx, `INSERT INTO conciliaciones_interbancarias(fecha) VALUES($1) ON CONFLICT DO NOTHING`, first.Format("2006-01-02"))
		if e != nil {
			return false, e
		}
		first = first.Add(24 * time.Hour)
	}
	var day string
	var page int
	e = tx.QueryRow(ctx, `SELECT fecha::text,pagina FROM conciliaciones_interbancarias WHERE estado='PENDING' AND proximo<=clock_timestamp() ORDER BY fecha FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&day, &page)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, tx.Commit(ctx)
	}
	if e != nil {
		return false, e
	}
	retry := func(cause error) (bool, error) {
		_, err := tx.Exec(ctx, `UPDATE conciliaciones_interbancarias SET proximo=clock_timestamp()+interval '30 seconds' WHERE fecha=$1`, day)
		if err != nil {
			return true, err
		}
		if err = tx.Commit(ctx); err != nil {
			return true, err
		}
		return true, cause
	}
	code, body, err := s.get(ctx, fmt.Sprintf("/reconciliation?date=%s&page=%d&pageSize=100", url.QueryEscape(day), page))
	if err != nil || code != 200 {
		return retry(errors.New("partner reconciliation unavailable"))
	}
	var remote reconciliationPage
	if json.Unmarshal(body, &remote) != nil || remote.Date != day || remote.Page != page || remote.Size != 100 || remote.Total < 0 || len(remote.Items) > 100 || page > 10000 || (len(remote.Items) == 0 && remote.Total > (page-1)*100) {
		return retry(errors.New("invalid reconciliation page"))
	}
	expected := remote.Total - (page-1)*100
	if expected < 0 {
		expected = 0
	}
	if expected > 100 {
		expected = 100
	}
	if len(remote.Items) != expected {
		return retry(errors.New("incomplete reconciliation page"))
	}
	for _, item := range remote.Items {
		id, err := uuid.Parse(item.ID)
		n, amountErr := Cents(item.Amount)
		if err != nil || id == uuid.Nil || amountErr != nil || n < 1 || n > 1000000 || item.Currency != "GTQ" || item.Updated.IsZero() {
			return retry(errors.New("invalid reconciliation item"))
		}
		switch item.Status {
		case "PROCESSING", "ACCEPTED", "REJECTED", "CANCELLED":
		default:
			return retry(errors.New("invalid reconciliation state"))
		}
		raw, _ := json.Marshal(item)
		var previous []byte
		err = tx.QueryRow(ctx, `SELECT contenido FROM conciliacion_items_interbancarios WHERE fecha=$1 AND id=$2`, day, item.ID).Scan(&previous)
		if err == nil {
			var old reconciliationItem
			if json.Unmarshal(previous, &old) != nil || old != item {
				if e = incident(ctx, tx, item.ID, "RECONCILIATION_DUPLICATE_ID"); e != nil {
					return true, e
				}
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return true, err
		}
		_, e = tx.Exec(ctx, `INSERT INTO conciliacion_items_interbancarios(fecha,id,contenido) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, day, item.ID, raw)
		if e != nil {
			return true, e
		}
		local, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM depositos_interbancarios WHERE id=$1 AND direccion='OUT' FOR UPDATE`, item.ID))
		if errors.Is(err, pgx.ErrNoRows) {
			if e = incident(ctx, tx, item.ID, "RECONCILIATION_UNKNOWN_TRANSFER"); e != nil {
				return true, e
			}
			continue
		}
		if err != nil {
			return true, err
		}
		if local.Transfer.Amount != item.Amount || local.Transfer.Currency != item.Currency {
			if e = incident(ctx, tx, item.ID, "RECONCILIATION_AMOUNT_MISMATCH"); e != nil {
				return true, e
			}
			continue
		}
		if item.Status == "ACCEPTED" && local.State == "REVERSED" {
			if e = s.needReversal(ctx, tx, local); e != nil {
				return true, e
			}
			continue
		}
		if (local.State == "RESERVED" || local.State == "PROCESSING" || local.State == "TIMEOUT") && item.Status != "PROCESSING" {
			result := Result{ID: item.ID, Destination: s.Config.Partner, Status: "REJECTED", Processed: item.Updated}
			if item.Status == "ACCEPTED" {
				result.Status = "ACCEPTED"
			}
			raw, _ := json.Marshal(result)
			// A newly learned ACCEPTED may supersede a pending release, but an older
			// rejection must never overwrite an already persisted ACCEPTED callback.
			if local.Result == nil || result.Status == "ACCEPTED" {
				_, e = tx.Exec(ctx, `UPDATE depositos_interbancarios SET resultado=$2,proximo=clock_timestamp() WHERE id=$1`, item.ID, raw)
				if e != nil {
					return true, e
				}
			}
		} else if local.State == "ACCEPTED" && item.Status != "ACCEPTED" {
			if e = incident(ctx, tx, item.ID, "RECONCILIATION_STATE_MISMATCH"); e != nil {
				return true, e
			}
		}
	}
	complete := page*100 >= remote.Total
	if complete {
		date, _ := time.Parse("2006-01-02", day)
		rows, e := tx.Query(ctx, `SELECT id::text,(solicitud->>'createdAt')::timestamptz FROM depositos_interbancarios d WHERE direccion='OUT' AND estado='ACCEPTED' AND (solicitud->>'createdAt')::timestamptz>=$1 AND (solicitud->>'createdAt')::timestamptz<$2 AND NOT EXISTS(SELECT 1 FROM conciliacion_items_interbancarios i WHERE i.fecha=$3 AND i.id=d.id)`, date, date.Add(24*time.Hour), day)
		if e != nil {
			return true, e
		}
		var missing []struct {
			id      string
			created time.Time
		}
		for rows.Next() {
			var item struct {
				id      string
				created time.Time
			}
			if e = rows.Scan(&item.id, &item.created); e != nil {
				rows.Close()
				return true, e
			}
			missing = append(missing, item)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return true, e
		}
		for _, item := range missing {
			// Received and created can straddle UTC midnight. Check boundary
			// ids directly instead of silently excluding them or reporting a
			// false missing entry from the adjacent day's listing.
			if item.created.Before(date.Add(5*time.Minute)) || !item.created.Before(date.Add(24*time.Hour-5*time.Minute)) {
				code, body, err := s.get(ctx, "/transfers/"+item.id)
				if err != nil || code >= 500 {
					return retry(errors.New("UTC boundary status unavailable"))
				}
				var status struct {
					ID     string `json:"interbankTransferId"`
					Status string `json:"status"`
				}
				if code == 200 && json.Unmarshal(body, &status) == nil && status.ID == item.id && status.Status == "ACCEPTED" {
					continue
				}
			}
			if e = incident(ctx, tx, item.id, "RECONCILIATION_MISSING_TRANSFER"); e != nil {
				return true, e
			}
		}
	}
	state := "PENDING"
	if complete {
		state = "COMPLETED"
	}
	_, e = tx.Exec(ctx, `UPDATE conciliaciones_interbancarias SET estado=$2,pagina=pagina+1,actualizado=clock_timestamp(),
 discrepancias=(SELECT count(*) FROM incidentes_interbancarios i WHERE tipo LIKE 'RECONCILIATION_%' AND (
 EXISTS(SELECT 1 FROM conciliacion_items_interbancarios c WHERE c.fecha=$1::date AND c.id=i.transferencia)
 OR EXISTS(SELECT 1 FROM depositos_interbancarios d WHERE d.id=i.transferencia AND direccion='OUT' AND ((solicitud->>'createdAt')::timestamptz AT TIME ZONE 'UTC')::date=$1::date))) WHERE fecha=$1`, day, state)
	if e != nil {
		return true, e
	}
	return true, tx.Commit(ctx)
}
