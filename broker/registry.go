package broker

import (
	"fmt"
	"sync"

	"github.com/miladsoleymani/eventmux/core"
)

// Factory creates a Broker from the given Config.
type Factory func(cfg Config) (core.Broker, error)

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register adds a named broker factory. Plugins call this from init().
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = factory
}

// Create instantiates a broker by name using the registered factory.
func Create(name string, cfg Config) (core.Broker, error) {
	mu.RLock()
	f, ok := factories[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("eventmux: unknown broker %q", name)
	}
	return f(cfg)
}
