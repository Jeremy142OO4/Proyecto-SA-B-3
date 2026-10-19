package bankrpc

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRabbitRPC(t *testing.T) {
	url := os.Getenv("TEST_AMQP_URL")
	if url == "" {
		t.Skip("TEST_AMQP_URL required: isolated broker only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue := "test.interbank." + uuid.NewString()
	conn, e := amqp.Dial(url)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	ch, e := conn.Channel()
	if e != nil {
		t.Fatal(e)
	}
	defer ch.Close()
	defer ch.QueueDelete(queue, false, false, false)
	seen := map[string]int{}
	var mu sync.Mutex
	handler := func(ctx context.Context, body json.RawMessage) Response {
		var r struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(body, &r) != nil {
			return Reply(400, nil)
		}
		mu.Lock()
		defer mu.Unlock()
		if seen[r.ID] == 0 {
			seen[r.ID]++
		}
		return Reply(200, map[string]any{"id": r.ID, "effects": seen[r.ID]})
	}
	serverctx, stop := context.WithCancel(ctx)
	defer stop()
	go Run(serverctx, url, queue, handler)
	id := uuid.NewString()
	ready := false
	for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
		r, e := Call(ctx, url, queue, map[string]string{"id": id})
		if e == nil && r.Status == 200 {
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatal("consumer not ready")
	}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := Call(ctx, url, queue, map[string]string{"id": id})
			if e != nil || r.Status != 200 {
				t.Errorf("RPC: %+v %v", r, e)
			}
		}()
	}
	wg.Wait()
	mu.Lock()
	effects := seen[id]
	mu.Unlock()
	if effects != 1 {
		t.Fatalf("effects=%d", effects)
	}
	stop()
	time.Sleep(100 * time.Millisecond)
	go Run(ctx, url, queue, handler)
	r, e := Call(ctx, url, queue, map[string]string{"id": id})
	if e != nil || r.Status != 200 {
		t.Fatalf("consumer restart: %+v %v", r, e)
	}
	if _, e = Call(ctx, url, "missing."+uuid.NewString(), nil); e == nil {
		t.Fatal("unroutable request succeeded")
	}
}
