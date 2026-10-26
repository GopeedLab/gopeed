package download

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
)

type jobWaitContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (c *jobWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestManualProtocolJobRunsAfterJoinedScheduledSkip(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var manualCalls atomic.Int32
	manager := &testJobManager{job: fetcher.CronJob{Name: "refresh", Every: time.Hour, Timeout: time.Second,
		Run: func(ctx context.Context, source fetcher.TriggerSource) error {
			if source == fetcher.TriggerManual {
				manualCalls.Add(1)
				return nil
			}
			close(entered)
			select {
			case <-release:
				return fetcher.ErrJobSkipped
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}}
	d := testBackendDownloader(t, manager)
	scheduled := make(chan error, 1)
	go func() {
		scheduled <- d.protocolJobs.trigger(context.Background(), "test/refresh", fetcher.TriggerScheduled)
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	waiter := &jobWaitContext{Context: ctx, waiting: make(chan struct{})}
	manual := make(chan error, 1)
	go func() { manual <- d.TriggerProtocolJob(waiter, "test", "refresh") }()
	<-waiter.waiting
	close(release)
	if err := <-scheduled; !errors.Is(err, fetcher.ErrJobSkipped) {
		t.Fatal(err)
	}
	if err := <-manual; err != nil {
		t.Fatal(err)
	}
	if manualCalls.Load() != 1 {
		t.Fatal("manual refresh was swallowed by skipped automatic work")
	}
}

func TestManualProtocolJobFailureKeepsScheduledBackoff(t *testing.T) {
	var calls atomic.Int32
	failure := errors.New("offline")
	manager := &testJobManager{job: fetcher.CronJob{Name: "refresh", Every: time.Hour, Timeout: time.Second,
		Run: func(context.Context, fetcher.TriggerSource) error { calls.Add(1); return failure },
	}}
	d := testBackendDownloader(t, manager)
	if err := d.TriggerProtocolJob(context.Background(), "test", "refresh"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	entry := d.protocolJobs.jobs["test/refresh"]
	entry.mu.Lock()
	next := entry.next
	entry.mu.Unlock()
	if time.Until(next) < 59*time.Second {
		t.Fatal("manual failure did not schedule retry backoff")
	}
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("manual failure immediately retried %d times", calls.Load())
	}
}
