package core

import (
	"context"
	"fmt"
	"sync"
)

// Router is the central message routing engine. It provides an Echo-like API
// for registering topic handlers and middleware.
type Router struct {
	broker      Broker
	binder      Binder
	middlewares []MiddlewareFunc
	routes      map[string]HandlerFunc
	matcher     TopicMatcher
	mu          sync.RWMutex
	started     bool
}

// New creates a Router bound to the given Broker.
// It uses DefaultMatcher for topic matching and JSONBinder for deserialization.
func New(b Broker) *Router {
	return &Router{
		broker:  b,
		binder:  JSONBinder{},
		routes:  make(map[string]HandlerFunc),
		matcher: DefaultMatcher{},
	}
}

// SetMatcher replaces the topic matcher. Must be called before Start.
func (r *Router) SetMatcher(m TopicMatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matcher = m
}

// SetBinder replaces the message binder used by Context.Bind().
// Use this to switch to Protobuf, Avro, or any custom format.
func (r *Router) SetBinder(b Binder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.binder = b
}

// Use registers global middleware. Middleware is applied in reverse
// registration order (last registered wraps outermost).
func (r *Router) Use(m MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, m)
}

// Handle registers a handler for a topic pattern.
//
//	r.Handle("orders.created", func(c eventmux.Context) error {
//	    var order Order
//	    if err := c.Bind(&order); err != nil {
//	        return err
//	    }
//	    return c.Ack()
//	})
func (r *Router) Handle(topic string, h HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[topic] = h
}

// Publish sends a message to the given topic through the broker.
func (r *Router) Publish(ctx context.Context, topic string, msg Message) error {
	return r.broker.Publish(ctx, topic, msg)
}

// Start subscribes to all registered topic patterns and begins consuming
// messages. It blocks until the context is cancelled or an error occurs.
func (r *Router) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.broker == nil {
		r.mu.Unlock()
		return ErrNoBroker
	}
	if r.started {
		r.mu.Unlock()
		return ErrAlreadyStarted
	}
	r.started = true

	// Snapshot routes, middleware, and config under lock
	routes := make(map[string]HandlerFunc, len(r.routes))
	for k, v := range r.routes {
		routes[k] = v
	}
	mws := make([]MiddlewareFunc, len(r.middlewares))
	copy(mws, r.middlewares)
	matcher := r.matcher
	binder := r.binder
	broker := r.broker
	r.mu.Unlock()

	// Build the dispatching handler for each route
	var wg sync.WaitGroup
	errCh := make(chan error, len(routes))

	for pattern, handler := range routes {
		wrapped := applyMiddleware(handler, mws)

		// Bridge from low-level Handler (broker subscription) to Context-based HandlerFunc
		bridgeHandler := func(c context.Context, msg Message) error {
			ec := NewContext(c, msg, pattern, broker, binder)
			return wrapped(ec)
		}

		// matcher available for future per-message filtering
		_ = matcher

		wg.Add(1)
		go func(p string, h Handler) {
			defer wg.Done()
			if err := broker.Subscribe(ctx, p, h); err != nil {
				errCh <- fmt.Errorf("eventmux: subscribe %q: %w", p, err)
			}
		}(pattern, bridgeHandler)
	}

	// Wait for context cancellation or subscription errors
	go func() {
		wg.Wait()
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return r.broker.Close()
	case err := <-errCh:
		if err != nil {
			return err
		}
		// All subscriptions returned without error — wait for context
		<-ctx.Done()
		return r.broker.Close()
	}
}

// applyMiddleware wraps a handler with middleware in reverse order.
// Given middleware [A, B, C], the call order is A -> B -> C -> handler.
func applyMiddleware(h HandlerFunc, mws []MiddlewareFunc) HandlerFunc {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
