package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/miladsoleymani/eventmux"
	"github.com/miladsoleymani/eventmux/broker"
	"github.com/miladsoleymani/eventmux/core/middleware"

	// Import plugins to trigger self-registration via init()
	_ "github.com/miladsoleymani/eventmux/plugins/kafka"
)

// Order represents a domain event payload.
type Order struct {
	ID     int    `json:"id"`
	Amount int    `json:"amount"`
	Status string `json:"status"`
}

func main() {
	cfg := broker.Config{
		Brokers: []string{"localhost:9092"},
		Group:   "my-service",
	}

	b, err := broker.Create("kafka", cfg)
	if err != nil {
		log.Fatalf("create broker: %v", err)
	}

	r := eventmux.New(b)

	// Global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logging())

	// Route handlers — Echo-style Context API
	r.Handle("orders.created", func(c eventmux.Context) error {
		var order Order
		if err := c.Bind(&order); err != nil {
			return c.Nack() // bad payload — nack for redelivery
		}

		fmt.Printf("Order created: %+v (trace: %s)\n", order, c.Header("trace-id"))

		// Process the order...
		if order.Amount > 10000 {
			// Route high-value orders to a review queue
			if err := c.Republish("orders.review"); err != nil {
				return err
			}
		}

		return c.Ack()
	})

	r.Handle("payments.completed", func(c eventmux.Context) error {
		fmt.Printf("Payment completed: %s\n", string(c.Value()))
		return c.Ack()
	})

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	log.Println("starting EventMux...")
	if err := r.Start(ctx); err != nil {
		log.Fatalf("router: %v", err)
	}
}
