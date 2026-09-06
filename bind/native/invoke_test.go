package nativebridge

import (
	"runtime"
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

func TestInvokeExecutorSurvivesCallbackPanic(t *testing.T) {
	executor := newInvokeExecutor(1, func(request invokeRequest) string {
		return request.path
	})
	executor.submit(invokeTask{complete: func(string, error) {
		panic("callback failed")
	}})

	completed := make(chan string, 1)
	executor.submit(invokeTask{
		request: invokeRequest{path: "next"},
		complete: func(result string, _ error) {
			completed <- result
		},
	})
	select {
	case result := <-completed:
		if result != "next" {
			t.Fatalf("unexpected result: %q", result)
		}
	case <-time.After(time.Second):
		t.Fatal("worker stopped after callback panic")
	}
	executor.close()
}

func TestInvokeExecutorPausesAndDrains(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	executor := newInvokeExecutor(1, func(request invokeRequest) string {
		if request.path == "running" {
			close(started)
			<-release
		}
		return request.path
	})
	completed := make(chan struct{})
	if !executor.submit(invokeTask{
		request: invokeRequest{path: "running"},
		complete: func(string, error) {
			close(completed)
		},
	}) {
		t.Fatal("initial task was rejected")
	}
	<-started

	drained := make(chan struct{})
	go func() {
		executor.pauseAndWait()
		close(drained)
	}()
	deadline := time.Now().Add(time.Second)
	for {
		executor.mu.Lock()
		paused := !executor.accepting
		executor.mu.Unlock()
		if paused {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("executor did not pause")
		}
		runtime.Gosched()
	}
	select {
	case <-drained:
		t.Fatal("pause returned before the running request completed")
	default:
	}
	if executor.submit(invokeTask{}) {
		t.Fatal("paused executor accepted a new request")
	}

	close(release)
	<-completed
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("pause did not return after pending requests drained")
	}

	executor.resume()
	resumed := make(chan struct{})
	if !executor.submit(invokeTask{complete: func(string, error) {
		close(resumed)
	}}) {
		t.Fatal("resumed executor rejected a request")
	}
	<-resumed
	executor.close()
}
