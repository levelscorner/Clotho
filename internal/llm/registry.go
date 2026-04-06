package llm

import (
	"fmt"
	"sort"
	"sync"
)

// ProviderRegistry holds named LLM providers for lookup at execution time.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates an empty ProviderRegistry.
func NewRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider under the given name.
func (r *ProviderRegistry) Register(name string, p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = p
}

// Get returns the provider registered under name, or an error if not found.
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' not configured", name)
	}
	return p, nil
}

// List returns a sorted list of registered provider names.
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
