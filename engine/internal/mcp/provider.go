// Package mcp manages code agent provider sessions via stdio-based JSON-line protocol.
package mcp

import (
	"sort"
	"sync"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// ProviderSpec describes a code agent provider's command and environment.
type ProviderSpec struct {
	Name    domain.Provider
	Command string
	Args    []string
	Env     map[string]string
}

// ProviderRegistry is a thread-safe registry of provider specifications.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[domain.Provider]ProviderSpec
}

// NewProviderRegistry creates an empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[domain.Provider]ProviderSpec),
	}
}

// Register adds a provider spec to the registry.
// Returns ErrProviderUnavailable if a provider with the same name is already registered.
func (r *ProviderRegistry) Register(spec ProviderSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[spec.Name]; exists {
		return domain.WrapEngineError(
			domain.ErrProviderUnavailable.Code,
			"provider already registered",
			nil,
		)
	}
	r.providers[spec.Name] = spec
	return nil
}

// Get returns the spec for the named provider, or ErrProviderUnavailable if not found.
func (r *ProviderRegistry) Get(name domain.Provider) (ProviderSpec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	spec, ok := r.providers[name]
	if !ok {
		return ProviderSpec{}, domain.ErrProviderUnavailable
	}
	return spec, nil
}

// List returns all registered provider names in sorted order.
func (r *ProviderRegistry) List() []domain.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]domain.Provider, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return string(names[i]) < string(names[j])
	})
	return names
}
