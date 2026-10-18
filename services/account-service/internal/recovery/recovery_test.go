package recovery

import (
	"context"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestServeClosesTransportAndJoinsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	blocked := make(chan struct{})
	started := make(chan struct{})
	var joined atomic.Bool
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, func() { close(blocked) }, nil, func(context.Context) error {
			close(started)
			<-blocked
			joined.Store(true)
			return nil
		})
	}()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked")
	}
	if !joined.Load() {
		t.Fatal("worker not joined")
	}
}

func TestRunRetriesStartupAndStopsDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		run(ctx, func(context.Context) error {
			if attempts.Add(1) == 3 {
				cancel()
			}
			return errors.New("offline")
		}, time.Millisecond, 2*time.Millisecond)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("retry loop blocked")
	}
	if attempts.Load() != 3 {
		t.Fatal(attempts.Load())
	}
}

func TestServeRestartsWhenConsumerStopsWithoutError(t *testing.T) {
	var closed atomic.Bool
	if err := Serve(context.Background(), func() { closed.Store(true) }, nil, func(context.Context) error { return nil }); err == nil {
		t.Fatal("closed deliveries must invalidate session")
	}
	if !closed.Load() {
		t.Fatal("transport not closed")
	}
}

func TestAMQPChannelAndConnectionRecovery(t *testing.T) {
	url := os.Getenv("TEST_RABBITMQ_URL")
	if url == "" {
		t.Skip("requires isolated TEST_RABBITMQ_URL")
	}
	type session struct {
		conn     *amqp.Connection
		pub      *amqp.Channel
		delivery <-chan amqp.Delivery
	}
	opened := make(chan session, 1)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer close(done)
		run(ctx, func(ctx context.Context) error {
			conn, err := amqp.DialConfig(url, amqp.Config{Heartbeat: time.Second, Dial: amqp.DefaultDial(3 * time.Second)})
			if err != nil {
				return err
			}
			defer conn.Close()
			pub, err := conn.Channel()
			if err != nil {
				return err
			}
			consumer, err := conn.Channel()
			if err != nil {
				return err
			}
			q, err := consumer.QueueDeclare("", false, true, true, false, nil)
			if err != nil {
				return err
			}
			deliveries, err := consumer.ConsumeWithContext(ctx, q.Name, "", false, false, false, false, nil)
			if err != nil {
				return err
			}
			if err = pub.PublishWithContext(ctx, "", q.Name, false, false, amqp.Publishing{Body: []byte("probe")}); err != nil {
				return err
			}
			select {
			case opened <- session{conn, pub, deliveries}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return Serve(ctx, func() { _ = conn.CloseDeadline(time.Now().Add(time.Second)) },
				[]func() bool{conn.IsClosed, pub.IsClosed, consumer.IsClosed}, func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
		}, time.Millisecond, 10*time.Millisecond)
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("supervisor leaked")
		}
	}()
	for i := 0; i < 3; i++ {
		var current session
		select {
		case current = <-opened:
		case <-time.After(15 * time.Second):
			t.Fatal("session did not recover", i)
		}
		select {
		case d := <-current.delivery:
			if string(d.Body) != "probe" {
				t.Fatal("delivery lost")
			}
			_ = d.Ack(false)
		case <-time.After(3 * time.Second):
			t.Fatal("rebuilt consumer not working")
		}
		if i == 0 {
			_ = current.pub.Close()
		} else if i == 1 {
			_ = current.conn.Close()
		}
	}
}
