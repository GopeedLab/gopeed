package http

import (
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

const mb = int64(1000 * 1000)

// simServer models a server for the pure controller tests: each connection
// gets perConn bytes/s and the total is capped at totalCap (0 means no cap).
type simServer struct {
	perConn  int64
	totalCap int64
	maxConns int // a new connection above this count is rejected (0 means no limit)
	// strayAt rejects the first new connection that would reach this count,
	// once, and accepts every later one (0 means never).
	strayAt int
	// strays rejects the first new connection that would reach each listed
	// count, once per count.
	strays []int
	// refuseResume models a server that still counts a pruned connection's
	// cancelled request for 10 s: every request that would bring the count
	// back to what it was before the prune is refused in that time.
	refuseResume bool
	// sqrt makes the speed grow with the square root of the count.
	sqrt bool
	// noise, when set, multiplies every window by 1 + U(-jitter, +jitter).
	noise  *rand.Rand
	jitter float64
}

func (s simServer) speed(active int) int64 {
	speed := int64(active) * s.perConn
	if s.sqrt {
		speed = int64(math.Sqrt(float64(active)) * float64(s.perConn))
	}
	if s.totalCap > 0 && speed > s.totalCap {
		speed = s.totalCap
	}
	if s.noise != nil {
		speed = int64(float64(speed) * (1 + s.jitter*(2*s.noise.Float64()-1)))
	}
	return speed
}

// simRun drives the controller with a simulated clock. It applies every
// decision to the connection count the way the engine does and records the
// count after each tick.
type simRun struct {
	c        *adaptiveController
	now      time.Time
	active   int
	timeline []int
	adds     []time.Time // every addConn
	probes   []time.Time // addConn that starts a judged addition (growth or probe)
	parks    []time.Time // every park decision
	prunes   []time.Time // parkNewest
	strayed  bool
	strayHit map[int]bool
	resumes  int // requests refused while a pruned connection's slot lingered
	pruneAt  time.Time
	pruneN   int
}

func newSimRun(ceiling int) *simRun {
	now := time.Unix(1_700_000_000, 0)
	return &simRun{c: newAdaptiveController(ceiling, now), now: now, active: 1}
}

func (r *simRun) step(s simServer) adaptiveDecision {
	r.now = r.now.Add(adaptiveTickInterval)
	d := r.c.tick(r.now, s.speed(r.active), r.active)
	switch d {
	case addConn:
		r.adds = append(r.adds, r.now)
		if r.c.phase == phaseAdded {
			r.probes = append(r.probes, r.now)
		}
		if s.maxConns > 0 && r.active+1 > s.maxConns {
			// The new connection gets 403/429 and is parked by the engine.
			r.c.limited(r.now, r.active, true)
		} else if s.strayAt > 0 && r.active+1 == s.strayAt && !r.strayed {
			r.strayed = true
			r.c.limited(r.now, r.active, true)
		} else if r.strayOnce(s, r.active+1) {
			r.c.limited(r.now, r.active, true)
		} else if s.refuseResume && r.active+1 >= r.pruneN && r.now.Sub(r.pruneAt) <= 10*time.Second {
			r.resumes++
			r.c.limited(r.now, r.active, true)
		} else {
			r.active++
		}
	case parkSlowest:
		r.parks = append(r.parks, r.now)
		r.active--
	case parkNewest:
		r.parks = append(r.parks, r.now)
		r.prunes = append(r.prunes, r.now)
		r.pruneAt, r.pruneN = r.now, r.active
		r.active--
	}
	r.timeline = append(r.timeline, r.active)
	return d
}

func (r *simRun) strayOnce(s simServer, count int) bool {
	for _, n := range s.strays {
		if n == count && !r.strayHit[n] {
			if r.strayHit == nil {
				r.strayHit = map[int]bool{}
			}
			r.strayHit[n] = true
			return true
		}
	}
	return false
}

func (r *simRun) run(s simServer, d time.Duration) {
	for end := r.now.Add(d); r.now.Before(end); {
		r.step(s)
	}
}

func maxOf(xs []int) int {
	m := 0
	for _, x := range xs {
		if x > m {
			m = x
		}
	}
	return m
}

func TestAdaptiveStartsAtOneAndGrowsWhileFaster(t *testing.T) {
	r := newSimRun(16)
	if r.c.tick(r.now.Add(adaptiveTickInterval), mb, 1) != holdConns {
		t.Fatal("the first window must only measure the single starting connection")
	}
	r.now = r.now.Add(adaptiveTickInterval)
	r.run(simServer{perConn: mb, totalCap: 6 * mb}, 3*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	if got := r.active; got < 6 || got > 7 {
		t.Fatalf("settled at %d connections, want 6 or 7", got)
	}
	if m := maxOf(r.timeline); m > 16 {
		t.Fatalf("reached %d connections, above the ceiling of 16", m)
	}
	if m := maxOf(r.timeline); m > 7 {
		t.Fatalf("reached %d connections, want at most one probe above 6", m)
	}
}

func TestAdaptivePlateauStopsGrowth(t *testing.T) {
	r := newSimRun(16)
	// The server gives 1 MB/s in total, however many connections ask.
	flat := simServer{perConn: mb, totalCap: mb}
	r.run(flat, 20*time.Second)
	t.Logf("timeline: %v", r.timeline)
	if len(r.adds) != 1 {
		t.Fatalf("got %d additions in the first 20 s, want exactly 1 (the one that found the plateau)", len(r.adds))
	}
	if len(r.parks) != 1 {
		t.Fatalf("got %d parks, want the useless connection given back once", len(r.parks))
	}
	if r.active != 1 {
		t.Fatalf("active = %d after the plateau, want 1", r.active)
	}
}

func TestAdaptiveShrinksOnDrop(t *testing.T) {
	r := newSimRun(16)
	fast := simServer{perConn: mb, totalCap: 4 * mb}
	r.run(fast, 30*time.Second)
	if r.active != 4 {
		t.Fatalf("setup: active = %d, want 4 (timeline %v)", r.active, r.timeline)
	}
	// The throughput drops by 20% for two windows, well before the next probe.
	slow := simServer{perConn: mb, totalCap: 4 * mb * 80 / 100}
	if d := r.step(slow); d != holdConns {
		t.Fatalf("first low window gave %v, want hold", d)
	}
	if d := r.step(slow); d != parkSlowest {
		t.Fatalf("second low window gave %v, want parkSlowest", d)
	}
}

func TestAdaptiveShrinkWalksDownWhileFree(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 6 * mb}, 30*time.Second)
	if r.active != 6 {
		t.Fatalf("setup: active = %d, want 6 (timeline %v)", r.active, r.timeline)
	}
	// The server now gives only 3 MB/s in total. Parking costs nothing down to
	// three connections, and the park that costs speed is undone.
	r.timeline = nil
	r.run(simServer{perConn: mb, totalCap: 3 * mb}, 25*time.Second)
	t.Logf("timeline after the slowdown: %v", r.timeline)
	if r.active != 3 {
		t.Fatalf("active = %d after the slowdown, want 3", r.active)
	}
}

func TestAdaptiveCeilingFromLimited(t *testing.T) {
	r := newSimRun(16)
	limit := simServer{perConn: mb, maxConns: 4}
	r.run(limit, 5*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	if m := maxOf(r.timeline); m > 4 {
		t.Fatalf("reached %d connections, want at most 4", m)
	}
	if r.active != 4 {
		t.Fatalf("active = %d, want 4", r.active)
	}
	// Two refusals within 10 s set the ceiling; after that the controller
	// never asks above 4 again.
	if len(r.probes) != 5 {
		t.Fatalf("got %d judged additions, want 5 (three that worked and the fifth refused twice)", len(r.probes))
	}
	if gap := r.probes[4].Sub(r.probes[3]); gap > 5*adaptiveTickInterval {
		t.Fatalf("the second try came %v after the first refusal, want within 5 windows", gap)
	}
}

func TestAdaptiveProbesEvery30s(t *testing.T) {
	r := newSimRun(16)
	capped := simServer{perConn: mb, totalCap: 3 * mb}
	r.run(capped, 30*time.Second)
	if r.active != 3 {
		t.Fatalf("setup: active = %d, want 3 (timeline %v)", r.active, r.timeline)
	}
	r.probes, r.prunes, r.timeline = nil, nil, nil
	r.run(capped, 5*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	// One test per 30 s: a probe that is given back because it does not
	// help, or (every 4th interval) a prune that is undone because the
	// connection is needed.
	events := append(append([]time.Time(nil), r.probes...), r.prunes...)
	sort.Slice(events, func(i, j int) bool { return events[i].Before(events[j]) })
	if n := len(events); n < 9 || n > 11 {
		t.Fatalf("got %d probes and prunes in 5 minutes, want 10 +- 1", n)
	}
	if n := len(r.prunes); n < 2 || n > 3 {
		t.Fatalf("got %d prunes in 10 intervals, want every 4th (2 or 3)", n)
	}
	for i := 1; i < len(events); i++ {
		if gap := events[i].Sub(events[i-1]); gap < 30*time.Second {
			t.Fatalf("probes %v apart, want at least 30 s", gap)
		}
	}
	if r.active != 3 {
		t.Fatalf("active = %d after useless probes, want 3", r.active)
	}
	if m := maxOf(r.timeline); m > 4 {
		t.Fatalf("reached %d connections, want at most one probe at a time", m)
	}

	// When the server opens up, a probe helps, stays, and growth resumes.
	r.run(simServer{perConn: mb, totalCap: 5 * mb}, 60*time.Second)
	if r.active != 5 {
		t.Fatalf("active = %d after the server opened up, want 5", r.active)
	}
}

func TestAdaptiveNeverAboveCeiling(t *testing.T) {
	r := newSimRun(3)
	r.run(simServer{perConn: mb}, 3*time.Minute)
	if m := maxOf(r.timeline); m > 3 {
		t.Fatalf("reached %d connections, above the ceiling of 3", m)
	}
	if r.active != 3 {
		t.Fatalf("active = %d, want 3", r.active)
	}
}

func TestAdaptiveLowerCeilingParks(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 6 * mb}, 30*time.Second)
	if r.active != 6 {
		t.Fatalf("setup: active = %d, want 6", r.active)
	}
	r.c.setCeiling(4)
	r.run(simServer{perConn: mb, totalCap: 6 * mb}, 10*time.Second)
	if r.active != 4 {
		t.Fatalf("active = %d after lowering the ceiling, want 4", r.active)
	}
}

func TestAdaptiveFinishingConnectionsAreNotASlowdown(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 4 * mb}, 30*time.Second)
	if r.active != 4 {
		t.Fatalf("setup: active = %d, want 4", r.active)
	}
	// Two connections run out of work: the engine reports two fewer, and the
	// task speed halves. That is not a reason to park anything.
	r.active = 2
	r.parks = nil
	for i := 0; i < 4; i++ {
		r.step(simServer{perConn: mb})
	}
	if len(r.parks) != 0 {
		t.Fatalf("parked %d connections after others finished, want 0", len(r.parks))
	}
}

func TestAdaptiveMidStreamLimitParksAndHoldsOff(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 4 * mb}, 30*time.Second)
	if r.active != 4 {
		t.Fatalf("setup: active = %d, want 4", r.active)
	}
	// One connection gets 429 on a later request; the engine parks it.
	r.active = 3
	r.c.limited(r.now, 3, false)
	r.probes = nil
	r.run(simServer{perConn: mb, totalCap: 4 * mb}, 25*time.Second)
	if len(r.probes) != 0 {
		t.Fatalf("got %d additions within 25 s of a 429, want none before the probe", len(r.probes))
	}
}

// A slowdown that starts while a probe is being judged must still shrink:
// giving the probe back must not make the slower speed the new best.
func TestAdaptiveSlowdownDuringProbeStillShrinks(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 6 * mb}, 30*time.Second)
	if r.active != 6 {
		t.Fatalf("setup: active = %d, want 6", r.active)
	}
	// Run until the next probe adds a seventh connection.
	for i := 0; i < 100 && r.active != 7; i++ {
		r.step(simServer{perConn: mb, totalCap: 6 * mb})
	}
	if r.active != 7 {
		t.Fatal("setup: no probe within 200 s")
	}
	r.timeline = nil
	r.run(simServer{perConn: mb, totalCap: 2 * mb}, 30*time.Second)
	t.Logf("timeline after the slowdown: %v", r.timeline)
	if r.active > 3 {
		t.Fatalf("active = %d after the server slowed to 2 MB/s, want connections given back (<= 3)", r.active)
	}
}

// One stray refusal of a new connection does not lower the ceiling.
func TestAdaptiveOneStrayRefusalKeepsCeiling(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 6 * mb, strayAt: 5}, 2*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	if !r.strayed {
		t.Fatal("setup: the stray refusal never happened")
	}
	if r.active < 6 {
		t.Fatalf("active = %d after one stray refusal, want 6 (the ceiling must not drop to 4)", r.active)
	}
}

// Each added connection must bring at least half of its fair share, so a
// server that scales linearly reaches a high ceiling.
// Up to about 21 connections each addition gains over 5% and growth chains,
// so the ramp is fast. Above that a linear gain (1/n) is under 5%: the
// connection is kept, but growth waits for the next probe interval (D4), and
// every 4th interval is a prune test (D5) instead of a probe.
func TestAdaptiveReachesHighCeilingOnLinearServer(t *testing.T) {
	r := newSimRun(32)
	server := simServer{perConn: mb, totalCap: 30 * mb}
	r.run(server, 90*time.Second)
	if r.active < 20 {
		t.Fatalf("active = %d after 90 s with a server that scales to 30, want at least 20", r.active)
	}
	r.run(server, 7*time.Minute-90*time.Second)
	t.Logf("timeline: %v", r.timeline)
	if r.active < 28 {
		t.Fatalf("active = %d after 7 minutes with a server that scales to 30, want at least 28", r.active)
	}
	if m := maxOf(r.timeline); m > 32 {
		t.Fatalf("reached %d connections, above the ceiling of 32", m)
	}
}

func TestAdaptiveKeepBar(t *testing.T) {
	cases := []struct {
		n    int
		want float64
	}{{1, 0.1}, {5, 0.1}, {8, 0.0625}, {10, 0.05}, {16, 0.03125}, {20, 0.03}, {30, 0.03}}
	for _, tc := range cases {
		if got := adaptiveKeepBar(tc.n); got != tc.want {
			t.Errorf("adaptiveKeepBar(%d) = %v, want %v", tc.n, got, tc.want)
		}
	}
}

// A server whose speed grows with the square root of the count still gets
// several connections quickly: the keep bar starts at 10%, not 50%.
func TestAdaptiveSublinearServerGrowsFast(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, sqrt: true}, 30*time.Second)
	t.Logf("timeline: %v", r.timeline)
	if r.active < 5 {
		t.Fatalf("active = %d after 30 s on a sqrt(n) server, want at least 5", r.active)
	}
}

// With +-5% noise on every window, a capped server does not ratchet the count
// upwards: useless connections that noise lets in are pruned again.
func TestAdaptiveJitterStaysNearCap(t *testing.T) {
	for _, capConns := range []int{6, 12, 20} {
		for seed := int64(1); seed <= 10; seed++ {
			r := newSimRun(32)
			s := simServer{perConn: mb, totalCap: int64(capConns) * mb, noise: rand.New(rand.NewSource(seed)), jitter: 0.05}
			r.run(s, 15*time.Minute)
			if r.active > capConns+2 || r.active < capConns-1 {
				t.Errorf("cap %d, seed %d: final count %d, want %d..%d (max %d, %d probes, %d prunes)",
					capConns, seed, r.active, capConns-1, capConns+2, maxOf(r.timeline), len(r.probes), len(r.prunes))
			} else {
				t.Logf("cap %d, seed %d: final %d, max %d, %d prunes", capConns, seed, r.active, maxOf(r.timeline), len(r.prunes))
			}
		}
	}
}

// A connection that does not help is found by the periodic prune, given
// back, and the count stays one lower for a probe interval.
func TestAdaptivePruneGivesBackUselessConnection(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 4 * mb}, 30*time.Second)
	if r.active != 4 {
		t.Fatalf("setup: active = %d, want 4", r.active)
	}
	// One extra connection slips in (as noise might let it): the server is
	// capped, so the fifth adds nothing.
	r.active = 5
	r.c.expect = 5
	r.prunes = nil
	r.run(simServer{perConn: mb, totalCap: 4 * mb}, 3*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	if len(r.prunes) == 0 {
		t.Fatal("no prune in 3 minutes, want one every 4th probe interval")
	}
	if r.active != 4 {
		t.Fatalf("active = %d, want the useless fifth connection given back (4)", r.active)
	}
}

// A strike is cleared once a later connection has run. Then a single refusal
// shortly after is a new stray, not the second strike.
func TestAdaptiveSuccessfulRetryClearsStrike(t *testing.T) {
	r := newSimRun(16)
	// The first try at 5 and the first try at 6 are both refused once. The
	// second refusal comes 8 s after the first, inside the 5-window strike
	// window, but the successful retry at 5 in between cleared the strike.
	r.run(simServer{perConn: mb, totalCap: 8 * mb, strays: []int{5, 6}}, 2*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	if len(r.strayHit) != 2 {
		t.Fatalf("setup: %d stray refusals happened, want 2", len(r.strayHit))
	}
	if r.c.ceiling != 16 {
		t.Fatalf("ceiling = %d, want 16 (two refusals with a successful retry between them are not a limit)", r.c.ceiling)
	}
	if r.active < 8 {
		t.Fatalf("active = %d, want 8", r.active)
	}
}

// D6: the request that resumes a pruned connection may be refused because
// the server still counts the request just cancelled. That is never a
// strike: the ceiling stays, and the connection stays parked until the next
// probe brings it back.
func TestAdaptiveRefusedPruneResumeKeepsCeiling(t *testing.T) {
	r := newSimRun(16)
	r.run(simServer{perConn: mb, totalCap: 6 * mb, refuseResume: true}, 15*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	if r.resumes < 3 {
		t.Fatalf("setup: only %d resumes were refused, want several prune cycles", r.resumes)
	}
	if r.c.ceiling != 16 {
		t.Fatalf("ceiling = %d after %d refused resumes, want 16 (unchanged)", r.c.ceiling, r.resumes)
	}
	if r.active < 5 {
		t.Fatalf("active = %d, want the count to recover to 5 or 6 between prunes", r.active)
	}
}
