package boot

import "sync"

type Registry struct {
	mu     sync.RWMutex
	values map[string]any
}

func NewRegistry() *Registry {
	return &Registry{values: map[string]any{}}
}

func (r *Registry) Set(name string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[name] = value
}

func (r *Registry) Get(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.values[name]
	return value, ok
}

func GetAs[T any](r *Registry, name string) (T, bool) {
	var zero T
	value, ok := r.Get(name)
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.values))
	for name := range r.values {
		names = append(names, name)
	}
	return names
}
