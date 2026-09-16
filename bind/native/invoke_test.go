package nativebridge

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestInvokeExecutorStartsEveryRequestIndependently(t *testing.T) {
	const taskCount = 64
	release := make(chan struct{})
	started := make(chan struct{}, taskCount)
	executor := newInvokeExecutor(func(request invokeRequest) string {
		started <- struct{}{}
		<-release
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
			t.Fatalf("request %d was rejected", i)
		}
	}

	// Every handler must start before any of them is released. A worker pool
	// would stall here once all of its workers were occupied.
	for range taskCount {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("a request waited behind other blocked requests")
		}
	}

	close(release)
	completed.Wait()
	executor.close()
}

func TestInvokeExecutorConvertsPanicsToErrors(t *testing.T) {
	executor := newInvokeExecutor(func(invokeRequest) string {
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
	executor := newInvokeExecutor(func(request invokeRequest) string {
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
	executor := newInvokeExecutor(func(request invokeRequest) string {
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
		executor.pauseAndWait(time.Second)
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

func TestInvokeExecutorPauseTimesOutAndFailsPendingRequests(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	executor := newInvokeExecutor(func(request invokeRequest) string {
		started <- struct{}{}
		<-release
		return request.path
	})

	errors := make(chan error, 2)
	for _, path := range []string{"first", "second"} {
		if !executor.submit(invokeTask{
			request: invokeRequest{path: path},
			complete: func(_ string, err error) {
				errors <- err
			},
		}) {
			t.Fatalf("request %q was rejected", path)
		}
	}
	for range 2 {
		<-started
	}

	const timeout = 30 * time.Millisecond
	startedAt := time.Now()
	if executor.pauseAndWait(timeout) {
		t.Fatal("pause unexpectedly reported a complete drain")
	}
	if elapsed := time.Since(startedAt); elapsed < timeout || elapsed > 10*timeout {
		t.Fatalf("pause returned after %s, want close to %s", elapsed, timeout)
	}
	for range 2 {
		select {
		case err := <-errors:
			if err == nil || err.Error() != errNativeBridgeStopping.Error() {
				t.Fatalf("unexpected timeout error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("pending request was not failed after the timeout")
		}
	}

	close(release)
	executor.close()
	select {
	case err := <-errors:
		t.Fatalf("request callback ran more than once: %v", err)
	default:
	}
}

func TestStopWithinUsesOneDeadlineForRequestsAndCleanup(t *testing.T) {
	releaseRequest := make(chan struct{})
	requestStarted := make(chan struct{})
	executor := newInvokeExecutor(func(invokeRequest) string {
		close(requestStarted)
		<-releaseRequest
		return "done"
	})
	requestResult := make(chan error, 1)
	executor.submit(invokeTask{complete: func(_ string, err error) {
		requestResult <- err
	}})
	<-requestStarted

	releaseCleanup := make(chan struct{})
	cleanupStarted := make(chan struct{})
	cleanup := func() {
		close(cleanupStarted)
		<-releaseCleanup
	}

	const timeout = 40 * time.Millisecond
	startedAt := time.Now()
	stopWithin(executor, cleanup, timeout)
	if elapsed := time.Since(startedAt); elapsed < timeout || elapsed > 10*timeout {
		t.Fatalf("stop returned after %s, want close to %s", elapsed, timeout)
	}
	select {
	case <-cleanupStarted:
	case <-time.After(time.Second):
		t.Fatal("cleanup was not started after the request deadline")
	}
	select {
	case err := <-requestResult:
		if err == nil || err.Error() != errNativeBridgeStopping.Error() {
			t.Fatalf("unexpected request result: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed-out request was not failed")
	}

	close(releaseRequest)
	close(releaseCleanup)
	executor.close()
}
