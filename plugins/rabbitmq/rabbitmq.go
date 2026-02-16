package rabbitmq

import (
	"context"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/miladsoleymani/eventmux/broker"
	"github.com/miladsoleymani/eventmux/core"
)

func init() {
	broker.Register("rabbitmq", func(cfg broker.Config) (core.Broker, error) {
		opts := optsFromConfig(cfg)
		if len(cfg.Brokers) == 0 {
			return nil, fmt.Errorf("eventmux/rabbitmq: at least one broker URI is required")
		}
		return New(cfg.Brokers[0], opts...)
	})
}

// Broker implements core.Broker for RabbitMQ using amqp091-go.
//
// Design decisions:
//   - Single connection, one channel per Broker instance.
//   - Manual ack mode — consumers must call Ack() or Nack() explicitly.
//   - Durable queues by default for production reliability.
//   - Configurable prefetch count for backpressure control.
//   - Graceful shutdown: context cancellation exits the consume loop,
//     Close() tears down channel and connection.
type Broker struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	opts options
	mu   sync.Mutex
	closed bool
}

// New creates a RabbitMQ Broker. uri is a standard AMQP URI (amqp://user:pass@host:port/vhost).
func New(uri string, fns ...Option) (*Broker, error) {
	opts := defaults()
	for _, fn := range fns {
		fn(&opts)
	}

	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("eventmux/rabbitmq: dial %q: %w", uri, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("eventmux/rabbitmq: open channel: %w", err)
	}

	if err := ch.Qos(opts.prefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("eventmux/rabbitmq: set qos: %w", err)
	}

	return &Broker{conn: conn, ch: ch, opts: opts}, nil
}

// Publish sends a message to the specified queue (or exchange routing key).
func (b *Broker) Publish(ctx context.Context, topic string, msg core.Message) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return core.ErrBrokerClosed
	}
	ch := b.ch
	b.mu.Unlock()

	headers := amqp.Table{}
	for k, v := range msg.Headers() {
		headers[k] = v
	}

	exchange := b.opts.exchange
	routingKey := topic
	if b.opts.routingKey != "" {
		routingKey = b.opts.routingKey
	}

	if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		Body:    msg.Value(),
		Headers: headers,
	}); err != nil {
		return fmt.Errorf("eventmux/rabbitmq: publish to %q: %w", topic, err)
	}
	return nil
}

// Subscribe declares a durable queue, binds it (if using an exchange),
// and consumes messages until the context is cancelled.
func (b *Broker) Subscribe(ctx context.Context, topic string, handler core.Handler) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return core.ErrBrokerClosed
	}
	ch := b.ch
	b.mu.Unlock()

	q, err := ch.QueueDeclare(
		topic,
		b.opts.durable,
		b.opts.autoDelete,
		b.opts.exclusive,
		false, // noWait
		nil,
	)
	if err != nil {
		return fmt.Errorf("eventmux/rabbitmq: declare queue %q: %w", topic, err)
	}

	// Bind to exchange if one is configured
	if b.opts.exchange != "" {
		rk := topic
		if b.opts.routingKey != "" {
			rk = b.opts.routingKey
		}
		if err := ch.QueueBind(q.Name, rk, b.opts.exchange, false, nil); err != nil {
			return fmt.Errorf("eventmux/rabbitmq: bind queue %q: %w", q.Name, err)
		}
	}

	deliveries, err := ch.Consume(
		q.Name,
		"",    // consumer tag (auto-generated)
		false, // autoAck — manual ack mode
		b.opts.exclusive,
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		return fmt.Errorf("eventmux/rabbitmq: consume %q: %w", q.Name, err)
	}

	return b.consumeLoop(ctx, deliveries, handler)
}

// consumeLoop processes deliveries until context cancellation or channel close.
func (b *Broker) consumeLoop(ctx context.Context, deliveries <-chan amqp.Delivery, handler core.Handler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil // channel closed
			}
			msg := &message{delivery: d, requeue: b.opts.requeueOnNack}
			if err := handler(ctx, msg); err != nil {
				_ = d.Nack(false, b.opts.requeueOnNack)
				continue
			}
		}
	}
}

// Close tears down the channel and connection.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true

	var errs []error
	if err := b.ch.Close(); err != nil {
		errs = append(errs, fmt.Errorf("eventmux/rabbitmq: close channel: %w", err))
	}
	if err := b.conn.Close(); err != nil {
		errs = append(errs, fmt.Errorf("eventmux/rabbitmq: close connection: %w", err))
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// optsFromConfig extracts options from broker.Config.Extra.
func optsFromConfig(cfg broker.Config) []Option {
	if cfg.Extra == nil {
		return nil
	}
	var opts []Option
	if ex, ok := cfg.Extra["exchange"].(string); ok {
		kind := "direct"
		if k, ok := cfg.Extra["exchange_type"].(string); ok {
			kind = k
		}
		opts = append(opts, WithExchange(ex, kind))
	}
	if rk, ok := cfg.Extra["routing_key"].(string); ok {
		opts = append(opts, WithRoutingKey(rk))
	}
	if pf, ok := cfg.Extra["prefetch_count"].(int); ok {
		opts = append(opts, WithPrefetchCount(pf))
	}
	return opts
}
