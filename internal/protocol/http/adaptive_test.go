package http

import (
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
}

func (s simServer) speed(active int) int64 {
	speed := int64(active) * s.perConn
	if s.totalCap > 0 && speed > s.totalCap {
		speed = s.totalCap
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
	adds     []time.Time
	parks    []time.Time
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
		if s.maxConns > 0 && r.active+1 > s.maxConns {
			// The new connection gets 403/429 and is parked by the engine.
			r.c.limited(r.now, r.active, true)
		} else {
			r.active++
		}
	case parkSlowest:
		r.parks = append(r.parks, r.now)
		r.active--
	}
	r.timeline = append(r.timeline, r.active)
	return d
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
	// After the first rejection, the controller never asks above 4 again.
	if len(r.adds) != 4 {
		t.Fatalf("got %d additions, want 4 (three that worked and the rejected fifth)", len(r.adds))
	}
}

func TestAdaptiveProbesEvery30s(t *testing.T) {
	r := newSimRun(16)
	capped := simServer{perConn: mb, totalCap: 3 * mb}
	r.run(capped, 30*time.Second)
	if r.active != 3 {
		t.Fatalf("setup: active = %d, want 3 (timeline %v)", r.active, r.timeline)
	}
	r.adds, r.timeline = nil, nil
	r.run(capped, 5*time.Minute)
	t.Logf("timeline: %v", r.timeline)
	// One probe per 30 s, each given back because it does not help.
	if n := len(r.adds); n < 9 || n > 11 {
		t.Fatalf("got %d probes in 5 minutes, want 10 +- 1", n)
	}
	for i := 1; i < len(r.adds); i++ {
		if gap := r.adds[i].Sub(r.adds[i-1]); gap < 30*time.Second {
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
	r.adds = nil
	r.run(simServer{perConn: mb, totalCap: 4 * mb}, 25*time.Second)
	if len(r.adds) != 0 {
		t.Fatalf("got %d additions within 25 s of a 429, want none before the probe", len(r.adds))
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
