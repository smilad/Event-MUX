package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Option configures the NATS broker.
type Option func(*options)

type options struct {
	// Stream
	maxMsgs     int64
	maxBytes    int64
	maxAge      time.Duration
	replicas    int
	retention   jetstream.RetentionPolicy
	storage     jetstream.StorageType

	// Consumer
	ackWait     time.Duration
	maxDeliver  int
	filterSubj  string
}

func defaults() options {
	return options{
		maxMsgs:    -1, // unlimited
		maxBytes:   -1,
		maxAge:     0,
		replicas:   1,
		retention:  jetstream.LimitsPolicy,
		storage:    jetstream.FileStorage,
		ackWait:    30 * time.Second,
		maxDeliver: 5,
	}
}

// WithMaxMessages sets the maximum number of messages per stream.
func WithMaxMessages(n int64) Option {
	return func(o *options) { o.maxMsgs = n }
}

// WithMaxBytes sets the maximum total size of a stream.
func WithMaxBytes(n int64) Option {
	return func(o *options) { o.maxBytes = n }
}

// WithMaxAge sets the maximum age of messages in the stream.
func WithMaxAge(d time.Duration) Option {
	return func(o *options) { o.maxAge = d }
}

// WithReplicas sets the stream replication factor.
func WithReplicas(n int) Option {
	return func(o *options) { o.replicas = n }
}

// WithRetention sets the stream retention policy.
func WithRetention(r jetstream.RetentionPolicy) Option {
	return func(o *options) { o.retention = r }
}

// WithStorage sets the stream storage type (file or memory).
func WithStorage(s jetstream.StorageType) Option {
	return func(o *options) { o.storage = s }
}

// WithAckWait sets how long the server waits for an ack before redelivering.
func WithAckWait(d time.Duration) Option {
	return func(o *options) { o.ackWait = d }
}

// WithMaxDeliver sets the maximum number of delivery attempts.
func WithMaxDeliver(n int) Option {
	return func(o *options) { o.maxDeliver = n }
}
