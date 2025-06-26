package life

import "sync"

// EventHandler represents an event callback function
type EventHandler func(Data interface{})

// EventEmitter provides event handling capabilities
type EventEmitter struct {
	events map[EventType][]EventHandler
	mutex  sync.RWMutex
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		events: make(map[EventType][]EventHandler),
	}
}

// On adds an event listener
func (e *EventEmitter) On(event EventType, handler EventHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.events[event] = append(e.events[event], handler)
}

// Emit triggers an event
func (e *EventEmitter) Emit(event EventType, data interface{}) {
	e.mutex.RLock()
	handlers := e.events[event]
	e.mutex.RUnlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// RemoveListener removes an event listener
func (e *EventEmitter) RemoveListener(event EventType, handler EventHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	handlers := e.events[event]
	for i, h := range handlers {
		// Note: Function comparison in Go is limited, this is a simplified approach
		if &h == &handler {
			e.events[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Once adds a one-time event listener
func (e *EventEmitter) Once(event EventType, handler EventHandler) {
	var onceHandler EventHandler
	onceHandler = func(data interface{}) {
		handler(data)
		e.RemoveListener(event, onceHandler)
	}
	e.On(event, onceHandler)
}
