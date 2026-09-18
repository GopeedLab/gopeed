package webview

import (
	"fmt"
	"sync"
)

// Event is a main-page notification. Data follows WebviewEventMap.
type Event struct {
	Name string         `json:"event"`
	Data map[string]any `json:"data"`
}

type EventSource interface {
	On(event string, handler func(Event)) (unsubscribe func(), err error)
}

func ValidEvent(event string) bool {
	switch event {
	case "url-changed", "load", "load-error", "closed":
		return true
	}
	return false
}

// EventHub delivers notifications in registration order. Handlers must only
// enqueue work, never block or enter the browser. Closed is terminal.
type EventHub struct {
	mu        sync.Mutex
	seq       uint64
	listeners []eventListener
	closed    bool
}
type eventListener struct {
	id      uint64
	name    string
	handler func(Event)
}

func (h *EventHub) On(name string, handler func(Event)) (func(), error) {
	if !ValidEvent(name) || handler == nil {
		return nil, fmt.Errorf("invalid webview event or handler: %s", name)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, fmt.Errorf("webview page is closed")
	}
	h.seq++
	id := h.seq
	h.listeners = append(h.listeners, eventListener{id, name, handler})
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		for i, listener := range h.listeners {
			if listener.id == id {
				h.listeners = append(h.listeners[:i], h.listeners[i+1:]...)
				break
			}
		}
	}, nil
}

func (h *EventHub) Emit(event Event) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	listeners := append([]eventListener(nil), h.listeners...)
	if event.Name == "closed" {
		h.closed = true
		h.listeners = nil
	}
	h.mu.Unlock()
	for _, listener := range listeners {
		if listener.name == event.Name {
			listener.handler(event)
		}
	}
}

func (p *PageHandle) On(name string, handler func(Event)) (func(), error) {
	page, err := p.page()
	if err != nil {
		return nil, err
	}
	source, ok := page.(EventSource)
	if !ok {
		return nil, fmt.Errorf("WebView provider does not support events")
	}
	return source.On(name, handler)
}
