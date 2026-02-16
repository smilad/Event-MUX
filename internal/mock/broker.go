package mock

import (
	"context"
	"sync"

	"github.com/miladsoleymani/eventmux/core"
)

// Broker is a test double for core.Broker.
type Broker struct {
	mu           sync.Mutex
	published    []PublishedMessage
	handlers     map[string]core.Handler
	SubscribeErr error
	PublishErr   error
	closed       bool
}

// PublishedMessage records a message sent through Publish.
type PublishedMessage struct {
	Topic   string
	Message core.Message
}

func NewBroker() *Broker {
	return &Broker{
		handlers: make(map[string]core.Handler),
	}
}

func (b *Broker) Publish(_ context.Context, topic string, msg core.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.PublishErr != nil {
		return b.PublishErr
	}
	b.published = append(b.published, PublishedMessage{Topic: topic, Message: msg})
	return nil
}

func (b *Broker) Subscribe(ctx context.Context, topic string, handler core.Handler) error {
	b.mu.Lock()
	if b.SubscribeErr != nil {
		err := b.SubscribeErr
		b.mu.Unlock()
		return err
	}
	b.handlers[topic] = handler
	b.mu.Unlock()

	// Block until context is cancelled (simulates a real subscription loop)
	<-ctx.Done()
	return nil
}

func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

// Deliver simulates an incoming message to a registered handler.
// It invokes the low-level Handler (which the Router bridges to Context internally).
func (b *Broker) Deliver(ctx context.Context, topic string, msg core.Message) error {
	b.mu.Lock()
	h, ok := b.handlers[topic]
	b.mu.Unlock()
	if !ok {
		return core.ErrNoHandler
	}
	return h(ctx, msg)
}

// Published returns all messages sent via Publish.
func (b *Broker) Published() []PublishedMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]PublishedMessage, len(b.published))
	copy(out, b.published)
	return out
}

// IsClosed reports whether Close was called.
func (b *Broker) IsClosed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}
