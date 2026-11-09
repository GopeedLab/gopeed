package http

import (
	"context"
	"testing"
	"time"
)

const testUnit = time.Second

func newTestPolicy(maxRetries int) *retryPolicy {
	return newRetryPolicy(maxRetries, testUnit, 5*testUnit, nil)
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

// allowanceUsed returns how many retries c has charged since the last reset.
func allowanceUsed(p *retryPolicy, c *connection) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.members[c]; ok {
		return m.used
	}
	return -1
}

// With nothing else running, a connection retries on its own allowance: five
// retries waiting 1 to 5 seconds, then the policy gives up.
func TestRetryPolicy_AllowanceThenGiveUp(t *testing.T) {
	p := newTestPolicy(5)
	c := &connection{}
	p.join(c)
	for retry := 1; retry <= 5; retry++ {
		mustRetry(t, p, c, time.Duration(retry)*testUnit)
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

func TestRetryPolicy_ZeroRetriesFailsAtOnce(t *testing.T) {
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
// using the allowance, with a delay growing per connection.
func TestRetryPolicy_HealthyPeerKeepsRetrying(t *testing.T) {
	p := newTestPolicy(1)
	healthy, failing := &connection{}, &connection{}
	p.join(healthy)
	p.join(failing)
	p.success(healthy)
	for i := 1; i <= 20; i++ {
		mustRetry(t, p, failing, min(time.Duration(i), 5)*testUnit)
	}
	if used := allowanceUsed(p, failing); used != 0 {
		t.Fatalf("allowance used = %d while a peer is healthy, want 0", used)
	}

	// A completed request no longer counts: the failing one is on its own,
	// with its allowance intact and its backoff kept.
	p.leave(healthy)
	mustRetry(t, p, failing, 5*testUnit)
	mustGiveUp(t, p, failing)
}

// A connection that fails while another one still waits for its response
// headers retries at its own pace: it does not wait for the slow one.
func TestRetryPolicy_FastFailureDoesNotWaitForPending(t *testing.T) {
	p := newTestPolicy(5)
	fast, slow := &connection{}, &connection{}
	p.join(fast)
	p.join(slow) // waits for its headers throughout
	for retry := 1; retry <= 3; retry++ {
		mustRetry(t, p, fast, time.Duration(retry)*testUnit)
	}
	if used := allowanceUsed(p, fast); used != 3 {
		t.Fatalf("fast connection used %d retries, want 3", used)
	}
	if used := allowanceUsed(p, slow); used != 0 {
		t.Fatalf("pending connection was charged %d retries, want 0", used)
	}
}

// Connections failing at the same time each pay from their own allowance; no
// shared counter runs down faster because more connections fail.
func TestRetryPolicy_SimultaneousFailuresChargeOwnAllowance(t *testing.T) {
	const members = 8
	p := newTestPolicy(2)
	conns := make([]*connection, members)
	for i := range conns {
		conns[i] = &connection{}
		p.join(conns[i])
	}
	for retry := 1; retry <= 2; retry++ {
		results := make([]<-chan time.Duration, members)
		for i, c := range conns {
			results[i] = nextAsync(p, c)
		}
		for i := range conns {
			if d := waitDelay(t, results[i]); d != time.Duration(retry)*testUnit {
				t.Fatalf("retry %d, conn %d: delay = %v, want %v", retry, i, d, time.Duration(retry)*testUnit)
			}
		}
	}
	for i, c := range conns {
		if used := allowanceUsed(p, c); used != 2 {
			t.Fatalf("conn %d used %d retries after two simultaneous failures, want 2", i, used)
		}
	}
	// All allowances are used up and nothing can recover: everyone gives up.
	results := make([]<-chan time.Duration, members)
	for i, c := range conns {
		results[i] = nextAsync(p, c)
	}
	for i := range conns {
		if d := waitDelay(t, results[i]); d != -1 {
			t.Fatalf("conn %d retried after %v with every allowance used up", i, d)
		}
	}
}

// A connection that used up its allowance keeps its chunk while a slower
// request may still succeed; that success gives it a fresh allowance.
func TestRetryPolicy_LateSuccessRestoresSpentConnection(t *testing.T) {
	p := newTestPolicy(1)
	spent, slow := &connection{}, &connection{}
	p.join(spent)
	p.join(slow)
	mustRetry(t, p, spent, testUnit)
	spentNext := nextAsync(p, spent)
	assertBlocked(t, spentNext)

	p.success(slow)
	if d := waitDelay(t, spentNext); d != 2*testUnit {
		t.Fatalf("restored connection's delay = %v, want its own backoff %v", d, 2*testUnit)
	}

	// The peer completes; the restored connection has its whole allowance again.
	p.leave(slow)
	if used := allowanceUsed(p, spent); used != 0 {
		t.Fatalf("allowance used after the reset = %d, want 0", used)
	}
	mustRetry(t, p, spent, 3*testUnit)
	mustGiveUp(t, p, spent)
}

// Any valid response resets the allowance of every unfinished connection, not
// only the one that recovered.
func TestRetryPolicy_SuccessResetsEveryAllowance(t *testing.T) {
	p := newTestPolicy(3)
	a, b, c := &connection{}, &connection{}, &connection{}
	for _, conn := range []*connection{a, b, c} {
		p.join(conn)
	}
	mustRetry(t, p, a, testUnit)
	mustRetry(t, p, a, 2*testUnit)
	mustRetry(t, p, b, testUnit)
	p.success(c)
	for _, conn := range []*connection{a, b} {
		if used := allowanceUsed(p, conn); used != 0 {
			t.Fatalf("allowance used after a peer's success = %d, want 0", used)
		}
	}
}

// The task fails only when every remaining connection has used up its
// allowance and no request is healthy or in flight.
func TestRetryPolicy_FailsWhenNothingCanRecover(t *testing.T) {
	p := newTestPolicy(1)
	a, b := &connection{}, &connection{}
	p.join(a)
	p.join(b)
	mustRetry(t, p, a, testUnit)
	aNext := nextAsync(p, a)
	assertBlocked(t, aNext) // b is still in flight

	mustRetry(t, p, b, testUnit) // b's own allowance
	assertBlocked(t, aNext)
	bNext := nextAsync(p, b) // b is spent too: nothing can recover
	if d := waitDelay(t, bNext); d != -1 {
		t.Fatalf("b retried after %v with nothing left to recover", d)
	}
	if d := waitDelay(t, aNext); d != -1 {
		t.Fatalf("a retried after %v with nothing left to recover", d)
	}
}

// A spent connection also gives up when the last request that could have
// recovered it leaves without succeeding.
func TestRetryPolicy_LastHopeLeaves(t *testing.T) {
	p := newTestPolicy(0)
	a, b := &connection{}, &connection{}
	p.join(a)
	p.join(b)
	aNext := nextAsync(p, a)
	assertBlocked(t, aNext)
	p.leave(b)
	if d := waitDelay(t, aNext); d != -1 {
		t.Fatalf("a retried after %v once nothing could recover it", d)
	}
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
	mustRetry(t, p, c, 5*testUnit)
	mustGiveUp(t, p, c)
}

func TestRetryPolicy_CancelWhileWaiting(t *testing.T) {
	p := newTestPolicy(0)
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
		t.Fatal("next() returned while the peer could still recover it")
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

// Failures charging allowances race with successes resetting them; run under
// -race. While successes keep coming, no connection may give up.
func TestRetryPolicy_ConcurrentFailuresAndSuccesses(t *testing.T) {
	const members, rounds = 8, 200
	p := newTestPolicy(2)
	conns := make([]*connection, members)
	for i := range conns {
		conns[i] = &connection{}
		p.join(conns[i])
	}
	healthy := &connection{}
	p.join(healthy)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				p.success(healthy)
				p.attempt(healthy)
			}
		}
	}()
	errs := make(chan string, members)
	for _, c := range conns {
		go func(c *connection) {
			for i := 0; i < rounds; i++ {
				if _, ok := p.next(context.Background(), c); !ok {
					errs <- "a connection gave up while a peer kept succeeding"
					return
				}
				if i%3 == 0 {
					p.success(c)
				}
			}
			errs <- ""
		}(c)
	}
	for range conns {
		if msg := <-errs; msg != "" {
			t.Fatal(msg)
		}
	}
	close(stop)
	<-done
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
			if p.maxRetries != tt.wantMax || p.maxDelay != tt.wantDelay || p.unit != time.Second {
				t.Fatalf("policy = {retries %d, max delay %v, unit %v}, want {%d, %v, 1s}", p.maxRetries, p.maxDelay, p.unit, tt.wantMax, tt.wantDelay)
			}
		})
	}
}
