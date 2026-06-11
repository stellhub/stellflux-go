package lifecycle

import (
	"context"
	stderrors "errors"
	"sync"
)

type Hook struct {
	Name    string
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

type Manager struct {
	mu      sync.Mutex
	hooks   []Hook
	started []Hook
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Append(hooks ...Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = append(m.hooks, hooks...)
}

func (m *Manager) Hooks() []Hook {
	m.mu.Lock()
	defer m.mu.Unlock()

	hooks := make([]Hook, len(m.hooks))
	copy(hooks, m.hooks)
	return hooks
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, hook := range m.hooks {
		if hook.OnStart == nil {
			m.started = append(m.started, hook)
			continue
		}
		if err := hook.OnStart(ctx); err != nil {
			return err
		}
		m.started = append(m.started, hook)
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var joined error
	for i := len(m.started) - 1; i >= 0; i-- {
		hook := m.started[i]
		if hook.OnStop == nil {
			continue
		}
		if err := hook.OnStop(ctx); err != nil {
			joined = stderrors.Join(joined, err)
		}
	}
	m.started = nil
	return joined
}
