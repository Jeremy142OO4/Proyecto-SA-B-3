// Package recovery rebuilds an entire AMQP session after a transport failure.
// Business messages are never retried here; durable Outboxes own that policy.
package recovery

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

func Run(ctx context.Context, cycle func(context.Context) error) {
	run(ctx, cycle, time.Second, 15*time.Second)
}

func run(ctx context.Context, cycle func(context.Context) error, initial, maximum time.Duration) {
	delay := initial
	for ctx.Err() == nil {
		started := time.Now()
		_ = cycle(ctx)
		if ctx.Err() != nil {
			return
		}
		slog.Warn("RabbitMQ session unavailable; rebuilding consumers and publishers", "retry_in", delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if time.Since(started) > 30*time.Second {
			delay = initial
		} else {
			delay *= 2
			if delay > maximum {
				delay = maximum
			}
		}
	}
}

// Serve cancels and joins all workers before the next session can be opened.
// Closing the transport also releases blocked publisher confirms and deliveries.
func Serve(ctx context.Context, closeTransport func(), closed []func() bool, workers ...func(context.Context) error) error {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	finished := make(chan error, len(workers))
	var group sync.WaitGroup
	for _, worker := range workers {
		group.Add(1)
		go func(work func(context.Context) error) {
			defer group.Done()
			finished <- work(child)
		}(worker)
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var result error
	waiting := true
	for waiting {
		select {
		case <-ctx.Done():
			result = ctx.Err()
			waiting = false
		case result = <-finished:
			if result == nil {
				result = errors.New("AMQP worker stopped")
			}
			waiting = false
		case <-ticker.C:
			for _, isClosed := range closed {
				if isClosed() {
					result = errors.New("AMQP transport or channel closed")
					waiting = false
					break
				}
			}
		}
	}
	cancel()
	closeTransport()
	group.Wait()
	return result
}
