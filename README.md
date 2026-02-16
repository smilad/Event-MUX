# EventMux

Broker-agnostic, middleware-driven message routing framework for Go.

EventMux provides an Echo-like API for message brokers. Register topic handlers, apply middleware, and swap brokers without changing business logic.

## Features

- **Broker-agnostic** — Kafka, RabbitMQ, NATS via plugins
- **Middleware-driven** — Logging, recovery, metrics (chainable)
- **Pluggable topic matching** — Exact, single-level (`*`), multi-level (`#`) wildcards
- **Clean architecture** — Core has zero broker dependencies
- **Production-grade** — Graceful shutdown, context propagation, no data races

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/miladsoleymani/eventmux"
    "github.com/miladsoleymani/eventmux/broker"
    "github.com/miladsoleymani/eventmux/core/middleware"

    _ "github.com/miladsoleymani/eventmux/plugins/kafka"
)

func main() {
    b, err := broker.Create("kafka", broker.Config{
        Brokers: []string{"localhost:9092"},
        Group:   "my-service",
    })
    if err != nil {
        log.Fatal(err)
    }

    r := eventmux.New(b)
    r.Use(middleware.Recovery())
    r.Use(middleware.Logging())

    r.Handle("orders.created", func(ctx context.Context, msg eventmux.Message) error {
        fmt.Printf("Order: %s\n", msg.Value())
        return msg.Ack()
    })

    r.Start(context.Background())
}
```

## Architecture

```
/core              Contracts, router, matcher, middleware (no broker imports)
/broker            Registry + config (factory pattern)
/plugins/kafka     Kafka adapter (segmentio/kafka-go)
/plugins/rabbitmq  RabbitMQ adapter (amqp091-go)
/plugins/nats      NATS JetStream adapter (nats.go)
/internal/mock     Test doubles
/examples          Usage examples
```

## Topic Matching

| Pattern | Topic | Match |
|---------|-------|-------|
| `orders.created` | `orders.created` | Exact |
| `orders.*` | `orders.created` | Single-level wildcard |
| `orders.*` | `orders.us.created` | No match |
| `payments.#` | `payments.us.created` | Multi-level wildcard |

Replace the matcher:

```go
r.SetMatcher(myCustomMatcher)
```

## Middleware

Middleware wraps handlers and executes in reverse registration order:

```go
r.Use(middleware.Recovery())  // outermost
r.Use(middleware.Logging())   // inner
```

### Built-in

- `middleware.Recovery()` — Panic recovery with stack trace logging
- `middleware.Logging()` — Request duration and error logging
- `middleware.Metrics(topic, collector)` — Pluggable metrics (bring your own backend)

### Custom Middleware

```go
func Auth() eventmux.Middleware {
    return func(next eventmux.Handler) eventmux.Handler {
        return func(ctx context.Context, msg eventmux.Message) error {
            token := msg.Headers()["auth-token"]
            if !valid(token) {
                return errors.New("unauthorized")
            }
            return next(ctx, msg)
        }
    }
}
```

## Broker Plugins

Import a plugin to register it:

```go
import _ "github.com/miladsoleymani/eventmux/plugins/kafka"
import _ "github.com/miladsoleymani/eventmux/plugins/rabbitmq"
import _ "github.com/miladsoleymani/eventmux/plugins/nats"
```

Then create by name:

```go
b, err := broker.Create("kafka", cfg)
b, err := broker.Create("rabbitmq", cfg)
b, err := broker.Create("nats", cfg)
```

## Development

```bash
make test      # Core + broker + internal tests (no external deps)
make build     # Compile all packages
make lint      # go vet
make tidy      # go mod tidy
```

## License

MIT
