// Package hook is handling event hookds
package hook

type EventHook struct {
	events map[string][]func()
}

// NewEventHook returns new instance of EventHook
func NewEventHook() *EventHook {
	return &EventHook{make(map[string][]func())}
}

// AddHook accepts a function which will be executed when the event is emitted.
func (e *EventHook) AddHook(eventName string, handler func()) {

	hooks, ok := e.events[eventName]
	if !ok {
		e.events[eventName] = []func(){handler}
		return
	}

	e.events[eventName] = append(hooks, handler)
}

// RunHooks executes all hooks installed for an event.
func (e *EventHook) RunHooks(eventName string) {

	hooks, ok := e.events[eventName]
	if !ok {
		return
	}

	for _, hook := range hooks {
		hook()
	}
}
