package core

import "context"

// Broker defines the contract for message broker implementations.
// Each broker plugin must implement this interface.
type Broker interface {
	Publish(ctx context.Context, topic string, msg Message) error
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Close() error
}
