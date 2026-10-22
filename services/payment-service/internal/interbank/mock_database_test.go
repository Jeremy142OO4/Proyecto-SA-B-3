package interbank

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestMockResultAndCancellationDatabase(t *testing.T) {
	db := testDB(t)
	s := &Service{DB: db}
	ctx := context.Background()
	id := uuid.NewString()
	expires := time.Now().UTC().Add(180 * time.Second)
	body, _ := json.Marshal(map[string]any{"interbankTransferId": id, "amount": "10.00", "expiresAt": expires})
	r := Request{Method: "POST", Path: "/transfers", Key: id, Body: body, Scenario: "reject"}
	if response := s.mock(ctx, r); response.Status != 202 {
		t.Fatal(response)
	}
	if response := s.mock(ctx, r); response.Status != 200 {
		t.Fatal(response)
	}
	_, err := db.Exec(ctx, "UPDATE mock_interbank SET due=clock_timestamp() WHERE id=$1", id)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.StepMock(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err = db.QueryRow(ctx, "SELECT status FROM mock_interbank WHERE id=$1", id).Scan(&state); err != nil || state != "REJECTED" {
		t.Fatal(state, err)
	}
	id = uuid.NewString()
	cancel := Request{Method: "POST", Path: "/transfers/" + id + "/cancel", Scenario: "delayed"}
	if response := s.mock(ctx, cancel); response.Status != 200 {
		t.Fatal(response)
	}
	if response := s.mock(ctx, Request{Method: "GET", Path: "/transfers/" + id, Scenario: "delayed"}); response.Status != 200 {
		t.Fatal(response)
	}
}

func TestRestoreUnavailableMockDatabase(t *testing.T) {
	t.Setenv("ENTORNO", "dev")
	db := testDB(t)
	s := &Service{DB: db}
	ctx := context.Background()
	id := uuid.NewString()
	body, _ := json.Marshal(map[string]any{"interbankTransferId": id, "expiresAt": time.Now().UTC().Add(time.Minute)})
	request := Request{Method: "POST", Path: "/transfers", Key: id, Body: body, Scenario: "unavailable"}
	if r := s.mock(ctx, request); r.Status != 503 {
		t.Fatal(r)
	}
	request.Path = "/transfers/" + id + "/resume"
	if r := s.mock(ctx, request); r.Status != 200 {
		t.Fatal(r)
	}
	request.Path = "/transfers"
	if r := s.mock(ctx, request); r.Status != 200 {
		t.Fatal(r)
	}
	if err := s.StepMock(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := db.QueryRow(ctx, "SELECT status FROM mock_interbank WHERE id=$1", id).Scan(&state); err != nil || state != "ACCEPTED" {
		t.Fatal(state, err)
	}
}
