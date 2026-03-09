package cognitive

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	factories = make(map[string]func(Config) (Gateway, error))
)

// Register registers a provider factory by name.
// Called from init() by provider packages.
func Register(name string, factory func(Config) (Gateway, error)) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = factory
}

// NewGateway creates a Gateway from config using the registered provider.
func NewGateway(cfg Config) (Gateway, error) {
	mu.RLock()
	factory, ok := factories[cfg.Provider]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("cognitive provider %q not registered", cfg.Provider)
	}
	return factory(cfg)
}
