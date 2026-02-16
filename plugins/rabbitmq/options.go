package rabbitmq

// Option configures the RabbitMQ broker.
type Option func(*options)

type options struct {
	// Exchange settings
	exchange     string
	exchangeType string
	routingKey   string

	// Queue settings
	durable    bool
	autoDelete bool
	exclusive  bool

	// Consumer settings
	prefetchCount int
	requeueOnNack bool
}

func defaults() options {
	return options{
		exchange:      "",       // default exchange
		exchangeType:  "direct", // direct, fanout, topic, headers
		durable:       true,
		prefetchCount: 10,
		requeueOnNack: true,
	}
}

// WithExchange sets the exchange name and type.
func WithExchange(name, kind string) Option {
	return func(o *options) {
		o.exchange = name
		o.exchangeType = kind
	}
}

// WithRoutingKey sets the routing key for queue binding.
func WithRoutingKey(key string) Option {
	return func(o *options) { o.routingKey = key }
}

// WithDurable controls whether queues survive broker restart.
func WithDurable(d bool) Option {
	return func(o *options) { o.durable = d }
}

// WithPrefetchCount sets how many messages are delivered before requiring ack.
func WithPrefetchCount(n int) Option {
	return func(o *options) { o.prefetchCount = n }
}

// WithRequeueOnNack controls whether nacked messages are requeued.
func WithRequeueOnNack(requeue bool) Option {
	return func(o *options) { o.requeueOnNack = requeue }
}

// WithAutoDelete causes the queue to be deleted when the last consumer disconnects.
func WithAutoDelete(d bool) Option {
	return func(o *options) { o.autoDelete = d }
}
