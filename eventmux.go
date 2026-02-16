// Package eventmux provides the top-level API for the EventMux framework.
// It re-exports core types for convenience, so users can write:
//
//	r := eventmux.New(b)
//	r.Handle("orders.created", handler)
//	r.Start(ctx)
package eventmux

import (
	"github.com/miladsoleymani/eventmux/core"
)

// Re-export core types at the package level for ergonomic usage.
type (
	Message    = core.Message
	Handler    = core.Handler
	Middleware = core.Middleware
	Broker     = core.Broker
	Router     = core.Router
)

// New creates a new Router bound to the given Broker.
func New(b Broker) *Router {
	return core.New(b)
}
