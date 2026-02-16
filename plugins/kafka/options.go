package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Option configures the Kafka broker.
type Option func(*options)

type options struct {
	// Writer
	balancer  kafka.Balancer
	batchSize int
	async     bool

	// Reader
	minBytes     int
	maxBytes     int
	maxWait      time.Duration
	startOffset  int64
	commitPeriod time.Duration

	// General
	dialer *kafka.Dialer
}

func defaults() options {
	return options{
		balancer:     &kafka.LeastBytes{},
		batchSize:    100,
		minBytes:     1,
		maxBytes:     10e6, // 10 MB
		maxWait:      500 * time.Millisecond,
		startOffset:  kafka.LastOffset,
		commitPeriod: 0, // manual commit by default
	}
}

// WithBalancer sets the partition balancer for the writer.
func WithBalancer(b kafka.Balancer) Option {
	return func(o *options) { o.balancer = b }
}

// WithBatchSize sets the maximum batch size for writes.
func WithBatchSize(n int) Option {
	return func(o *options) { o.batchSize = n }
}

// WithAsync enables asynchronous writes.
func WithAsync(async bool) Option {
	return func(o *options) { o.async = async }
}

// WithMaxBytes sets the maximum bytes per fetch.
func WithMaxBytes(n int) Option {
	return func(o *options) { o.maxBytes = n }
}

// WithMaxWait sets the maximum wait time for fetches.
func WithMaxWait(d time.Duration) Option {
	return func(o *options) { o.maxWait = d }
}

// WithStartOffset sets the consumer start offset (kafka.FirstOffset or kafka.LastOffset).
func WithStartOffset(offset int64) Option {
	return func(o *options) { o.startOffset = offset }
}

// WithDialer sets a custom dialer for TLS/SASL connections.
func WithDialer(d *kafka.Dialer) Option {
	return func(o *options) { o.dialer = d }
}
