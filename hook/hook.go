// Package hook is handling event hookds
package hook

import "sync"

type EventHook struct {
	mu     sync.Mutex
	events map[string][]func()
}

// NewEventHook returns new instance of EventHook
func NewEventHook() *EventHook {
	return &EventHook{events: make(map[string][]func())}
}

// AddHook accepts a function which will be executed when the event is emitted.
func (e *EventHook) AddHook(eventName string, handler func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	hooks, ok := e.events[eventName]
	if !ok {
		e.events[eventName] = []func(){handler}
		return
	}

	e.events[eventName] = append(hooks, handler)
}

// RunHooks executes all hooks installed for an event.
func (e *EventHook) RunHooks(eventName string) {
	e.mu.Lock()
	hooks, ok := e.events[eventName]
	if !ok {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	for _, hook := range hooks {
		hook()
	}
}
