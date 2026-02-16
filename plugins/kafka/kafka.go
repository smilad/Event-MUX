package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/miladsoleymani/eventmux/broker"
	"github.com/miladsoleymani/eventmux/core"
)

func init() {
	broker.Register("kafka", func(cfg broker.Config) (core.Broker, error) {
		opts := optsFromConfig(cfg)
		return New(cfg.Brokers, cfg.Group, opts...)
	})
}

// Broker implements core.Broker for Apache Kafka using segmentio/kafka-go.
//
// Design decisions:
//   - One kafka.Writer shared across all Publish calls (thread-safe by library).
//   - One kafka.Reader per Subscribe call, each running in its own goroutine.
//   - Manual offset commit via Ack(); not committing (Nack) causes redelivery.
//   - Graceful shutdown: context cancellation breaks the fetch loop, Close()
//     flushes the writer and closes all readers.
type Broker struct {
	brokers []string
	group   string
	opts    options

	writer  *kafka.Writer
	readers []*kafka.Reader
	mu      sync.Mutex
	closed  bool
}

// New creates a Kafka Broker.
func New(brokers []string, group string, fns ...Option) (*Broker, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("eventmux/kafka: at least one broker address is required")
	}

	opts := defaults()
	for _, fn := range fns {
		fn(&opts)
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     opts.balancer,
		BatchSize:    opts.batchSize,
		Async:        opts.async,
		RequiredAcks: kafka.RequireAll,
	}
	if opts.dialer != nil {
		w.Transport = &kafka.Transport{
			TLS:  opts.dialer.TLS,
			SASL: opts.dialer.SASLMechanism,
		}
	}

	return &Broker{
		brokers: brokers,
		group:   group,
		opts:    opts,
		writer:  w,
	}, nil
}

// Publish sends a message to the specified topic.
func (b *Broker) Publish(ctx context.Context, topic string, msg core.Message) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return core.ErrBrokerClosed
	}
	b.mu.Unlock()

	km := kafka.Message{
		Topic:   topic,
		Key:     msg.Key(),
		Value:   msg.Value(),
		Headers: toHeaders(msg.Headers()),
	}
	if err := b.writer.WriteMessages(ctx, km); err != nil {
		return fmt.Errorf("eventmux/kafka: publish to %q: %w", topic, err)
	}
	return nil
}

// Subscribe creates a consumer for the topic and blocks, delivering messages
// to the handler until the context is cancelled.
func (b *Broker) Subscribe(ctx context.Context, topic string, handler core.Handler) error {
	cfg := kafka.ReaderConfig{
		Brokers:  b.brokers,
		Topic:    topic,
		GroupID:  b.group,
		MinBytes: b.opts.minBytes,
		MaxBytes: b.opts.maxBytes,
		MaxWait:  b.opts.maxWait,
	}
	if b.opts.dialer != nil {
		cfg.Dialer = b.opts.dialer
	}
	if b.group == "" {
		cfg.StartOffset = b.opts.startOffset
	}

	r := kafka.NewReader(cfg)

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		r.Close()
		return core.ErrBrokerClosed
	}
	b.readers = append(b.readers, r)
	b.mu.Unlock()

	return b.consumeLoop(ctx, r, handler)
}

// consumeLoop fetches messages and dispatches them to the handler.
func (b *Broker) consumeLoop(ctx context.Context, r *kafka.Reader, handler core.Handler) error {
	for {
		raw, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // graceful shutdown
			}
			return fmt.Errorf("eventmux/kafka: fetch: %w", err)
		}

		msg := &message{raw: raw, reader: r, ctx: ctx}
		if err := handler(ctx, msg); err != nil {
			// Handler returned an error — offset is NOT committed.
			// The message will be redelivered after rebalance or restart.
			continue
		}
	}
}

// Close flushes the writer and closes all readers.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true

	var errs []error
	if err := b.writer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("eventmux/kafka: close writer: %w", err))
	}
	for _, r := range b.readers {
		if err := r.Close(); err != nil {
			errs = append(errs, fmt.Errorf("eventmux/kafka: close reader: %w", err))
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// toHeaders converts a string map to Kafka headers.
func toHeaders(h map[string]string) []kafka.Header {
	if len(h) == 0 {
		return nil
	}
	headers := make([]kafka.Header, 0, len(h))
	for k, v := range h {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}
	return headers
}

// optsFromConfig extracts options from the broker.Config.Extra map.
func optsFromConfig(cfg broker.Config) []Option {
	if cfg.Extra == nil {
		return nil
	}
	var opts []Option
	if v, ok := cfg.Extra["async"].(bool); ok && v {
		opts = append(opts, WithAsync(true))
	}
	if v, ok := cfg.Extra["batch_size"].(int); ok {
		opts = append(opts, WithBatchSize(v))
	}
	if v, ok := cfg.Extra["max_bytes"].(int); ok {
		opts = append(opts, WithMaxBytes(v))
	}
	return opts
}
