// Package nativebridge contains transport-neutral behavior shared by the
// Desktop C FFI and gomobile bridge adapters.
package nativebridge

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/GopeedLab/gopeed/pkg/rest"
)

const (
	minInvokeConcurrency = 4
	maxInvokeConcurrency = 32
)

type invokeRequest struct {
	method string
	path   string
	query  string
	body   string
}

type invokeTask struct {
	request  invokeRequest
	complete func(result string, err error)
}

// InvokeAsync queues an in-process API request and completes it through the
// supplied callback. Desktop and mobile adapters share this executor.
func InvokeAsync(method, path, query, body string, callback func(result string, err error)) {
	if callback == nil {
		return
	}
	task := invokeTask{
		request: invokeRequest{
			method: method,
			path:   path,
			query:  query,
			body:   body,
		},
		complete: callback,
	}
	if !getInvokeExecutor().submit(task) {
		completeCallback(callback, "", fmt.Errorf("native bridge is stopping"))
	}
}

// ResumeInvokes enables request submission after the native runtime starts.
func ResumeInvokes() {
	getInvokeExecutor().resume()
}

// PauseInvokesAndWait prevents new submissions and waits for every queued or
// running request to finish before a native runtime is stopped.
func PauseInvokesAndWait() {
	getInvokeExecutor().pauseAndWait()
}

var (
	invokeExecutorOnce sync.Once
	sharedInvoker      *invokeExecutor
)

func getInvokeExecutor() *invokeExecutor {
	invokeExecutorOnce.Do(func() {
		sharedInvoker = newDefaultInvokeExecutor(func(request invokeRequest) string {
			return rest.Dispatch(request.method, request.path, request.query, request.body)
		})
	})
	return sharedInvoker
}

// invokeExecutor keeps native API work concurrent without rejecting requests
// when every worker is busy. Additional requests wait in FIFO order and are
// completed through their callback.
type invokeExecutor struct {
	mu        sync.Mutex
	ready     *sync.Cond
	queue     []invokeTask
	closed    bool
	accepting bool
	pending   int
	workers   sync.WaitGroup
	handler   func(invokeRequest) string
}

func invokeConcurrency(cpuCount int) int {
	concurrency := cpuCount * 2
	if concurrency < minInvokeConcurrency {
		return minInvokeConcurrency
	}
	if concurrency > maxInvokeConcurrency {
		return maxInvokeConcurrency
	}
	return concurrency
}

func newInvokeExecutor(workerCount int, handler func(invokeRequest) string) *invokeExecutor {
	if workerCount < 1 {
		workerCount = 1
	}
	executor := &invokeExecutor{handler: handler, accepting: true}
	executor.ready = sync.NewCond(&executor.mu)
	executor.workers.Add(workerCount)
	for range workerCount {
		go executor.run()
	}
	return executor
}

func newDefaultInvokeExecutor(handler func(invokeRequest) string) *invokeExecutor {
	return newInvokeExecutor(invokeConcurrency(runtime.GOMAXPROCS(0)), handler)
}

func (e *invokeExecutor) submit(task invokeTask) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed || !e.accepting {
		return false
	}
	e.queue = append(e.queue, task)
	e.pending++
	e.ready.Signal()
	return true
}

func (e *invokeExecutor) resume() {
	e.mu.Lock()
	e.accepting = true
	e.mu.Unlock()
}

func (e *invokeExecutor) pauseAndWait() {
	e.mu.Lock()
	e.accepting = false
	for e.pending > 0 {
		e.ready.Wait()
	}
	e.mu.Unlock()
}

func (e *invokeExecutor) close() {
	e.mu.Lock()
	e.closed = true
	e.ready.Broadcast()
	e.mu.Unlock()
	e.workers.Wait()
}

func (e *invokeExecutor) run() {
	defer e.workers.Done()
	for {
		task, ok := e.take()
		if !ok {
			return
		}
		result, err := e.execute(task.request)
		completeCallback(task.complete, result, err)
		e.mu.Lock()
		e.pending--
		if e.pending == 0 {
			e.ready.Broadcast()
		}
		e.mu.Unlock()
	}
}

func (e *invokeExecutor) take() (invokeTask, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for len(e.queue) == 0 && !e.closed {
		e.ready.Wait()
	}
	if len(e.queue) == 0 {
		return invokeTask{}, false
	}
	task := e.queue[0]
	e.queue[0] = invokeTask{}
	e.queue = e.queue[1:]
	if len(e.queue) == 0 {
		e.queue = nil
	}
	return task, true
}

func (e *invokeExecutor) execute(request invokeRequest) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("invoke panic: %v", recovered)
		}
	}()
	return e.handler(request), nil
}

func completeCallback(callback func(string, error), result string, err error) {
	defer func() {
		_ = recover()
	}()
	callback(result, err)
}
