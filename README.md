# EventMux

Broker-agnostic, middleware-driven message routing framework for Go.

EventMux provides an **Echo-like Context API** for message brokers. Bind payloads, ack/nack messages, republish to other topics — all through a clean `c.Method()` interface, just like writing HTTP handlers.

## Features

- **Echo-style Context** — `c.Bind()`, `c.Ack()`, `c.Nack()`, `c.Republish()`
- **Broker-agnostic** — Kafka, RabbitMQ, NATS via plugins
- **Middleware-driven** — Logging, recovery, metrics (chainable)
- **Pluggable topic matching** — Exact, single-level (`*`), multi-level (`#`) wildcards
- **Pluggable serialization** — JSON by default, swap to Protobuf/Avro via `Binder`
- **Clean architecture** — Core has zero broker dependencies
- **Production-grade** — Graceful shutdown, context propagation, no data races

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "context"

    "github.com/miladsoleymani/eventmux"
    "github.com/miladsoleymani/eventmux/broker"
    "github.com/miladsoleymani/eventmux/core/middleware"

    _ "github.com/miladsoleymani/eventmux/plugins/kafka"
)

type Order struct {
    ID     int    `json:"id"`
    Amount int    `json:"amount"`
}

func main() {
    b, _ := broker.Create("kafka", broker.Config{
        Brokers: []string{"localhost:9092"},
        Group:   "my-service",
    })

    r := eventmux.New(b)
    r.Use(middleware.Recovery())
    r.Use(middleware.Logging())

    r.Handle("orders.created", func(c eventmux.Context) error {
        var order Order
        if err := c.Bind(&order); err != nil {
            return c.Nack()
        }

        fmt.Printf("Order %d: $%d\n", order.ID, order.Amount)

        if order.Amount > 10000 {
            c.Republish("orders.review") // send to review queue
        }

        return c.Ack()
    })

    r.Start(context.Background())
}
```

## Context API

Every handler receives an `eventmux.Context` — the central abstraction:

```go
r.Handle("topic", func(c eventmux.Context) error {
    // === Read ===
    c.Topic()                  // subscription topic
    c.Key()                    // message key ([]byte)
    c.Value()                  // raw body ([]byte)
    c.Header("trace-id")      // single header
    c.Headers()                // all headers
    c.Message()                // underlying Message interface
    c.Context()                // context.Context

    // === Deserialize ===
    var payload MyStruct
    c.Bind(&payload)           // JSON by default, pluggable via Binder

    // === Respond ===
    c.Ack()                    // commit offset / acknowledge
    c.Nack()                   // reject / trigger redelivery
    c.Republish("other.topic") // send to another topic (dead-letter, fan-out)

    // === Store (middleware → handler data passing) ===
    c.Set("key", value)
    val, ok := c.Get("key")

    return nil
})
```

## Architecture

```
/core              Contracts, context, router, matcher, binder, middleware
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

## Serialization

JSON by default. Swap the binder for Protobuf, Avro, or anything:

```go
type ProtobufBinder struct{}

func (ProtobufBinder) Bind(data []byte, v any) error {
    return proto.Unmarshal(data, v.(proto.Message))
}

r.SetBinder(ProtobufBinder{})
```

## Middleware

Middleware wraps handlers and executes in reverse registration order:

```go
r.Use(middleware.Recovery())  // outermost
r.Use(middleware.Logging())   // inner
```

### Built-in

- `middleware.Recovery()` — Panic recovery with stack trace logging
- `middleware.Logging()` — Topic, key, duration, and error logging
- `middleware.Metrics(collector)` — Pluggable metrics (bring your own backend)

### Custom Middleware

```go
func Auth() eventmux.MiddlewareFunc {
    return func(next eventmux.HandlerFunc) eventmux.HandlerFunc {
        return func(c eventmux.Context) error {
            token := c.Header("auth-token")
            if !valid(token) {
                return c.Nack()
            }
            c.Set("user-id", extractUserID(token))
            return next(c)
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

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a pull request.

### Quick Steps

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit (`git commit -m 'Add my feature'`)
6. Push (`git push origin feature/my-feature`)
7. Open a Pull Request

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
