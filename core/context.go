package core

import (
	"context"
	"fmt"
	"sync"
)

// Context is the handler context for EventMux, inspired by echo.Context.
// It wraps the incoming message, provides deserialization via Bind,
// and exposes response methods (Ack, Nack, Republish).
type Context interface {
	// Context returns the underlying context.Context.
	Context() context.Context

	// SetContext replaces the underlying context.Context.
	// Useful for middleware that enriches the context with values or deadlines.
	SetContext(ctx context.Context)

	// Message returns the raw underlying Message.
	Message() Message

	// Topic returns the topic this message was received on.
	Topic() string

	// Key returns the message key.
	Key() []byte

	// Value returns the raw message body.
	Value() []byte

	// Header returns a single header value by key.
	Header(key string) string

	// Headers returns all message headers.
	Headers() map[string]string

	// Bind deserializes the message body into the given struct
	// using the router's configured Binder.
	Bind(v any) error

	// Ack acknowledges the message (commits offset / removes from queue).
	Ack() error

	// Nack negatively acknowledges the message (triggers redelivery).
	Nack() error

	// Republish sends the current message to a different topic.
	// Useful for dead-letter routing, fan-out, or saga patterns.
	Republish(topic string) error

	// Set stores a key-value pair in the context store.
	// Used by middleware to pass data to downstream handlers.
	Set(key string, val any)

	// Get retrieves a value from the context store.
	Get(key string) (any, bool)
}

// HandlerFunc is the function signature for EventMux handlers.
// Handlers receive a Context and return an error.
//
//	r.Handle("orders.created", func(c eventmux.Context) error {
//	    var order Order
//	    if err := c.Bind(&order); err != nil {
//	        return err
//	    }
//	    // process order...
//	    return c.Ack()
//	})
type HandlerFunc func(c Context) error

// MiddlewareFunc wraps a HandlerFunc to add cross-cutting behavior.
//
//	func MyMiddleware() eventmux.MiddlewareFunc {
//	    return func(next eventmux.HandlerFunc) eventmux.HandlerFunc {
//	        return func(c eventmux.Context) error {
//	            // before
//	            err := next(c)
//	            // after
//	            return err
//	        }
//	    }
//	}
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// ---------------------------------------------------------------------------
// Default implementation
// ---------------------------------------------------------------------------

type eventContext struct {
	ctx    context.Context
	msg    Message
	topic  string
	broker Broker
	binder Binder
	store  map[string]any
	mu     sync.RWMutex
}

// NewContext creates a Context for the given message.
// This is called internally by the Router for each incoming message.
func NewContext(ctx context.Context, msg Message, topic string, b Broker, binder Binder) Context {
	return &eventContext{
		ctx:    ctx,
		msg:    msg,
		topic:  topic,
		broker: b,
		binder: binder,
		store:  make(map[string]any),
	}
}

func (c *eventContext) Context() context.Context { return c.ctx }

func (c *eventContext) SetContext(ctx context.Context) { c.ctx = ctx }

func (c *eventContext) Message() Message { return c.msg }

func (c *eventContext) Topic() string { return c.topic }

func (c *eventContext) Key() []byte { return c.msg.Key() }

func (c *eventContext) Value() []byte { return c.msg.Value() }

func (c *eventContext) Header(key string) string {
	return c.msg.Headers()[key]
}

func (c *eventContext) Headers() map[string]string {
	return c.msg.Headers()
}

func (c *eventContext) Bind(v any) error {
	if c.binder == nil {
		return fmt.Errorf("eventmux: no binder configured")
	}
	if err := c.binder.Bind(c.msg.Value(), v); err != nil {
		return fmt.Errorf("eventmux: bind: %w", err)
	}
	return nil
}

func (c *eventContext) Ack() error {
	if err := c.msg.Ack(); err != nil {
		return fmt.Errorf("eventmux: ack: %w", err)
	}
	return nil
}

func (c *eventContext) Nack() error {
	if err := c.msg.Nack(); err != nil {
		return fmt.Errorf("eventmux: nack: %w", err)
	}
	return nil
}

func (c *eventContext) Republish(topic string) error {
	if c.broker == nil {
		return ErrNoBroker
	}
	if err := c.broker.Publish(c.ctx, topic, c.msg); err != nil {
		return fmt.Errorf("eventmux: republish to %q: %w", topic, err)
	}
	return nil
}

func (c *eventContext) Set(key string, val any) {
	c.mu.Lock()
	c.store[key] = val
	c.mu.Unlock()
}

func (c *eventContext) Get(key string) (any, bool) {
	c.mu.RLock()
	val, ok := c.store[key]
	c.mu.RUnlock()
	return val, ok
}
