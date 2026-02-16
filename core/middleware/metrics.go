package middleware

import (
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
func Metrics(collector MetricsCollector) core.MiddlewareFunc {
	return func(next core.HandlerFunc) core.HandlerFunc {
		return func(c core.Context) error {
			start := time.Now()
			err := next(c)
			collector.MessageProcessed(c.Topic(), time.Since(start), err)
			return err
		}
	}
}
