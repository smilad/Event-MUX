package nats

import (
	"context"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/miladsoleymani/eventmux/broker"
	"github.com/miladsoleymani/eventmux/core"
)

func init() {
	broker.Register("nats", func(cfg broker.Config) (core.Broker, error) {
		opts := optsFromConfig(cfg)
		if len(cfg.Brokers) == 0 {
			return nil, fmt.Errorf("eventmux/nats: at least one broker URL is required")
		}
		return New(cfg.Brokers[0], cfg.Group, opts...)
	})
}

// Broker implements core.Broker for NATS JetStream.
//
// Design decisions:
//   - One NATS connection per Broker instance.
//   - JetStream is used for persistence and at-least-once delivery.
//   - Each Subscribe call creates (or updates) a stream and a durable consumer.
//   - Manual ack via Ack(); Nack() triggers server-side redelivery.
//   - Configurable stream retention, storage type, and consumer ack policy.
//   - Graceful shutdown: context cancellation stops consumers, Close() drains
//     the connection.
type Broker struct {
	conn  *nats.Conn
	js    jetstream.JetStream
	group string
	opts  options

	mu     sync.Mutex
	closed bool
	subs   []jetstream.ConsumeContext
}

// New creates a NATS JetStream Broker. url is a standard NATS URL (nats://host:port).
func New(url, group string, fns ...Option) (*Broker, error) {
	opts := defaults()
	for _, fn := range fns {
		fn(&opts)
	}

	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("eventmux/nats: connect to %q: %w", url, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("eventmux/nats: init jetstream: %w", err)
	}

	return &Broker{
		conn:  nc,
		js:    js,
		group: group,
		opts:  opts,
	}, nil
}

// Publish sends a message to the specified subject via JetStream.
func (b *Broker) Publish(ctx context.Context, topic string, msg core.Message) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return core.ErrBrokerClosed
	}
	b.mu.Unlock()

	headers := nats.Header{}
	for k, v := range msg.Headers() {
		headers.Set(k, v)
	}

	nm := &nats.Msg{
		Subject: topic,
		Data:    msg.Value(),
		Header:  headers,
	}
	if _, err := b.js.PublishMsg(ctx, nm); err != nil {
		return fmt.Errorf("eventmux/nats: publish to %q: %w", topic, err)
	}
	return nil
}

// Subscribe creates or updates a JetStream stream and durable consumer
// for the given subject, then consumes messages until the context is cancelled.
func (b *Broker) Subscribe(ctx context.Context, topic string, handler core.Handler) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return core.ErrBrokerClosed
	}
	b.mu.Unlock()

	streamName := sanitizeStreamName(topic)
	stream, err := b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{topic},
		MaxMsgs:   b.opts.maxMsgs,
		MaxBytes:  b.opts.maxBytes,
		MaxAge:    b.opts.maxAge,
		Replicas:  b.opts.replicas,
		Retention: b.opts.retention,
		Storage:   b.opts.storage,
	})
	if err != nil {
		return fmt.Errorf("eventmux/nats: create stream %q: %w", streamName, err)
	}

	consumerName := b.group
	if consumerName == "" {
		consumerName = "eventmux-" + streamName
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:    consumerName,
		AckPolicy:  jetstream.AckExplicitPolicy,
		AckWait:    b.opts.ackWait,
		MaxDeliver: b.opts.maxDeliver,
	})
	if err != nil {
		return fmt.Errorf("eventmux/nats: create consumer %q: %w", consumerName, err)
	}

	cc, err := cons.Consume(func(jsMsg jetstream.Msg) {
		msg := &message{msg: jsMsg}
		if err := handler(ctx, msg); err != nil {
			_ = jsMsg.Nak()
		}
	})
	if err != nil {
		return fmt.Errorf("eventmux/nats: start consume on %q: %w", consumerName, err)
	}

	b.mu.Lock()
	b.subs = append(b.subs, cc)
	b.mu.Unlock()

	// Block until context is cancelled
	<-ctx.Done()
	cc.Stop()
	return nil
}

// Close stops all consumers and drains the NATS connection.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true

	for _, s := range b.subs {
		s.Stop()
	}
	b.conn.Close()
	return nil
}

// sanitizeStreamName converts a subject pattern to a valid stream name
// by replacing special characters.
func sanitizeStreamName(topic string) string {
	buf := make([]byte, len(topic))
	for i := range len(topic) {
		c := topic[i]
		if c == '.' || c == '*' || c == '>' {
			buf[i] = '-'
		} else {
			buf[i] = c
		}
	}
	return string(buf)
}

// optsFromConfig extracts options from broker.Config.Extra.
func optsFromConfig(cfg broker.Config) []Option {
	if cfg.Extra == nil {
		return nil
	}
	var opts []Option
	if v, ok := cfg.Extra["max_deliver"].(int); ok {
		opts = append(opts, WithMaxDeliver(v))
	}
	if v, ok := cfg.Extra["replicas"].(int); ok {
		opts = append(opts, WithReplicas(v))
	}
	return opts
}
