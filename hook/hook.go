package hook

type EventHook struct {
	events map[string][]func()
}

func NewEventHook() EventHook {
	return EventHook{make(map[string][]func())}
}

func (e *EventHook) AddHook(eventName string, handler func()) {

	hooks, ok := e.events[eventName]
	if !ok {
		e.events[eventName] = []func(){handler}
	}

	e.events[eventName] = append(hooks, handler)
}

func (e *EventHook) RunHooks(eventName string) {

	hooks, ok := e.events[eventName]
	if !ok {
		return
	}

	for _, hook := range hooks {
		hook()
	}
}
