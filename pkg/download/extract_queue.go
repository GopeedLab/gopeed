package download

import (
	"sync"
)

// ExtractionJob represents a single extraction job in the queue
type ExtractionJob struct {
	// ID is a unique identifier for this job (usually task ID or multi-part base name)
	ID string
	// Execute is the function that performs the actual extraction
	Execute func()
	// done channel is signaled when the job has been executed
	done chan struct{}
}

// NewExtractionJob creates a new extraction job
func NewExtractionJob(id string, execute func()) *ExtractionJob {
	return &ExtractionJob{
		ID:      id,
		Execute: execute,
		done:    make(chan struct{}),
	}
}

// Wait blocks until the job has been executed
func (j *ExtractionJob) Wait() {
	<-j.done
}

// ExtractionQueue manages a queue of extraction jobs to prevent resource exhaustion
// by ensuring only one extraction (or one multi-part archive extraction) runs at a time
type ExtractionQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	jobs     []*ExtractionJob
	running  bool
	shutdown bool
	wg       sync.WaitGroup
}

// NewExtractionQueue creates a new extraction queue
func NewExtractionQueue() *ExtractionQueue {
	q := &ExtractionQueue{
		jobs: make([]*ExtractionJob, 0),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Start starts the queue worker that processes jobs sequentially
func (q *ExtractionQueue) Start() {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.shutdown = false
	q.mu.Unlock()

	q.wg.Add(1)
	go q.worker()
}

// Stop stops the queue worker gracefully
// It waits for the current job to complete but discards pending jobs
func (q *ExtractionQueue) Stop() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.shutdown = true
	q.cond.Signal()
	q.mu.Unlock()

	q.wg.Wait()
}

// Enqueue adds a new extraction job to the queue
// The job will be executed when its turn comes (FIFO order)
// Returns the job so the caller can wait for completion if needed
func (q *ExtractionQueue) Enqueue(job *ExtractionJob) *ExtractionJob {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.shutdown {
		// Queue is shutting down, signal done immediately without executing
		close(job.done)
		return job
	}

	q.jobs = append(q.jobs, job)
	q.cond.Signal()
	return job
}

// EnqueueAndWait adds a new extraction job to the queue and waits for it to complete
func (q *ExtractionQueue) EnqueueAndWait(job *ExtractionJob) {
	q.Enqueue(job)
	job.Wait()
}

// QueueLength returns the current number of pending jobs in the queue
func (q *ExtractionQueue) QueueLength() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

// IsRunning returns true if the queue worker is running
func (q *ExtractionQueue) IsRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running
}

// HasPendingJob checks if there's a pending job with the given ID
func (q *ExtractionQueue) HasPendingJob(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, job := range q.jobs {
		if job.ID == id {
			return true
		}
	}
	return false
}

// RemovePendingJob removes a pending job with the given ID from the queue
// Returns true if a job was removed, false if not found
// Note: This cannot remove a job that is currently being executed
func (q *ExtractionQueue) RemovePendingJob(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, job := range q.jobs {
		if job.ID == id {
			// Close the done channel to unblock any waiters
			close(job.done)
			// Remove from queue
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			return true
		}
	}
	return false
}

// worker is the main loop that processes jobs sequentially
func (q *ExtractionQueue) worker() {
	defer q.wg.Done()

	for {
		q.mu.Lock()
		// Wait for a job or shutdown signal
		for len(q.jobs) == 0 && !q.shutdown {
			q.cond.Wait()
		}

		if q.shutdown {
			// Close done channels for all remaining jobs
			for _, job := range q.jobs {
				close(job.done)
			}
			q.jobs = nil
			q.running = false
			q.mu.Unlock()
			return
		}

		// Dequeue the first job
		job := q.jobs[0]
		q.jobs = q.jobs[1:]
		q.mu.Unlock()

		// Execute the job outside the lock
		if job.Execute != nil {
			job.Execute()
		}

		// Signal that the job is done
		close(job.done)
	}
}

// Global extraction queue instance
var globalExtractionQueue *ExtractionQueue
var extractionQueueOnce sync.Once

// GetExtractionQueue returns the global extraction queue instance
func GetExtractionQueue() *ExtractionQueue {
	extractionQueueOnce.Do(func() {
		globalExtractionQueue = NewExtractionQueue()
		globalExtractionQueue.Start()
	})
	return globalExtractionQueue
}
