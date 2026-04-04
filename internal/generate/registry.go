package generate

import (
	"fmt"
	"sync"
)

// Registry holds named Template factories.
type Registry struct {
	mu       sync.RWMutex
	factories map[string]func() Template
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]func() Template),
	}
}

// Register adds a named template factory to the registry.
// Panics if name is already registered.
func (r *Registry) Register(name string, factory func() Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[name]; exists {
		panic(fmt.Sprintf("generate: template %q already registered", name))
	}
	r.factories[name] = factory
}

// New creates a new Template instance from the named factory.
// Returns an error if the name is not registered.
func (r *Registry) New(name string) (Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("generate: template %q not registered", name)
	}
	return factory(), nil
}

// Names returns all registered template names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for n := range r.factories {
		names = append(names, n)
	}
	return names
}

// DefaultRegistry is the package-level registry used by the CLI.
var DefaultRegistry = NewRegistry()

func init() {
	DefaultRegistry.Register("basic", func() Template {
		return &BaseTemplate{WorkflowName: "CI", Runner: "ubuntu-latest"}
	})
	DefaultRegistry.Register("advanced", func() Template {
		return NewAdvancedTemplate(
			"Advanced CI",
			[]string{"ubuntu-latest", "macos-latest"},
			[]string{"1.21", "1.22", "1.23"},
			"ci-${{ github.ref }}",
		)
	})
}
