// Package nativebridge contains transport-neutral behavior shared by the
// Desktop C FFI and gomobile bridge adapters.
package nativebridge

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GopeedLab/gopeed/pkg/rest"
)

// StopTimeout bounds native shutdown even when an in-process request never returns.
const StopTimeout = 3 * time.Second

var errNativeBridgeStopping = errors.New("native bridge is stopping")

type invokeRequest struct {
	method string
	path   string
	query  string
	body   string
}

type invokeTask struct {
	id       uint64
	request  invokeRequest
	complete func(result string, err error)
	once     sync.Once
}

func (t *invokeTask) finish(result string, err error) {
	t.once.Do(func() {
		completeCallback(t.complete, result, err)
	})
}

// InvokeAsync starts an in-process API request and completes it through the
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
		completeCallback(callback, "", errNativeBridgeStopping)
	}
}

// ResumeInvokes enables request submission after the native runtime starts.
func ResumeInvokes() {
	getInvokeExecutor().resume()
}

// PauseInvokesAndWait prevents new submissions and gives running requests a
// bounded opportunity to finish before a native runtime is stopped.
// Requests still pending after StopTimeout are failed and detached so shutdown
// can continue without invoking their callbacks a second time.
func PauseInvokesAndWait() bool {
	return getInvokeExecutor().pauseAndWait(StopTimeout)
}

// Stop pauses native requests and shuts down the shared runtime, but never
// keeps the host application waiting longer than StopTimeout. Cleanup that
// cannot finish within the deadline is allowed to continue in the background.
func Stop() {
	stopWithin(getInvokeExecutor(), rest.Stop, StopTimeout)
}

func stopWithin(executor *invokeExecutor, cleanup func(), timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	executor.pauseAndWait(time.Until(deadline))

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		cleanup()
	}()

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-stopped:
	case <-timer.C:
	}
}

var (
	invokeExecutorOnce sync.Once
	sharedInvoker      *invokeExecutor
)

func getInvokeExecutor() *invokeExecutor {
	invokeExecutorOnce.Do(func() {
		sharedInvoker = newInvokeExecutor(func(request invokeRequest) string {
			return rest.Dispatch(request.method, request.path, request.query, request.body)
		})
	})
	return sharedInvoker
}

// invokeExecutor starts every native API request in its own goroutine, matching
// net/http's request concurrency. It only tracks them to support bounded drain
// during shutdown; one stuck request must never prevent another from starting.
type invokeExecutor struct {
	mu        sync.Mutex
	closed    bool
	accepting bool
	nextID    uint64
	pending   map[uint64]*invokeTask
	drained   chan struct{}
	running   sync.WaitGroup
	handler   func(invokeRequest) string
}

func newInvokeExecutor(handler func(invokeRequest) string) *invokeExecutor {
	drained := make(chan struct{})
	close(drained)
	return &invokeExecutor{
		handler:   handler,
		accepting: true,
		pending:   make(map[uint64]*invokeTask),
		drained:   drained,
	}
}

func (e *invokeExecutor) submit(task invokeTask) bool {
	e.mu.Lock()
	if e.closed || !e.accepting {
		e.mu.Unlock()
		return false
	}
	if len(e.pending) == 0 {
		e.drained = make(chan struct{})
	}
	taskID := e.nextID
	e.nextID++
	task.id = taskID
	taskPtr := &task
	e.pending[taskID] = taskPtr
	e.running.Add(1)
	e.mu.Unlock()

	go e.run(taskPtr)
	return true
}

func (e *invokeExecutor) resume() {
	e.mu.Lock()
	e.accepting = true
	e.mu.Unlock()
}

func (e *invokeExecutor) pauseAndWait(timeout time.Duration) bool {
	e.mu.Lock()
	e.accepting = false
	if len(e.pending) == 0 {
		e.mu.Unlock()
		return true
	}
	drained := e.drained
	e.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-drained:
		return true
	case <-timer.C:
	}

	e.mu.Lock()
	if len(e.pending) == 0 {
		e.mu.Unlock()
		return true
	}
	pending := make([]*invokeTask, 0, len(e.pending))
	for _, task := range e.pending {
		pending = append(pending, task)
	}
	e.pending = make(map[uint64]*invokeTask)
	close(e.drained)
	e.mu.Unlock()

	for _, task := range pending {
		task.finish("", errNativeBridgeStopping)
	}
	return false
}

func (e *invokeExecutor) close() {
	e.mu.Lock()
	e.closed = true
	e.accepting = false
	e.mu.Unlock()
	e.running.Wait()
}

func (e *invokeExecutor) run(task *invokeTask) {
	defer e.running.Done()
	result, err := e.execute(task.request)
	task.finish(result, err)
	e.mu.Lock()
	if _, exists := e.pending[task.id]; exists {
		delete(e.pending, task.id)
		if len(e.pending) == 0 {
			close(e.drained)
		}
	}
	e.mu.Unlock()
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
