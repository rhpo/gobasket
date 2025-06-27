package life

import "sync"

type EventHandler func(Data interface{})

type EventEmitter struct {
	events map[EventType][]EventHandler
	mutex  sync.RWMutex
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		events: make(map[EventType][]EventHandler),
	}
}

func (e *EventEmitter) On(event EventType, handler EventHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.events[event] = append(e.events[event], handler)
}

func (e *EventEmitter) Emit(event EventType, data interface{}) {
	e.mutex.RLock()
	handlers := e.events[event]
	e.mutex.RUnlock()

	for _, handler := range handlers {
		handler(data)
	}
}

func (e *EventEmitter) RemoveListener(event EventType, handler EventHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	handlers := e.events[event]
	for i, h := range handlers {

		if &h == &handler {
			e.events[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (e *EventEmitter) Once(event EventType, handler EventHandler) {
	var onceHandler EventHandler
	onceHandler = func(data interface{}) {
		handler(data)
		e.RemoveListener(event, onceHandler)
	}
	e.On(event, onceHandler)
}
