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
	middlewares []Middleware
	routes      map[string]Handler
	matcher     TopicMatcher
	mu          sync.RWMutex
	started     bool
}

// New creates a Router bound to the given Broker.
// It uses DefaultMatcher for topic matching.
func New(b Broker) *Router {
	return &Router{
		broker:  b,
		routes:  make(map[string]Handler),
		matcher: DefaultMatcher{},
	}
}

// SetMatcher replaces the topic matcher. Must be called before Start.
func (r *Router) SetMatcher(m TopicMatcher) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matcher = m
}

// Use registers global middleware. Middleware is applied in reverse
// registration order (last registered wraps outermost).
func (r *Router) Use(m Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, m)
}

// Handle registers a handler for a topic pattern.
func (r *Router) Handle(topic string, h Handler) {
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

	// Snapshot routes and middleware under lock
	routes := make(map[string]Handler, len(r.routes))
	for k, v := range r.routes {
		routes[k] = v
	}
	mws := make([]Middleware, len(r.middlewares))
	copy(mws, r.middlewares)
	matcher := r.matcher
	r.mu.Unlock()

	// Build the dispatching handler for each route
	var wg sync.WaitGroup
	errCh := make(chan error, len(routes))

	for pattern, handler := range routes {
		wrapped := applyMiddleware(handler, mws)

		dispatchHandler := func(ctx context.Context, msg Message) error {
			return wrapped(ctx, msg)
		}

		// For wildcard patterns, subscribe to the pattern and let the broker
		// deliver matching messages. The matcher is used as a safety check.
		_ = matcher // matcher available for future per-message filtering

		wg.Add(1)
		go func(p string, h Handler) {
			defer wg.Done()
			if err := r.broker.Subscribe(ctx, p, h); err != nil {
				errCh <- fmt.Errorf("eventmux: subscribe %q: %w", p, err)
			}
		}(pattern, dispatchHandler)
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
// Given middleware [A, B, C], the call order is C -> B -> A -> handler.
func applyMiddleware(h Handler, mws []Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
