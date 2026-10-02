package http

import "time"

// Adaptive connections: start with one connection, add one while the task
// speed rises, and give connections back when more of them makes it slower.
// The controller below is pure: it sees only the clock, the task speed and
// the number of running connections, and returns a decision that the fetcher
// carries out. Its numbers come from the design:
//
//   - every window (adaptiveTickInterval) is one speed sample;
//   - an addition is judged on the mean of the next adaptiveEvalWindows
//     windows against the mean of the adaptiveEvalWindows windows before it;
//   - the keep bar for a connection added to n running ones is
//     adaptiveKeepBar(n) = max(3%, min(10%, 0.5/n)): 10% at the start, then
//     half of the new connection's fair share (1/n), never under 3%, so a
//     server that scales linearly can be followed up to a ceiling of 32;
//   - a gain of at least the bar and at least adaptivePlateauGain (5%) keeps
//     the connection and adds another;
//   - a gain of at least the smaller of the two keeps the connection but
//     stops growth until the next probe interval;
//   - a smaller gain is a plateau: the connection is given back;
//   - adaptiveEvalWindows windows below (1 - adaptiveShrinkDrop) of the best
//     speed park the slowest connection;
//   - every adaptiveProbeInterval, below the ceiling, one more connection is
//     tried and kept only if it helps;
//   - every adaptivePruneEvery-th interval the newest connection is parked
//     for adaptiveEvalWindows windows instead. If the speed falls by less than
//     the keep bar, the connection did not help: it stays parked, and the
//     count may not grow past n-1 for about two probe intervals (the hold
//     ends just after the next probe, so the one after that is the first
//     that may add). Otherwise it is resumed. This gives back connections
//     that window noise let in;
//   - a refusal of the request that resumes a pruned connection is never a
//     strike: the server may still count the request just cancelled. The
//     connection stays parked until the next probe, and the ceiling is
//     untouched;
//   - a 403 or 429 on a new connection, twice within adaptiveStrikeWindows
//     windows, sets the ceiling to the count that worked; a 429 or 503 on a
//     connection that already had data parks that connection (the fetcher
//     does this on the second such response in a row; while it waits to
//     retry it delivers nothing, so it is the slowest).
var (
	adaptiveTickInterval  = 2 * time.Second
	adaptiveProbeInterval = 30 * time.Second
)

const (
	adaptiveEvalWindows   = 2
	adaptiveFairShare     = 0.5  // a new connection must bring this part of 1/n
	adaptiveMinKeepGain   = 0.03 // the keep bar is never lower
	adaptiveMaxKeepGain   = 0.10 // nor higher
	adaptivePlateauGain   = 0.05
	adaptiveShrinkDrop    = 0.15
	adaptiveStrikeWindows = 5 // two refusals within this many windows set the ceiling
	adaptivePruneEvery    = 4 // every 4th probe interval is a prune test
)

// adaptiveKeepBar is the gain a connection added to n running ones must
// bring: max(3%, min(10%, 0.5/n)).
func adaptiveKeepBar(n int) float64 {
	if n < 1 {
		n = 1
	}
	return max(adaptiveMinKeepGain, min(adaptiveMaxKeepGain, adaptiveFairShare/float64(n)))
}

type adaptiveDecision int

const (
	holdConns adaptiveDecision = iota
	addConn
	parkSlowest
	parkNewest
)

func (d adaptiveDecision) String() string {
	switch d {
	case addConn:
		return "addConn"
	case parkSlowest:
		return "parkSlowest"
	case parkNewest:
		return "parkNewest"
	default:
		return "holdConns"
	}
}

type adaptivePhase int

const (
	// phaseMeasure collects windows at the current count, then either grows
	// (when grow is set) or settles.
	phaseMeasure adaptivePhase = iota
	// phaseAdded judges the connection that was just added.
	phaseAdded
	// phaseParked judges a connection parked because the speed dropped.
	phaseParked
	// phasePruned judges the newest connection, parked by a prune test.
	phasePruned
	// phaseSteady watches for a drop and probes on a timer.
	phaseSteady
)

type adaptiveController struct {
	ceiling int
	phase   adaptivePhase
	grow    bool // phaseMeasure: add a connection once measured
	// keepBest (phaseMeasure): the count is one the controller has measured
	// before, so a lower result is a slowdown to react to, not the new best.
	keepBest bool

	// expect is the running count the controller's own decisions should
	// produce. Any other count was caused from outside (a connection ran out
	// of work, or an addition found nothing to split) and starts a new
	// measurement instead of being read as a slowdown.
	expect int

	samples []int64 // windows collected for the current judgement
	before  int64   // speed before the change being judged
	best    int64   // best mean speed at the current count

	lastWindow int64 // phaseSteady: the previous window, for a two-window mean
	lowWindows int   // phaseSteady: consecutive windows below the shrink line
	lowSum     int64

	// probeAt is when the next probe may start. It is zero until the first
	// settle, and a probe moves it a full interval on from its own start.
	probeAt time.Time

	// strikeAt is when a new connection was last refused. A second refusal
	// within strikeWindow sets the ceiling; one alone does not. A connection
	// added after it that runs clears it.
	strikeAt time.Time

	// intervals counts probe intervals; every adaptivePruneEvery-th is a
	// prune test instead of a probe.
	intervals int
	// holdCap is the count growth may not pass until holdUntil, after a
	// prune test found a connection that did not help. holdUntil is one probe
	// interval after the judgement, which is just after the next probe, so
	// the hold lasts about two probe intervals.
	holdCap   int
	holdUntil time.Time

	// resuming is set while the controller brings back a connection that a
	// prune test found needed. A refusal of that request is not a strike.
	resuming bool

	// The clock, read once so a running controller never sees it change.
	probeEvery   time.Duration
	strikeWindow time.Duration
}

func newAdaptiveController(ceiling int, now time.Time) *adaptiveController {
	if ceiling < 1 {
		ceiling = 1
	}
	return &adaptiveController{
		ceiling:      ceiling,
		phase:        phaseMeasure,
		grow:         true,
		expect:       1,
		probeEvery:   adaptiveProbeInterval,
		strikeWindow: adaptiveStrikeWindows * adaptiveTickInterval,
	}
}

// growLimit is the count growth may reach now: the ceiling, or less while a
// prune test's hold lasts.
func (c *adaptiveController) growLimit(now time.Time) int {
	if c.holdCap > 0 && now.Before(c.holdUntil) && c.holdCap < c.ceiling {
		return c.holdCap
	}
	return c.ceiling
}

// setCeiling changes the maximum, for example when the task's connections
// option is patched while it runs. Running connections above it are parked
// one per tick.
func (c *adaptiveController) setCeiling(n int) {
	if n < 1 {
		n = 1
	}
	c.ceiling = n
}

// tick is called once per window with the task speed (bytes/s) measured over
// that window and the number of running connections.
func (c *adaptiveController) tick(now time.Time, speed int64, active int) adaptiveDecision {
	if active > c.ceiling && active > 1 {
		// A lower ceiling makes a lower speed expected.
		c.measure(active-1, false, false)
		return parkSlowest
	}
	if active != c.expect {
		// Not our doing: measure the new count before judging anything.
		c.measure(active, false, false)
	}

	if c.phase == phaseSteady {
		return c.steady(now, speed, active)
	}

	c.samples = append(c.samples, speed)
	if len(c.samples) < adaptiveEvalWindows {
		return holdConns
	}
	mean := meanOf(c.samples)
	c.samples = c.samples[:0]

	switch c.phase {
	case phaseMeasure:
		if !c.keepBest || mean > c.best {
			c.best = mean
		}
		// A resumed connection that ran its measuring windows was not refused.
		c.resuming = false
		if c.grow && active < c.growLimit(now) {
			return c.add(mean, active)
		}
		c.settle(now)
		return holdConns

	case phaseAdded:
		// The added connection ran for two windows, so a refusal streak is
		// over.
		c.strikeAt = time.Time{}
		gain := gainOf(mean, c.before)
		bar := adaptiveKeepBar(active - 1)
		switch {
		case gain >= max(bar, adaptivePlateauGain):
			c.best = mean
			if active < c.growLimit(now) {
				return c.add(mean, active)
			}
			c.settle(now)
			return holdConns
		case gain >= min(bar, adaptivePlateauGain):
			// Kept, but growth waits for the next probe interval.
			c.best = mean
			c.settle(now)
			return holdConns
		default:
			// Plateau: the new connection did not help. Give it back and
			// measure the smaller count before settling. Its earlier speed
			// stays the best, so a slowdown that began meanwhile still shows.
			if active > 1 {
				c.best = c.before
				c.measure(active-1, false, true)
				return parkSlowest
			}
			c.settle(now)
			return holdConns
		}

	case phaseParked:
		if float64(mean) >= float64(c.before)*(1-adaptivePlateauGain) {
			// The parked connection was not needed. Keep walking down while
			// parking is free.
			c.best = mean
			if active > 1 {
				return c.park(mean, active)
			}
			c.settle(now)
			return holdConns
		}
		// This park cost speed: undo it and settle at the count before it.
		c.best = c.before
		if active < c.ceiling {
			c.measure(active+1, false, true)
			return addConn
		}
		c.settle(now)
		return holdConns

	case phasePruned:
		// active is n-1 now. The n-th connection's gain is measured as for an
		// addition: the speed with it against the speed without it. Below the
		// bar it would need to be added, it did not help.
		if gainOf(c.before, mean) < adaptiveKeepBar(active) {
			c.best = mean
			c.holdCap = active
			c.holdUntil = now.Add(c.probeEvery)
			c.settle(now)
			return holdConns
		}
		// It was needed: resume it.
		c.best = c.before
		if active < c.ceiling {
			c.measure(active+1, false, true)
			c.resuming = true
			return addConn
		}
		c.settle(now)
		return holdConns
	}
	return holdConns
}

// limited reports that a connection got a limiting HTTP status; the fetcher
// has parked it, and active is the running count after the park.
//
// With newConn, a 403 or 429 on a connection that never got data: the first
// refusal may be stray (for example a request the server has not yet seen
// end), so the controller measures and tries once more. A second refusal
// within adaptiveStrikeWindows windows means the server's limit is the count
// that worked, which becomes the ceiling.
//
// Otherwise a 429 or 503 hit a connection mid-stream: the next probe waits a
// full interval, and the lower count is measured afresh rather than read as
// a further slowdown.
func (c *adaptiveController) limited(now time.Time, active int, newConn bool) {
	if active < 1 {
		active = 1
	}
	if !newConn {
		c.probeAt = now.Add(c.probeEvery)
		c.measure(active, false, false)
		return
	}
	if c.resuming {
		// The request that resumes a pruned connection was refused. The server
		// may still count the request just cancelled, so this is not a
		// strike and the ceiling stays. The connection stays parked; the next
		// probe brings it back. The lower count is measured afresh so it is
		// not read as a slowdown.
		c.resuming = false
		c.measure(active, false, false)
		return
	}
	// A refused new connection changed nothing that runs, so the best stands.
	if !c.strikeAt.IsZero() && now.Sub(c.strikeAt) <= c.strikeWindow {
		c.strikeAt = time.Time{}
		if active < c.ceiling {
			c.ceiling = active
		}
		c.probeAt = now.Add(c.probeEvery)
		c.measure(active, false, true)
		return
	}
	c.strikeAt = now
	c.measure(active, true, true)
}

func (c *adaptiveController) steady(now time.Time, speed int64, active int) adaptiveDecision {
	if c.lastWindow > 0 {
		if pair := (c.lastWindow + speed) / 2; pair > c.best {
			c.best = pair
		}
	}
	prev := c.lastWindow
	c.lastWindow = speed

	if float64(speed) < float64(c.best)*(1-adaptiveShrinkDrop) {
		c.lowWindows++
		c.lowSum += speed
	} else {
		c.lowWindows, c.lowSum = 0, 0
	}
	if c.lowWindows >= adaptiveEvalWindows {
		low := c.lowSum / int64(c.lowWindows)
		c.lowWindows, c.lowSum = 0, 0
		if active > 1 {
			return c.park(low, active)
		}
		// One connection is already the minimum: accept the new speed.
		c.best = low
		return holdConns
	}

	// Probe intervals. The speed before a test is the mean of the last two
	// windows, so wait for a second window after settling.
	if prev > 0 && !now.Before(c.probeAt) {
		c.probeAt = now.Add(c.probeEvery)
		c.intervals++
		before := (prev + speed) / 2
		if c.intervals%adaptivePruneEvery == 0 && active > 1 {
			return c.prune(before, active)
		}
		if active < c.growLimit(now) {
			return c.add(before, active)
		}
	}
	return holdConns
}

func (c *adaptiveController) prune(before int64, active int) adaptiveDecision {
	c.phase = phasePruned
	c.before = before
	c.expect = active - 1
	c.samples = c.samples[:0]
	return parkNewest
}

func (c *adaptiveController) add(before int64, active int) adaptiveDecision {
	c.phase = phaseAdded
	c.before = before
	c.expect = active + 1
	c.samples = c.samples[:0]
	return addConn
}

func (c *adaptiveController) park(before int64, active int) adaptiveDecision {
	c.phase = phaseParked
	c.before = before
	c.expect = active - 1
	c.samples = c.samples[:0]
	return parkSlowest
}

func (c *adaptiveController) measure(expect int, grow, keepBest bool) {
	c.phase = phaseMeasure
	c.grow = grow
	c.keepBest = keepBest
	c.expect = expect
	c.samples = c.samples[:0]
}

func (c *adaptiveController) settle(now time.Time) {
	c.phase = phaseSteady
	c.lastWindow = 0
	c.lowWindows, c.lowSum = 0, 0
	if c.probeAt.IsZero() || c.probeAt.Before(now) {
		c.probeAt = now.Add(c.probeEvery)
	}
}

func meanOf(xs []int64) int64 {
	var sum int64
	for _, x := range xs {
		sum += x
	}
	return sum / int64(len(xs))
}

func gainOf(after, before int64) float64 {
	if before <= 0 {
		if after > 0 {
			return 1
		}
		return 0
	}
	return float64(after)/float64(before) - 1
}
