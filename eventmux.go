// Package eventmux provides the top-level API for the EventMux framework.
// It re-exports core types for convenience, so users can write:
//
//	r := eventmux.New(b)
//	r.Handle("orders.created", func(c eventmux.Context) error {
//	    var order Order
//	    if err := c.Bind(&order); err != nil {
//	        return err
//	    }
//	    return c.Ack()
//	})
//	r.Start(ctx)
package eventmux

import (
	"github.com/miladsoleymani/eventmux/core"
)

// Re-export core types at the package level for ergonomic usage.
type (
	Context       = core.Context
	Message       = core.Message
	HandlerFunc   = core.HandlerFunc
	MiddlewareFunc = core.MiddlewareFunc
	Broker        = core.Broker
	Binder        = core.Binder
	Router        = core.Router
)

// New creates a new Router bound to the given Broker.
func New(b Broker) *Router {
	return core.New(b)
}
