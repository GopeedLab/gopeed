package main

import (
	"fmt"
	"runtime"
	"sync"
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

// invokeExecutor keeps native API work concurrent without ever rejecting a
// request because all workers are busy. submit only appends to the waiting
// queue; callers observe backpressure through the Future completed by the
// callback rather than by blocking the FFI entry point.
type invokeExecutor struct {
	mu      sync.Mutex
	ready   *sync.Cond
	queue   []invokeTask
	closed  bool
	workers sync.WaitGroup
	handler func(invokeRequest) string
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
	executor := &invokeExecutor{handler: handler}
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
	if e.closed {
		return false
	}
	e.queue = append(e.queue, task)
	e.ready.Signal()
	return true
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
		task.complete(result, err)
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
