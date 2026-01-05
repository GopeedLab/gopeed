package download

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewExtractionQueue(t *testing.T) {
	q := NewExtractionQueue()
	if q == nil {
		t.Fatal("NewExtractionQueue returned nil")
	}
	if q.jobs == nil {
		t.Fatal("jobs slice should be initialized")
	}
	if len(q.jobs) != 0 {
		t.Fatalf("expected empty jobs slice, got %d", len(q.jobs))
	}
	if q.running {
		t.Fatal("queue should not be running initially")
	}
	if q.cond == nil {
		t.Fatal("condition variable should be initialized")
	}
}

func TestExtractionQueue_StartStop(t *testing.T) {
	q := NewExtractionQueue()

	// Test Start
	q.Start()
	if !q.IsRunning() {
		t.Fatal("queue should be running after Start")
	}

	// Test double Start (should be idempotent)
	q.Start()
	if !q.IsRunning() {
		t.Fatal("queue should still be running after second Start")
	}

	// Test Stop
	q.Stop()
	if q.IsRunning() {
		t.Fatal("queue should not be running after Stop")
	}

	// Test double Stop (should be idempotent)
	q.Stop()
	if q.IsRunning() {
		t.Fatal("queue should still not be running after second Stop")
	}
}

func TestExtractionQueue_EnqueueSingleJob(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	defer q.Stop()

	executed := false
	job := NewExtractionJob("test-1", func() {
		executed = true
	})

	q.Enqueue(job)
	job.Wait()

	if !executed {
		t.Fatal("job should have been executed")
	}
}

func TestExtractionQueue_EnqueueAndWait(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	defer q.Stop()

	executed := false
	job := NewExtractionJob("test-1", func() {
		executed = true
	})

	q.EnqueueAndWait(job)

	if !executed {
		t.Fatal("job should have been executed after EnqueueAndWait returns")
	}
}

func TestExtractionQueue_FIFOOrder(t *testing.T) {
	q := NewExtractionQueue()
	// Don't start yet - we want to queue multiple jobs first

	var executionOrder []string
	var mu sync.Mutex

	addToOrder := func(id string) {
		mu.Lock()
		executionOrder = append(executionOrder, id)
		mu.Unlock()
	}

	job1 := NewExtractionJob("job-1", func() {
		addToOrder("job-1")
	})
	job2 := NewExtractionJob("job-2", func() {
		addToOrder("job-2")
	})
	job3 := NewExtractionJob("job-3", func() {
		addToOrder("job-3")
	})

	// Enqueue jobs before starting (they'll wait in queue)
	q.mu.Lock()
	q.jobs = append(q.jobs, job1, job2, job3)
	q.mu.Unlock()

	// Now start processing
	q.Start()
	defer q.Stop()

	// Wait for all jobs
	job1.Wait()
	job2.Wait()
	job3.Wait()

	// Verify FIFO order
	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executionOrder))
	}
	if executionOrder[0] != "job-1" || executionOrder[1] != "job-2" || executionOrder[2] != "job-3" {
		t.Fatalf("expected FIFO order [job-1, job-2, job-3], got %v", executionOrder)
	}
}

func TestExtractionQueue_SequentialExecution(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	defer q.Stop()

	var activeJobs int32
	var maxConcurrent int32
	var mu sync.Mutex

	// Create jobs that take some time to execute
	createJob := func(id string) *ExtractionJob {
		return NewExtractionJob(id, func() {
			current := atomic.AddInt32(&activeJobs, 1)
			mu.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&activeJobs, -1)
		})
	}

	jobs := make([]*ExtractionJob, 5)
	for i := range jobs {
		jobs[i] = q.Enqueue(createJob(string(rune('A' + i))))
	}

	// Wait for all jobs
	for _, job := range jobs {
		job.Wait()
	}

	// Verify only one job ran at a time
	if maxConcurrent > 1 {
		t.Fatalf("expected max 1 concurrent job, got %d", maxConcurrent)
	}
}

func TestExtractionQueue_QueueLength(t *testing.T) {
	q := NewExtractionQueue()

	if q.QueueLength() != 0 {
		t.Fatalf("expected queue length 0, got %d", q.QueueLength())
	}

	// Add jobs without starting (they'll stay in queue)
	blockChan := make(chan struct{})
	blockingJob := NewExtractionJob("blocking", func() {
		<-blockChan // Block until signaled
	})

	q.Start()
	defer q.Stop()

	q.Enqueue(blockingJob)

	// Give worker time to pick up the blocking job
	time.Sleep(20 * time.Millisecond)

	// Queue more jobs while blocking job is running
	job2 := q.Enqueue(NewExtractionJob("job-2", func() {}))
	job3 := q.Enqueue(NewExtractionJob("job-3", func() {}))

	// Should have 2 jobs waiting
	if q.QueueLength() != 2 {
		t.Fatalf("expected queue length 2, got %d", q.QueueLength())
	}

	// Unblock and wait
	close(blockChan)
	blockingJob.Wait()
	job2.Wait()
	job3.Wait()

	if q.QueueLength() != 0 {
		t.Fatalf("expected queue length 0 after completion, got %d", q.QueueLength())
	}
}

func TestExtractionQueue_HasPendingJob(t *testing.T) {
	q := NewExtractionQueue()

	blockChan := make(chan struct{})
	blockingJob := NewExtractionJob("blocking", func() {
		<-blockChan
	})

	q.Start()
	defer q.Stop()

	q.Enqueue(blockingJob)
	time.Sleep(20 * time.Millisecond) // Let blocking job start

	// Queue more jobs
	job2 := q.Enqueue(NewExtractionJob("job-2", func() {}))

	if !q.HasPendingJob("job-2") {
		t.Fatal("expected HasPendingJob to return true for job-2")
	}

	if q.HasPendingJob("job-99") {
		t.Fatal("expected HasPendingJob to return false for non-existent job")
	}

	// Unblock and wait
	close(blockChan)
	blockingJob.Wait()
	job2.Wait()

	if q.HasPendingJob("job-2") {
		t.Fatal("expected HasPendingJob to return false after job completion")
	}
}

func TestExtractionQueue_RemovePendingJob(t *testing.T) {
	q := NewExtractionQueue()

	blockChan := make(chan struct{})
	blockingJob := NewExtractionJob("blocking", func() {
		<-blockChan
	})

	executed := false
	job2 := NewExtractionJob("job-2", func() {
		executed = true
	})

	q.Start()
	defer q.Stop()

	q.Enqueue(blockingJob)
	time.Sleep(20 * time.Millisecond) // Let blocking job start
	q.Enqueue(job2)

	// Remove job-2 before it executes
	removed := q.RemovePendingJob("job-2")
	if !removed {
		t.Fatal("expected RemovePendingJob to return true")
	}

	// Try to remove again (should return false)
	removed = q.RemovePendingJob("job-2")
	if removed {
		t.Fatal("expected second RemovePendingJob to return false")
	}

	// Unblock
	close(blockChan)
	blockingJob.Wait()

	// Wait a bit for any potential execution
	time.Sleep(50 * time.Millisecond)

	if executed {
		t.Fatal("removed job should not have been executed")
	}

	// The removed job's done channel should be closed
	select {
	case <-job2.done:
		// Expected
	default:
		t.Fatal("removed job's done channel should be closed")
	}
}

func TestExtractionQueue_StopDiscardsPendingJobs(t *testing.T) {
	q := NewExtractionQueue()

	blockChan := make(chan struct{})
	executed1 := false
	executed2 := false

	blockingJob := NewExtractionJob("blocking", func() {
		<-blockChan
		executed1 = true
	})
	pendingJob := NewExtractionJob("pending", func() {
		executed2 = true
	})

	q.Start()

	q.Enqueue(blockingJob)
	time.Sleep(20 * time.Millisecond) // Let blocking job start
	q.Enqueue(pendingJob)

	// Stop without unblocking
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(blockChan)
	}()
	q.Stop()

	// Blocking job should have completed, but pending job was discarded
	if !executed1 {
		t.Log("Note: blocking job may not complete if Stop() won races")
	}
	if executed2 {
		t.Fatal("pending job should not have been executed after Stop")
	}

	// Pending job's done channel should be closed
	select {
	case <-pendingJob.done:
		// Expected
	default:
		t.Fatal("pending job's done channel should be closed after Stop")
	}
}

func TestExtractionQueue_EnqueueAfterShutdown(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	q.Stop()

	executed := false
	job := NewExtractionJob("after-shutdown", func() {
		executed = true
	})

	q.Enqueue(job)

	// Job should be immediately done (without execution)
	select {
	case <-job.done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("job should be immediately done after shutdown")
	}

	if executed {
		t.Fatal("job should not be executed after shutdown")
	}
}

func TestNewExtractionJob(t *testing.T) {
	job := NewExtractionJob("test-id", func() {})

	if job.ID != "test-id" {
		t.Fatalf("expected ID 'test-id', got '%s'", job.ID)
	}
	if job.Execute == nil {
		t.Fatal("Execute should not be nil")
	}
	if job.done == nil {
		t.Fatal("done channel should be initialized")
	}
}

func TestExtractionJob_Wait(t *testing.T) {
	job := NewExtractionJob("test", func() {})

	done := make(chan struct{})
	go func() {
		job.Wait()
		close(done)
	}()

	// Wait should block
	select {
	case <-done:
		t.Fatal("Wait should block until job is done")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}

	// Close done channel
	close(job.done)

	// Now Wait should return
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Wait should return after job is done")
	}
}

func TestExtractionQueue_NilExecuteFunction(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	defer q.Stop()

	// Job with nil Execute should not panic
	job := &ExtractionJob{
		ID:      "nil-execute",
		Execute: nil,
		done:    make(chan struct{}),
	}

	q.Enqueue(job)
	job.Wait()
	// No panic means success
}

func TestExtractionQueue_ConcurrentEnqueue(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	defer q.Stop()

	var counter int32
	var wg sync.WaitGroup
	numGoroutines := 10
	jobsPerGoroutine := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				job := NewExtractionJob("", func() {
					atomic.AddInt32(&counter, 1)
				})
				q.EnqueueAndWait(job)
			}
		}(i)
	}

	wg.Wait()

	expected := int32(numGoroutines * jobsPerGoroutine)
	if counter != expected {
		t.Fatalf("expected counter %d, got %d", expected, counter)
	}
}

func TestExtractionQueue_Restart(t *testing.T) {
	q := NewExtractionQueue()

	// First run
	q.Start()
	executed1 := false
	job1 := NewExtractionJob("job-1", func() {
		executed1 = true
	})
	q.EnqueueAndWait(job1)
	q.Stop()

	if !executed1 {
		t.Fatal("first job should have been executed")
	}

	// Restart
	q.Start()
	executed2 := false
	job2 := NewExtractionJob("job-2", func() {
		executed2 = true
	})
	q.EnqueueAndWait(job2)
	q.Stop()

	if !executed2 {
		t.Fatal("second job should have been executed after restart")
	}
}

func TestGetExtractionQueue(t *testing.T) {
	// Note: This test modifies global state, so it should ideally be run in isolation
	// However, the implementation ensures the queue is created only once

	queue := GetExtractionQueue()
	if queue == nil {
		t.Fatal("GetExtractionQueue should not return nil")
	}
	if !queue.IsRunning() {
		t.Fatal("global queue should be running")
	}

	// Calling again should return the same instance
	queue2 := GetExtractionQueue()
	if queue != queue2 {
		t.Fatal("GetExtractionQueue should return the same instance")
	}
}

func TestExtractionQueue_LongRunningJob(t *testing.T) {
	q := NewExtractionQueue()
	q.Start()
	defer q.Stop()

	startTime := time.Now()
	longJobDuration := 100 * time.Millisecond

	longJob := NewExtractionJob("long-job", func() {
		time.Sleep(longJobDuration)
	})

	shortJobExecuted := false
	shortJob := NewExtractionJob("short-job", func() {
		shortJobExecuted = true
	})

	q.Enqueue(longJob)
	q.Enqueue(shortJob)

	shortJob.Wait()

	elapsed := time.Since(startTime)
	if elapsed < longJobDuration {
		t.Fatalf("short job should wait for long job, elapsed: %v", elapsed)
	}

	if !shortJobExecuted {
		t.Fatal("short job should have been executed")
	}
}
