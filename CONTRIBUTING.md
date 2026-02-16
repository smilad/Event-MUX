# Contributing to EventMux

Thank you for your interest in contributing to EventMux! This document provides guidelines and information to make the contribution process smooth for everyone.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment. Be kind, constructive, and professional in all interactions.

## How to Contribute

### Reporting Bugs

Before filing a bug report:

1. Check existing [issues](https://github.com/miladsoleymani/eventmux/issues) to avoid duplicates
2. Use the bug report template and include:
   - Go version (`go version`)
   - EventMux version or commit hash
   - Broker type and version (Kafka, RabbitMQ, NATS)
   - Minimal reproducible example
   - Expected vs actual behavior

### Suggesting Features

Open an issue with the **feature request** label. Include:

- A clear description of the problem it solves
- Proposed API design (if applicable)
- Alternatives you considered

### Submitting Code

#### Setup

```bash
# Fork and clone
git clone https://github.com/<your-username>/eventmux.git
cd eventmux

# Install dependencies
go mod tidy

# Verify everything works
make test
make lint
```

#### Development Workflow

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Write your code** following the guidelines below

3. **Add tests** for any new functionality

4. **Run the full check**:
   ```bash
   make test
   make lint
   ```

5. **Commit** with a clear message:
   ```bash
   git commit -m 'Add support for dead-letter queues in Kafka plugin'
   ```

6. **Push** and open a Pull Request against `main`

#### Pull Request Requirements

- [ ] All existing tests pass (`make test`)
- [ ] New code has test coverage
- [ ] No data races (`-race` flag passes)
- [ ] Code passes `go vet`
- [ ] Commit messages are clear and descriptive
- [ ] PR description explains **what** and **why**

## Code Guidelines

### Architecture Rules

These rules are non-negotiable:

- **`/core` must never import broker-specific packages.** The core package defines contracts only. If your change introduces a broker import into `/core`, it will be rejected.
- **Core contracts (`Message`, `Handler`, `Middleware`, `Broker`) are frozen.** Do not modify these interfaces.
- **Plugins self-register via `init()`.** Every broker plugin must call `broker.Register()` in its `init()` function.

### Go Style

- Follow standard [Go conventions](https://go.dev/doc/effective_go)
- Use `gofmt` / `goimports` for formatting
- Exported types and functions must have doc comments
- Error messages follow the format: `eventmux/<package>: <context>: %w`
- Wrap errors with `fmt.Errorf` and `%w` for unwrapping support
- No panics in production code paths

### Naming

- Interfaces describe behavior: `TopicMatcher`, `MetricsCollector`
- Options use the functional pattern: `WithPrefetchCount(n int) Option`
- Test files are `*_test.go` in the same package (or `_test` package for black-box tests)

### Concurrency

- Protect shared state with `sync.Mutex`
- All tests must pass with `-race`
- Use `context.Context` for cancellation propagation
- Avoid goroutine leaks — every goroutine must have a clear exit path

### Testing

- Unit tests go alongside the code they test
- Use `/internal/mock` for broker test doubles — do not mock with external libraries
- Plugin integration tests (requiring a running broker) should use build tags:
  ```go
  //go:build integration
  ```
- Aim for table-driven tests where appropriate

## Adding a New Broker Plugin

To add support for a new broker (e.g., Redis Streams, Pulsar):

1. Create `/plugins/<name>/` with three files:
   - `<name>.go` — `Broker` struct implementing `core.Broker`
   - `message.go` — message adapter implementing `core.Message`
   - `options.go` — functional options with `defaults()`

2. Self-register in `init()`:
   ```go
   func init() {
       broker.Register("<name>", func(cfg broker.Config) (core.Broker, error) {
           return New(cfg)
       })
   }
   ```

3. Implement all `core.Broker` methods:
   - `Publish` — send a message
   - `Subscribe` — blocking consume loop, respects context cancellation
   - `Close` — graceful teardown of all connections

4. Implement `core.Message`:
   - `Ack()` — acknowledge/commit
   - `Nack()` — negative acknowledge with appropriate broker semantics

5. Add the dependency to `go.mod`

6. Add an example in `/examples/`

7. Document the plugin in `README.md`

## Adding Middleware

Built-in middleware lives in `/core/middleware/`. To add new middleware:

1. Create a new file in `/core/middleware/`
2. Follow the signature: `func MyMiddleware() core.Middleware`
3. Add tests in `middleware_test.go`
4. Keep it decoupled — define interfaces for external dependencies (like `MetricsCollector`)

## Release Process

Releases follow [semantic versioning](https://semver.org/):

- **PATCH** (v1.0.x): Bug fixes, no API changes
- **MINOR** (v1.x.0): New features, backward compatible
- **MAJOR** (vX.0.0): Breaking changes to core contracts (extremely rare)

## Questions?

Open a [discussion](https://github.com/miladsoleymani/eventmux/discussions) or reach out via issues. We're happy to help you get started.
