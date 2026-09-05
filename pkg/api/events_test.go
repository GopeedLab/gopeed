package api

import (
	"encoding/json"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download"
)

func TestTaskEventSubscriptionFiltersMask(t *testing.T) {
	service := &Service{}
	var payloads []string
	service.SubscribeTaskEvents(TaskEventDone, func(payload string) {
		payloads = append(payloads, payload)
	})

	task := &download.Task{ID: "task-1"}
	service.emitTaskEvent(&download.Event{Key: download.EventKeyError, Task: task})
	service.emitTaskEvent(&download.Event{Key: download.EventKeyDone, Task: task})

	if len(payloads) != 1 {
		t.Fatalf("got %d callbacks, want 1", len(payloads))
	}
	var event TaskEvent
	if err := json.Unmarshal([]byte(payloads[0]), &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "task.done" || event.TaskID != task.ID {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestTaskEventListenerPanicIsContained(t *testing.T) {
	service := &Service{}
	service.SubscribeTaskEvents(TaskEventDone, func(string) { panic("callback failed") })
	service.emitTaskEvent(&download.Event{Key: download.EventKeyDone, Task: &download.Task{ID: "task-1"}})
}
