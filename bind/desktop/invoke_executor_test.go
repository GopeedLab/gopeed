package main

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInvokeConcurrency(t *testing.T) {
	tests := []struct {
		cpus int
		want int
	}{
		{cpus: 1, want: 4},
		{cpus: 2, want: 4},
		{cpus: 4, want: 8},
		{cpus: 8, want: 16},
		{cpus: 16, want: 32},
		{cpus: 64, want: 32},
	}
	for _, test := range tests {
		t.Run(strconv.Itoa(test.cpus), func(t *testing.T) {
			if got := invokeConcurrency(test.cpus); got != test.want {
				t.Fatalf("invokeConcurrency(%d) = %d, want %d", test.cpus, got, test.want)
			}
		})
	}
}

func TestInvokeExecutorWaitsWithoutRejecting(t *testing.T) {
	const (
		workerCount = 2
		taskCount   = 6
	)
	release := make(chan struct{})
	started := make(chan struct{}, taskCount)
	var active atomic.Int32
	var maxActive atomic.Int32
	executor := newInvokeExecutor(workerCount, func(request invokeRequest) string {
		current := active.Add(1)
		for {
			maximum := maxActive.Load()
			if current <= maximum || maxActive.CompareAndSwap(maximum, current) {
				break
			}
		}
		started <- struct{}{}
		<-release
		active.Add(-1)
		return request.path
	})

	var completed sync.WaitGroup
	completed.Add(taskCount)
	for i := range taskCount {
		accepted := executor.submit(invokeTask{
			request: invokeRequest{path: strconv.Itoa(i)},
			complete: func(string, error) {
				completed.Done()
			},
		})
		if !accepted {
			t.Fatalf("task %d was rejected while all workers were busy", i)
		}
	}

	for range workerCount {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not start queued tasks")
		}
	}
	if got := maxActive.Load(); got != workerCount {
		t.Fatalf("maximum active tasks = %d, want %d", got, workerCount)
	}

	close(release)
	completed.Wait()
	executor.close()
}

func TestInvokeExecutorConvertsPanicsToErrors(t *testing.T) {
	executor := newInvokeExecutor(1, func(invokeRequest) string {
		panic("boom")
	})
	completed := make(chan error, 1)
	executor.submit(invokeTask{complete: func(_ string, err error) {
		completed <- err
	}})
	select {
	case err := <-completed:
		if err == nil || err.Error() != "invoke panic: boom" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("panic result was not completed")
	}
	executor.close()
}
