package middleware

import (
	"context"
	"time"

	"github.com/miladsoleymani/eventmux/core"
)

// MetricsCollector is the interface that metrics backends must implement.
// This keeps the middleware decoupled from any specific metrics library.
type MetricsCollector interface {
	// MessageProcessed records that a message was processed.
	// topic is the subscription pattern, duration is processing time,
	// and err is nil on success.
	MessageProcessed(topic string, duration time.Duration, err error)
}

// Metrics returns middleware that reports processing metrics to the given collector.
// The topic parameter identifies the subscription for metric labeling.
func Metrics(topic string, collector MetricsCollector) core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, msg core.Message) error {
			start := time.Now()
			err := next(ctx, msg)
			collector.MessageProcessed(topic, time.Since(start), err)
			return err
		}
	}
}
