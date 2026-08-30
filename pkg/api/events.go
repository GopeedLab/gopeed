package api

import (
	"encoding/json"
	"sync"

	"github.com/GopeedLab/gopeed/pkg/download"
)

type TaskEventMask uint64

const (
	TaskEventDone TaskEventMask = 1 << iota
	TaskEventError
)

type TaskEvent struct {
	Type   string `json:"type"`
	TaskID string `json:"taskId"`
	Name   string `json:"name,omitempty"`
	Error  string `json:"error,omitempty"`
}

type TaskEventListener func(payload string)

type taskEventSubscription struct {
	mu       sync.RWMutex
	mask     TaskEventMask
	listener TaskEventListener
}

func (s *Service) SubscribeTaskEvents(mask TaskEventMask, listener TaskEventListener) {
	s.taskEvents.mu.Lock()
	defer s.taskEvents.mu.Unlock()
	s.taskEvents.mask = mask
	s.taskEvents.listener = listener
}

func (s *Service) emitTaskEvent(event *download.Event) {
	if event == nil || event.Task == nil {
		return
	}

	var mask TaskEventMask
	switch event.Key {
	case download.EventKeyDone:
		mask = TaskEventDone
	case download.EventKeyError:
		mask = TaskEventError
	default:
		return
	}

	s.taskEvents.mu.RLock()
	if s.taskEvents.mask&mask == 0 || s.taskEvents.listener == nil {
		s.taskEvents.mu.RUnlock()
		return
	}
	listener := s.taskEvents.listener
	s.taskEvents.mu.RUnlock()

	payload := TaskEvent{
		Type:   "task." + string(event.Key),
		TaskID: event.Task.ID,
	}
	if event.Task.Meta != nil {
		payload.Name = event.Task.Name()
	}
	if event.Err != nil {
		payload.Error = event.Err.Error()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// A platform callback must never be able to crash a download worker.
	func() {
		defer func() { _ = recover() }()
		listener(string(data))
	}()
}
