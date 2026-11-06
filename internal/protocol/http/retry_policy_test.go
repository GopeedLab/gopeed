package http

import (
	"context"
	"testing"
	"time"
)

const testUnit = time.Second

func newTestPolicy(maxRounds int) *retryPolicy {
	return newRetryPolicy(maxRounds, testUnit, 5*testUnit, nil)
}

func mustRetry(t *testing.T, p *retryPolicy, c *connection, want time.Duration) {
	t.Helper()
	got, ok := p.next(context.Background(), c)
	if !ok {
		t.Fatalf("next() gave up, want a retry after %v", want)
	}
	if got != want {
		t.Fatalf("next() delay = %v, want %v", got, want)
	}
}

func mustGiveUp(t *testing.T, p *retryPolicy, c *connection) {
	t.Helper()
	if d, ok := p.next(context.Background(), c); ok {
		t.Fatalf("next() = retry after %v, want to give up", d)
	}
}

// nextAsync runs next in the background, for a member that has to wait.
func nextAsync(p *retryPolicy, c *connection) <-chan time.Duration {
	ch := make(chan time.Duration, 1)
	go func() {
		d, ok := p.next(context.Background(), c)
		if !ok {
			d = -1
		}
		ch <- d
	}()
	return ch
}

func waitDelay(t *testing.T, ch <-chan time.Duration) time.Duration {
	t.Helper()
	select {
	case d := <-ch:
		return d
	case <-time.After(5 * time.Second):
		t.Fatal("next() did not return")
		return 0
	}
}

func assertBlocked(t *testing.T, ch <-chan time.Duration) {
	t.Helper()
	select {
	case d := <-ch:
		t.Fatalf("next() returned %v, want it to wait for the pending request", d)
	case <-time.After(50 * time.Millisecond):
	}
}

// With nothing else running, every failure is a round of its own: five rounds
// waiting 1 to 5 seconds, then the policy gives up.
func TestRetryPolicy_RoundsThenGiveUp(t *testing.T) {
	p := newTestPolicy(5)
	c := &connection{}
	p.join(c)
	for round := 1; round <= 5; round++ {
		mustRetry(t, p, c, time.Duration(round)*testUnit)
	}
	mustGiveUp(t, p, c)
	// Exhaustion is final for the attempt.
	p.join(c)
	mustGiveUp(t, p, c)
}

func TestRetryPolicy_DelayCap(t *testing.T) {
	p := newRetryPolicy(6, testUnit, 3*testUnit, nil)
	c := &connection{}
	p.join(c)
	for _, want := range []time.Duration{1, 2, 3, 3, 3, 3} {
		mustRetry(t, p, c, want*testUnit)
	}
	mustGiveUp(t, p, c)
}

func TestRetryPolicy_ZeroRoundsFailsAtOnce(t *testing.T) {
	p := newTestPolicy(0)
	c := &connection{}
	p.join(c)
	mustGiveUp(t, p, c)
}

func TestRetryPolicy_Unlimited(t *testing.T) {
	p := newTestPolicy(-1)
	c := &connection{}
	p.join(c)
	for i := 1; i <= 100; i++ {
		mustRetry(t, p, c, min(time.Duration(i), 5)*testUnit)
	}
}

// While another request downloads, failures of any status are retried without
// using the budget, with a delay growing per request.
func TestRetryPolicy_HealthyPeerKeepsRetrying(t *testing.T) {
	p := newTestPolicy(1)
	healthy, failing := &connection{}, &connection{}
	p.join(healthy)
	p.join(failing)
	p.success(healthy)
	for i := 1; i <= 20; i++ {
		mustRetry(t, p, failing, min(time.Duration(i), 5)*testUnit)
	}
	if p.rounds != 0 {
		t.Fatalf("rounds = %d while a peer is healthy, want 0", p.rounds)
	}

	// A completed request no longer counts: the failing one is on its own.
	p.leave(healthy)
	mustRetry(t, p, failing, testUnit)
	mustGiveUp(t, p, failing)
}

// A request that fails while another one still waits for its response waits
// too, instead of starting a round or retrying freely: two failing requests
// must not keep each other going.
func TestRetryPolicy_WaitsForPendingRequest(t *testing.T) {
	p := newTestPolicy(5)
	a, b := &connection{}, &connection{}
	p.join(a)
	p.join(b)

	aNext := nextAsync(p, a)
	assertBlocked(t, aNext)

	// b gets a valid response: a is retried freely.
	p.success(b)
	if d := waitDelay(t, aNext); d != testUnit {
		t.Fatalf("delay after the peer recovered = %v, want %v", d, testUnit)
	}

	// b fails while a is pending (sleeping before its retry): b waits.
	bNext := nextAsync(p, b)
	assertBlocked(t, bNext)
	// a fails too: every request has failed, one round starts for both.
	aNext = nextAsync(p, a)
	if d := waitDelay(t, aNext); d != testUnit {
		t.Fatalf("a: first round delay = %v, want %v", d, testUnit)
	}
	if d := waitDelay(t, bNext); d != testUnit {
		t.Fatalf("b: first round delay = %v, want %v", d, testUnit)
	}
	if p.rounds != 1 {
		t.Fatalf("rounds = %d, want one round for two failures", p.rounds)
	}
}

// Rounds count once for the task, however many requests fail in them, and
// each round lets every failed request try once.
func TestRetryPolicy_RoundsCountedPerTask(t *testing.T) {
	const members = 4
	p := newTestPolicy(3)
	conns := make([]*connection, members)
	for i := range conns {
		conns[i] = &connection{}
		p.join(conns[i])
	}
	for round := 1; round <= 3; round++ {
		results := make([]<-chan time.Duration, members)
		for i, c := range conns {
			results[i] = nextAsync(p, c)
		}
		for i := range conns {
			if d := waitDelay(t, results[i]); d != time.Duration(round)*testUnit {
				t.Fatalf("round %d, conn %d: delay = %v, want %v", round, i, d, time.Duration(round)*testUnit)
			}
		}
		if p.rounds != round {
			t.Fatalf("rounds = %d after round %d", p.rounds, round)
		}
	}
	results := make([]<-chan time.Duration, members)
	for i, c := range conns {
		results[i] = nextAsync(p, c)
	}
	for i := range conns {
		if d := waitDelay(t, results[i]); d != -1 {
			t.Fatalf("conn %d retried after %v with the rounds used up", i, d)
		}
	}
}

// A valid response from any request resets the whole task: the rounds and
// their delays start over for every request, not only the one that recovered.
func TestRetryPolicy_SuccessResetsTask(t *testing.T) {
	p := newTestPolicy(3)
	a, b := &connection{}, &connection{}
	p.join(a)
	p.join(b)
	for round := 1; round <= 2; round++ {
		aNext, bNext := nextAsync(p, a), nextAsync(p, b)
		waitDelay(t, aNext)
		waitDelay(t, bNext)
	}
	if p.rounds != 2 {
		t.Fatalf("rounds = %d, want 2", p.rounds)
	}

	// a recovers in the third round; b failed in it.
	bNext := nextAsync(p, b)
	assertBlocked(t, bNext)
	p.success(a)
	if d := waitDelay(t, bNext); d != testUnit {
		t.Fatalf("b's delay next to a healthy peer = %v, want the reset %v", d, testUnit)
	}
	if p.rounds != 0 {
		t.Fatalf("rounds = %d after a valid response, want 0", p.rounds)
	}

	// a completes, b keeps failing: it gets a fresh allowance of three rounds.
	p.leave(a)
	for round := 1; round <= 3; round++ {
		mustRetry(t, p, b, time.Duration(round)*testUnit)
	}
	mustGiveUp(t, p, b)
}

func TestRetryPolicy_StreamingCountsAsHealthy(t *testing.T) {
	streaming := true
	p := newRetryPolicy(1, testUnit, 5*testUnit, func() bool { return streaming })
	c := &connection{}
	p.join(c)
	for i := 1; i <= 10; i++ {
		mustRetry(t, p, c, min(time.Duration(i), 5)*testUnit)
	}
	streaming = false
	mustRetry(t, p, c, testUnit)
	mustGiveUp(t, p, c)
}

func TestRetryPolicy_CancelWhileWaiting(t *testing.T) {
	p := newTestPolicy(5)
	a, b := &connection{}, &connection{}
	p.join(a)
	p.join(b)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool, 1)
	go func() {
		_, ok := p.next(ctx, a)
		done <- ok
	}()
	select {
	case <-done:
		t.Fatal("next() returned while the peer was pending")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("next() allowed a retry after cancellation")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("next() did not return after cancellation")
	}
}

func TestSleepContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if sleepContext(ctx, time.Hour) {
		t.Fatal("sleepContext() completed after cancellation")
	}
	if time.Since(start) > time.Second {
		t.Fatal("sleepContext() did not stop at once")
	}
	if !sleepContext(context.Background(), time.Millisecond) {
		t.Fatal("sleepContext() did not complete")
	}
}

func TestFetcher_NewRetryPolicyFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config
		wantMax   int
		wantDelay time.Duration
	}{
		{"no config", nil, 5, 5 * time.Second},
		{"defaults", new(FetcherManager).DefaultConfig().(*config), 5, 5 * time.Second},
		{"custom", &config{MaxRetries: 2, MaxRetryDelaySeconds: 9}, 2, 9 * time.Second},
		{"zero", &config{}, 0, 5 * time.Second},
		{"negative uses default", &config{MaxRetries: -3}, 5, 5 * time.Second},
		{"unlimited", &config{MaxRetries: 2, UnlimitedRetries: true}, -1, 5 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := (&Fetcher{config: tt.cfg}).newRetryPolicy(nil)
			if p.maxRounds != tt.wantMax || p.maxDelay != tt.wantDelay || p.unit != time.Second {
				t.Fatalf("policy = {rounds %d, max delay %v, unit %v}, want {%d, %v, 1s}", p.maxRounds, p.maxDelay, p.unit, tt.wantMax, tt.wantDelay)
			}
		})
	}
}
