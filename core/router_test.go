package core_test

import (
	"context"
	"encoding/json"
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
	r.Handle("orders.created", func(c core.Context) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

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

func TestRouter_ContextAccess(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	var gotTopic string
	var gotKey string
	var gotHeader string

	r.Handle("orders.created", func(c core.Context) error {
		gotTopic = c.Topic()
		gotKey = string(c.Key())
		gotHeader = c.Header("trace-id")
		return c.Ack()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	msg := &mock.Message{
		K: []byte("order-123"),
		V: []byte(`{"id":123}`),
		H: map[string]string{"trace-id": "abc-def"},
	}
	if err := mb.Deliver(ctx, "orders.created", msg); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	if gotTopic != "orders.created" {
		t.Errorf("topic = %q, want %q", gotTopic, "orders.created")
	}
	if gotKey != "order-123" {
		t.Errorf("key = %q, want %q", gotKey, "order-123")
	}
	if gotHeader != "abc-def" {
		t.Errorf("header = %q, want %q", gotHeader, "abc-def")
	}
	if !msg.Acked {
		t.Error("message should have been acked")
	}
}

func TestRouter_Bind(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	type Order struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var got Order
	r.Handle("orders.created", func(c core.Context) error {
		if err := c.Bind(&got); err != nil {
			return err
		}
		return c.Ack()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	payload, _ := json.Marshal(Order{ID: 42, Name: "test"})
	msg := &mock.Message{K: []byte("k"), V: payload}
	if err := mb.Deliver(ctx, "orders.created", msg); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	if got.ID != 42 || got.Name != "test" {
		t.Errorf("bind got %+v, want {ID:42 Name:test}", got)
	}
}

func TestRouter_Nack(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	r.Handle("orders.failed", func(c core.Context) error {
		return c.Nack()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	if err := mb.Deliver(ctx, "orders.failed", msg); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	if !msg.Nacked {
		t.Error("message should have been nacked")
	}
}

func TestRouter_Republish(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	r.Handle("orders.created", func(c core.Context) error {
		if err := c.Republish("orders.dlq"); err != nil {
			return err
		}
		return c.Ack()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	if err := mb.Deliver(ctx, "orders.created", msg); err != nil {
		t.Fatalf("deliver: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	pubs := mb.Published()
	if len(pubs) != 1 {
		t.Fatalf("expected 1 republished message, got %d", len(pubs))
	}
	if pubs[0].Topic != "orders.dlq" {
		t.Errorf("republished to %q, want %q", pubs[0].Topic, "orders.dlq")
	}
}

func TestRouter_ContextStore(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	// Middleware sets a value
	r.Use(func(next core.HandlerFunc) core.HandlerFunc {
		return func(c core.Context) error {
			c.Set("user-id", "usr-42")
			return next(c)
		}
	})

	var gotUserID string
	r.Handle("test.topic", func(c core.Context) error {
		if v, ok := c.Get("user-id"); ok {
			gotUserID = v.(string)
		}
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

	if gotUserID != "usr-42" {
		t.Errorf("user-id = %q, want %q", gotUserID, "usr-42")
	}
}

func TestRouter_Middleware(t *testing.T) {
	mb := mock.NewBroker()
	r := core.New(mb)

	var order []string

	mw := func(name string) core.MiddlewareFunc {
		return func(next core.HandlerFunc) core.HandlerFunc {
			return func(c core.Context) error {
				order = append(order, name+":before")
				err := next(c)
				order = append(order, name+":after")
				return err
			}
		}
	}

	r.Use(mw("A"))
	r.Use(mw("B"))

	r.Handle("test.topic", func(c core.Context) error {
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
