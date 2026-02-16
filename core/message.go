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

// Handler processes a message within a context.
type Handler func(ctx context.Context, msg Message) error

// Middleware wraps a Handler to add cross-cutting behavior.
type Middleware func(Handler) Handler
