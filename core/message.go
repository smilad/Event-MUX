package core

import "context"

// Message is the broker-agnostic message abstraction.
// Implementations are provided by broker plugins.
type Message interface {
	Key() []byte
	Value() []byte
	Headers() map[string]string
	Ack() error
	Nack() error
}

// Handler is the low-level handler used by broker subscriptions.
// Users should prefer HandlerFunc which receives a Context.
type Handler func(ctx context.Context, msg Message) error

// Middleware is the low-level middleware used internally.
// Users should prefer MiddlewareFunc which receives a Context.
type Middleware func(Handler) Handler
