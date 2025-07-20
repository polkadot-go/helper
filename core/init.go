package core

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Initializer interface {
	Name() string
	Dependencies() []string
	Init() error
}

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

type Component interface {
	Initializer
	Shutdowner
}

type Registry struct {
	mu            sync.Mutex
	components    map[string]interface{}
	initialized   map[string]bool
	initOrder     []string
	shutdownHooks []func(context.Context) error
}

var (
	registry = &Registry{
		components:  make(map[string]interface{}),
		initialized: make(map[string]bool),
	}
)

func Register(component interface{}) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if init, ok := component.(Initializer); ok {
		registry.components[init.Name()] = component
	}
}

func Initialize() error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	order, err := registry.topologicalSort()
	if err != nil {
		return err
	}
	registry.initOrder = order

	for _, name := range order {
		if err := registry.initOne(name); err != nil {
			return fmt.Errorf("initializing %s: %w", name, err)
		}
	}

	return nil
}

func Shutdown(ctx context.Context) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Shutdown in reverse order
	for i := len(registry.initOrder) - 1; i >= 0; i-- {
		name := registry.initOrder[i]
		if comp, ok := registry.components[name]; ok {
			if s, ok := comp.(Shutdowner); ok {
				if err := s.Shutdown(ctx); err != nil {
					return fmt.Errorf("shutting down %s: %w", name, err)
				}
			}
		}
	}

	// Run additional shutdown hooks
	for _, hook := range registry.shutdownHooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

func RegisterShutdownHook(hook func(context.Context) error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.shutdownHooks = append(registry.shutdownHooks, hook)
}

func (r *Registry) initOne(name string) error {
	if r.initialized[name] {
		return nil
	}

	comp, ok := r.components[name]
	if !ok {
		return fmt.Errorf("unknown component: %s", name)
	}

	init, ok := comp.(Initializer)
	if !ok {
		return fmt.Errorf("%s does not implement Initializer", name)
	}

	for _, dep := range init.Dependencies() {
		if !r.initialized[dep] {
			if err := r.initOne(dep); err != nil {
				return err
			}
		}
	}

	if err := init.Init(); err != nil {
		return err
	}

	r.initialized[name] = true
	return nil
}

func (r *Registry) topologicalSort() ([]string, error) {
	var order []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}
		if visited[name] {
			return nil
		}

		visiting[name] = true

		if comp, ok := r.components[name]; ok {
			if init, ok := comp.(Initializer); ok {
				for _, dep := range init.Dependencies() {
					if err := visit(dep); err != nil {
						return err
					}
				}
			}
		}

		visiting[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

func GetComponent(name string) interface{} {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.components[name]
}

func IsInitialized(name string) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.initialized[name]
}

func GetInitOrder() []string {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return append([]string{}, registry.initOrder...)
}

func MustInitialize() {
	if err := Initialize(); err != nil {
		panic(err)
	}
}

func GracefulShutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Shutdown(ctx)
}
