package http

import (
	"context"
	"sync"
	"time"
)

const (
	defaultMaxRetries           = 5
	defaultMaxRetryDelaySeconds = 5
)

// retryPolicy decides when the failed requests of one download attempt are
// tried again. It looks at the whole task instead of the status code of each
// failure:
//
//   - While at least one request of the task is healthy (it got a valid 200 or
//     206 and is still downloading), a failed request is retried without using
//     the task's retry budget, after a delay that grows with its own
//     consecutive failures.
//   - Once every unfinished request has failed, the task gets a limited number
//     of retry rounds. A round gives every failed request one more try after
//     the round's delay; it counts once for the task, however many requests
//     fail in it. Requests still waiting for response headers, or waiting to be
//     sent, are let through first rather than counted as failures.
//   - Any valid response resets the task: the rounds start again from the
//     first one, with the shortest delay.
//
// When the rounds are used up the policy is exhausted and every request gives
// up, so the task fails. A policy serves one download attempt; a new attempt
// (Start after a failure or a pause) gets a new policy and so a full budget.
type retryPolicy struct {
	// maxRounds is the number of retry rounds once every request has failed;
	// negative means unlimited.
	maxRounds int
	// unit is the delay of the first retry; the n-th waits n units.
	unit time.Duration
	// maxDelay caps every delay.
	maxDelay time.Duration
	// streaming, when set, reports a healthy download outside the tracked
	// requests: the Resolve response still being read in the background.
	streaming func() bool

	mu        sync.Mutex
	members   map[*connection]*retryMember
	rounds    int  // retry rounds started since every request last failed
	exhausted bool // the rounds are used up; the attempt has failed
	changed   chan struct{}
}

type retryState int

const (
	// retryPending: a request is about to be sent, or waits for its response
	// headers. It has not failed, but it is not healthy yet either.
	retryPending retryState = iota
	// retryHealthy: the last request got a valid 200 or 206 and is downloading.
	retryHealthy
	// retryFailed: the last request failed; it waits for a decision.
	retryFailed
)

type retryMember struct {
	state retryState
	// failures counts the failed requests since the last valid response.
	failures int
	// delay is the wait before the next try, set when the member is let go.
	delay time.Duration
}

func newRetryPolicy(maxRounds int, unit, maxDelay time.Duration, streaming func() bool) *retryPolicy {
	if unit <= 0 {
		unit = time.Second
	}
	if maxDelay < unit {
		maxDelay = unit
	}
	return &retryPolicy{
		maxRounds: maxRounds,
		unit:      unit,
		maxDelay:  maxDelay,
		streaming: streaming,
		members:   make(map[*connection]*retryMember),
		changed:   make(chan struct{}),
	}
}

// delayFor returns the wait before the n-th retry: n units, capped.
func (p *retryPolicy) delayFor(n int) time.Duration {
	if n < 1 {
		n = 1
	}
	if limit := int64(p.maxDelay / p.unit); int64(n) > limit {
		return p.maxDelay
	}
	return min(time.Duration(n)*p.unit, p.maxDelay)
}

func (p *retryPolicy) notifyLocked() {
	close(p.changed)
	p.changed = make(chan struct{})
}

// join adds a request of the task; it starts out pending.
func (p *retryPolicy) join(c *connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.members[c]; ok {
		m.state = retryPending
	} else {
		p.members[c] = &retryMember{state: retryPending}
	}
	p.notifyLocked()
}

// leave removes a request that is done: its chunk is complete, it gave up, or
// it was cancelled. A completed request no longer counts as healthy.
func (p *retryPolicy) leave(c *connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.members[c]; !ok {
		return
	}
	delete(p.members, c)
	p.notifyLocked()
}

// attempt marks c as sending a new request, for example after it finished its
// chunk and took over part of another one.
func (p *retryPolicy) attempt(c *connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.members[c]; ok && m.state != retryPending {
		m.state = retryPending
		p.notifyLocked()
	}
}

// success records a valid 200 or 206 for c. It resets the budget of the whole
// task: the next time every request fails, the rounds start from the first.
func (p *retryPolicy) success(c *connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m, ok := p.members[c]
	if !ok {
		m = &retryMember{}
		p.members[c] = m
	}
	m.state = retryHealthy
	m.failures = 0
	if p.rounds > 0 && !p.exhausted {
		// Leave the limited phase: every request gets the full allowance
		// and the shortest delay again.
		p.rounds = 0
		for _, other := range p.members {
			other.failures = 0
		}
	}
	p.notifyLocked()
}

// next records that the last request of c failed and decides whether and when
// it is tried again. It blocks while no request is healthy but some are still
// pending, until those succeed or fail. It returns the delay to wait before
// the next request, or false when the task's retries are used up or ctx ends.
func (p *retryPolicy) next(ctx context.Context, c *connection) (time.Duration, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m, ok := p.members[c]
	if !ok {
		m = &retryMember{}
		p.members[c] = m
	}
	m.state = retryFailed
	m.failures++
	p.notifyLocked()

	for {
		if p.exhausted {
			return 0, false
		}
		if m.state == retryPending {
			// Let go by a round another request started.
			return m.delay, true
		}
		if p.anyHealthyLocked() {
			m.state = retryPending
			m.delay = p.delayFor(m.failures)
			p.notifyLocked()
			return m.delay, true
		}
		if p.anyLocked(retryPending) {
			changed := p.changed
			p.mu.Unlock()
			select {
			case <-changed:
			case <-ctx.Done():
			}
			p.mu.Lock()
			if ctx.Err() != nil {
				return 0, false
			}
			continue
		}

		// Every unfinished request has failed.
		if p.maxRounds >= 0 && p.rounds >= p.maxRounds {
			p.exhausted = true
			p.notifyLocked()
			return 0, false
		}
		p.rounds++
		delay := p.delayFor(p.rounds)
		for _, other := range p.members {
			if other.state == retryFailed {
				other.state = retryPending
				other.delay = delay
			}
		}
		p.notifyLocked()
	}
}

func (p *retryPolicy) anyHealthyLocked() bool {
	if p.anyLocked(retryHealthy) {
		return true
	}
	return p.streaming != nil && p.streaming()
}

func (p *retryPolicy) anyLocked(state retryState) bool {
	for _, m := range p.members {
		if m.state == state {
			return true
		}
	}
	return false
}

// sleepContext waits for d, or until ctx ends; it reports whether the full
// delay passed.
func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// newRetryPolicy builds the policy for one download attempt from the HTTP
// configuration. streaming is optional, see retryPolicy.streaming.
func (f *Fetcher) newRetryPolicy(streaming func() bool) *retryPolicy {
	maxRounds := defaultMaxRetries
	delaySeconds := defaultMaxRetryDelaySeconds
	if f.config != nil {
		switch {
		case f.config.UnlimitedRetries:
			maxRounds = -1
		case f.config.MaxRetries >= 0:
			maxRounds = f.config.MaxRetries
		}
		if f.config.MaxRetryDelaySeconds > 0 {
			delaySeconds = f.config.MaxRetryDelaySeconds
		}
	}
	unit := f.retryUnit
	if unit <= 0 {
		unit = time.Second
	}
	return newRetryPolicy(maxRounds, unit, time.Duration(delaySeconds)*unit, streaming)
}
