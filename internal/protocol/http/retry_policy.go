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
// tried again. Every connection keeps its own pace and its own allowance:
//
//   - A connection whose request fails while another connection of the task
//     is healthy (it got a valid 200 or 206 and is still downloading) retries
//     without using its allowance.
//   - Otherwise the retry is charged to that connection's own allowance:
//     maxRetries additional requests, or unlimited. Other connections are not
//     charged, and nobody waits for anybody: a request still waiting for its
//     response headers does not hold back a connection that has already
//     failed.
//   - Any valid response resets the allowance of every unfinished connection.
//   - The delay before a retry grows with the connection's own consecutive
//     failures: one unit, two units, ... up to maxDelay. A peer's success does
//     not shorten it.
//
// A connection that has used up its allowance does not give up yet: its chunk
// stays recoverable while another request of the task is healthy or still in
// flight. It waits, and if that request succeeds it gets the reset allowance
// and tries again. Only when every remaining connection has used up its
// allowance, and no request is healthy or in flight, is the policy exhausted:
// every connection then gives up and the task fails. A policy serves one
// download attempt; a new attempt (Start after a failure or a pause) gets a new
// policy and so full allowances.
type retryPolicy struct {
	// maxRetries is the number of additional requests a connection may make
	// on its own allowance; negative means unlimited.
	maxRetries int
	// unit is the delay of the first retry; the n-th waits n units.
	unit time.Duration
	// maxDelay caps every delay.
	maxDelay time.Duration
	// streaming, when set, reports a healthy download outside the tracked
	// requests: the Resolve response still being read in the background.
	streaming func() bool

	mu        sync.Mutex
	members   map[*connection]*retryMember
	exhausted bool // no connection can recover; the attempt has failed
	changed   chan struct{}
}

type retryState int

const (
	// retryPending: a request is about to be sent, waits for its response
	// headers, or waits out its retry delay. It has not failed, but it is not
	// healthy yet either; it may still recover the task.
	retryPending retryState = iota
	// retryHealthy: the last request got a valid 200 or 206 and is downloading.
	retryHealthy
	// retrySpent: the last request failed and the connection's allowance is
	// used up. It waits for a peer's success to reset it.
	retrySpent
)

type retryMember struct {
	state retryState
	// used counts the retries charged to the allowance since the last reset.
	used int
	// failures counts the consecutive failed requests of this connection; it
	// sets the backoff and is cleared by the connection's own success.
	failures int
}

func newRetryPolicy(maxRetries int, unit, maxDelay time.Duration, streaming func() bool) *retryPolicy {
	if unit <= 0 {
		unit = time.Second
	}
	if maxDelay < unit {
		maxDelay = unit
	}
	return &retryPolicy{
		maxRetries: maxRetries,
		unit:       unit,
		maxDelay:   maxDelay,
		streaming:  streaming,
		members:    make(map[*connection]*retryMember),
		changed:    make(chan struct{}),
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

func (p *retryPolicy) memberLocked(c *connection) *retryMember {
	m, ok := p.members[c]
	if !ok {
		m = &retryMember{}
		p.members[c] = m
	}
	return m
}

// join adds a request of the task; it starts out pending.
func (p *retryPolicy) join(c *connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.memberLocked(c).state = retryPending
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

// success records a valid 200 or 206 for c. It resets the allowance of every
// unfinished connection, and lets connections that had used theirs up try
// again.
func (p *retryPolicy) success(c *connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m := p.memberLocked(c)
	m.state = retryHealthy
	m.failures = 0
	if !p.exhausted {
		for _, other := range p.members {
			other.used = 0
			if other.state == retrySpent {
				other.state = retryPending
			}
		}
	}
	p.notifyLocked()
}

// next records that the last request of c failed and decides whether and when
// it is tried again. It returns the delay to wait before the next request, or
// false when the task cannot recover or ctx ends. It blocks only while c has
// used up its allowance and another request may still recover the task.
func (p *retryPolicy) next(ctx context.Context, c *connection) (time.Duration, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m := p.memberLocked(c)
	m.failures++
	if p.exhausted {
		return 0, false
	}

	// c is not healthy any more, so it does not count for itself.
	m.state = retrySpent
	switch {
	case p.anyHealthyLocked():
		// Free retry: the task is making progress.
		m.state = retryPending
	case p.maxRetries < 0 || m.used < p.maxRetries:
		m.used++
		m.state = retryPending
	}
	p.notifyLocked()
	if m.state == retryPending {
		return p.delayFor(m.failures), true
	}

	// c's allowance is used up: wait for a peer's success to reset it, or for
	// the last request that could do so to go.
	for {
		if p.exhausted {
			return 0, false
		}
		if m.state == retryPending {
			// A peer succeeded and reset the allowances. Like any retry next
			// to a healthy peer, this one is free.
			return p.delayFor(m.failures), true
		}
		if !p.recoverableLocked() {
			p.exhausted = true
			p.notifyLocked()
			return 0, false
		}
		changed := p.changed
		p.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
		}
		p.mu.Lock()
		if ctx.Err() != nil {
			// c leaves the task; a spent member no longer holds anyone.
			return 0, false
		}
	}
}

// recoverableLocked reports whether a request of the task is healthy or still
// in flight, and so may yet reset the allowances of the connections that used
// theirs up.
func (p *retryPolicy) recoverableLocked() bool {
	return p.anyHealthyLocked() || p.anyLocked(retryPending)
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
	maxRetries := defaultMaxRetries
	delaySeconds := defaultMaxRetryDelaySeconds
	if f.config != nil {
		switch {
		case f.config.UnlimitedRetries:
			maxRetries = -1
		case f.config.MaxRetries >= 0:
			maxRetries = f.config.MaxRetries
		}
		if f.config.MaxRetryDelaySeconds > 0 {
			delaySeconds = f.config.MaxRetryDelaySeconds
		}
	}
	unit := f.retryUnit
	if unit <= 0 {
		unit = time.Second
	}
	return newRetryPolicy(maxRetries, unit, time.Duration(delaySeconds)*unit, streaming)
}
