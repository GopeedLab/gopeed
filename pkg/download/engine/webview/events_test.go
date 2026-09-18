package webview

import (
	"reflect"
	"sync"
	"testing"
)

func TestEventHubLifecycle(t *testing.T) {
	var hub EventHub
	var got []string
	hub.Emit(Event{Name: "load"}) // Never replay pre-registration events.
	off, err := hub.On("load", func(Event) { got = append(got, "first") })
	if err != nil {
		t.Fatal(err)
	}
	_, _ = hub.On("load", func(Event) { got = append(got, "second") })
	_, _ = hub.On("closed", func(e Event) {
		got = append(got, e.Data["reason"].(string))
		off() // Unsubscription from a callback must not deadlock.
	})
	hub.Emit(Event{Name: "load"})
	off()
	off()
	hub.Emit(Event{Name: "load"})
	hub.Emit(Event{Name: "closed", Data: map[string]any{"reason": "api"}})
	hub.Emit(Event{Name: "load"})
	hub.Emit(Event{Name: "closed"})
	if !reflect.DeepEqual(got, []string{"first", "second", "second", "api"}) {
		t.Fatal(got)
	}
	if len(hub.listeners) != 0 {
		t.Fatal("closed page retained listeners")
	}
	if _, err := hub.On("load", func(Event) {}); err == nil {
		t.Fatal("registration on closed page succeeded")
	}
}

func TestEventHubConcurrentUnsubscribe(t *testing.T) {
	var hub EventHub
	off, _ := hub.On("load", func(Event) {})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); off(); hub.Emit(Event{Name: "load"}) }()
	}
	wg.Wait()
	if _, err := hub.On("invalid", func(Event) {}); err == nil {
		t.Fatal("accepted invalid event")
	}
}
