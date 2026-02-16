package core_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miladsoleymani/eventmux/core"
	"github.com/miladsoleymani/eventmux/internal/mock"
)

func TestRouter_HandleAndStart(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	var called atomic.Bool
	r.Handle("orders.created", func(ctx context.Context, msg core.Message) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Start in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Start(ctx)
	}()

	// Give Start time to subscribe
	time.Sleep(50 * time.Millisecond)

	// Deliver a message
	msg := &mock.Message{K: []byte("key1"), V: []byte("value1")}
	if err := mb.Deliver(ctx, "orders.created", msg); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	if !called.Load() {
		t.Error("handler was not called")
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if !mb.IsClosed() {
		t.Error("broker should be closed after Start returns")
	}
}

func TestRouter_Middleware(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	var order []string

	mw := func(name string) core.Middleware {
		return func(next core.Handler) core.Handler {
			return func(ctx context.Context, msg core.Message) error {
				order = append(order, name+":before")
				err := next(ctx, msg)
				order = append(order, name+":after")
				return err
			}
		}
	}

	r.Use(mw("A"))
	r.Use(mw("B"))

	r.Handle("test.topic", func(ctx context.Context, msg core.Message) error {
		order = append(order, "handler")
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	if err := mb.Deliver(ctx, "test.topic", msg); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	// Middleware applied in reverse iteration order: last registered (B) wraps innermost.
	// applyMiddleware loops from end to start, so A wraps B wraps handler.
	// Call order: A:before -> B:before -> handler -> B:after -> A:after
	expected := []string{"A:before", "B:before", "handler", "B:after", "A:after"}
	if len(order) != len(expected) {
		t.Fatalf("got %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestRouter_Publish(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	if err := r.Publish(context.Background(), "out.topic", msg); err != nil {
		t.Fatalf("publish: %v", err)
	}

	pubs := mb.Published()
	if len(pubs) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pubs))
	}
	if pubs[0].Topic != "out.topic" {
		t.Errorf("published to %q, want %q", pubs[0].Topic, "out.topic")
	}
}

func TestRouter_NilBroker(t *testing.T) {
	r := core.New(nil)
	err := r.Start(context.Background())
	if err != core.ErrNoBroker {
		t.Errorf("expected ErrNoBroker, got %v", err)
	}
}

func TestRouter_DoubleStart(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	err := r.Start(ctx)
	if err != core.ErrAlreadyStarted {
		t.Errorf("expected ErrAlreadyStarted, got %v", err)
	}
}
