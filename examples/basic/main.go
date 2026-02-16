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

	// Route handlers
	r.Handle("orders.created", func(ctx context.Context, msg eventmux.Message) error {
		fmt.Printf("Order created: %s\n", string(msg.Value()))
		return msg.Ack()
	})

	r.Handle("payments.completed", func(ctx context.Context, msg eventmux.Message) error {
		fmt.Printf("Payment completed: %s\n", string(msg.Value()))
		return msg.Ack()
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
