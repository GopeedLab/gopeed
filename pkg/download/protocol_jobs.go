package download

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/util"
)

var ErrProtocolJobNotFound = errors.New("protocol job not found")
var jobNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type protocolJobState struct {
	LastSuccess time.Time `json:"lastSuccess"`
	Failures    int       `json:"failures"`
}
type protocolJobRun struct {
	done   chan struct{}
	err    error
	source fetcher.TriggerSource
}
type registeredJob struct {
	protocol string
	spec     fetcher.CronJob
	mu       sync.Mutex
	state    protocolJobState
	active   *protocolJobRun
	next     time.Time
	wake     chan struct{}
	force    atomic.Bool
}
type protocolScheduler struct {
	d    *Downloader
	jobs map[string]*registeredJob
	wg   sync.WaitGroup
	// gate prevents adding work while shutdown waits for it.
	gate    sync.Mutex
	stopped bool
}

func (d *Downloader) setupProtocolJobs() error {
	scheduler := &protocolScheduler{d: d, jobs: map[string]*registeredJob{}}
	for _, fm := range d.cfg.FetchManagers {
		provider, ok := fm.(fetcher.CronProvider)
		if !ok {
			continue
		}
		name := fm.Name()
		host := fetcher.ProtocolHost{
			Read: func(dst any) error {
				value, err := d.readProtocolConfig(name)
				if err != nil {
					return err
				}
				return util.MapToStruct(value, dst)
			},
			UpdateState: func(change func(config, state json.RawMessage) (any, error)) error {
				return d.updateProtocolState(name, change)
			},
			Fetch: func(ctx context.Context, raw string, limit int64, validate func([]byte) error) ([]byte, error) {
				return d.fetchPublicResource(ctx, raw, limit, validate)
			},
		}
		for _, spec := range provider.CronJobs(host) {
			if !jobNamePattern.MatchString(spec.Name) || spec.Every <= 0 || spec.Timeout <= 0 || spec.Run == nil {
				return fmt.Errorf("invalid protocol job: %s/%s", name, spec.Name)
			}
			key := name + "/" + spec.Name
			if _, ok := scheduler.jobs[key]; ok {
				return fmt.Errorf("duplicate protocol job: %s", key)
			}
			entry := &registeredJob{protocol: name, spec: spec, wake: make(chan struct{}, 1)}
			if _, err := d.storage.Get(bucketProtocolJobs, key, &entry.state); err != nil {
				return err
			}
			scheduler.jobs[key] = entry
		}
	}
	d.protocolJobs = scheduler
	return nil
}
func (s *protocolScheduler) start() {
	for key, entry := range s.jobs {
		s.wg.Add(1)
		go s.loop(key, entry)
	}
}
func (s *protocolScheduler) signal(entry *registeredJob) {
	select {
	case entry.wake <- struct{}{}:
	default:
	}
}
func (s *protocolScheduler) changed(protocol string) {
	for _, entry := range s.jobs {
		if protocol == "" || entry.protocol == protocol {
			entry.force.Store(true)
			s.signal(entry)
		}
	}
}
func (s *protocolScheduler) loop(key string, entry *registeredJob) {
	defer s.wg.Done()
	entry.mu.Lock()
	if entry.next.IsZero() {
		entry.next = entry.state.LastSuccess.Add(entry.spec.Every)
		if entry.next.Before(time.Now()) {
			entry.next = time.Now().Add(time.Second)
		}
	}
	next := entry.next
	entry.mu.Unlock()
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()
	for {
		select {
		case <-s.d.backgroundCtx.Done():
			return
		case <-entry.wake:
			entry.mu.Lock()
			next = entry.next
			entry.mu.Unlock()
			if entry.force.Swap(false) {
				next = time.Now()
			}
		case <-timer.C:
			_ = s.trigger(s.d.backgroundCtx, key, fetcher.TriggerScheduled)
			entry.mu.Lock()
			next = entry.next
			entry.mu.Unlock()
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(max(time.Until(next), time.Millisecond))
	}
}
func (s *protocolScheduler) trigger(ctx context.Context, key string, source fetcher.TriggerSource) error {
	entry, ok := s.jobs[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrProtocolJobNotFound, key)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.gate.Lock()
		if s.stopped || s.d.closed.Load() {
			s.gate.Unlock()
			return errors.New("downloader is closed")
		}
		entry.mu.Lock()
		run := entry.active
		if run == nil {
			run = &protocolJobRun{done: make(chan struct{}), source: source}
			entry.active = run
			s.wg.Add(1)
			go s.execute(key, entry, run, source)
		}
		entry.mu.Unlock()
		s.gate.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-run.done:
			// A manual request must still run when the scheduled attempt it
			// joined was skipped (for example, automatic refresh is disabled).
			if source == fetcher.TriggerManual && run.source == fetcher.TriggerScheduled && errors.Is(run.err, fetcher.ErrJobSkipped) {
				continue
			}
			return run.err
		}
	}
}
func (s *protocolScheduler) execute(key string, entry *registeredJob, run *protocolJobRun, source fetcher.TriggerSource) {
	defer s.wg.Done()
	ctx, cancel := context.WithTimeout(s.d.backgroundCtx, entry.spec.Timeout)
	defer cancel()
	err := func() (err error) {
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("protocol job panic: %v", v)
			}
		}()
		return entry.spec.Run(ctx, source)
	}()
	if err == nil {
		err = ctx.Err()
	}
	entry.mu.Lock()
	switch {
	case err == nil:
		entry.state.LastSuccess = time.Now().UTC()
		entry.state.Failures = 0
	case errors.Is(err, fetcher.ErrJobSkipped), errors.Is(err, fetcher.ErrConfigChanged), errors.Is(err, context.Canceled):
	default:
		entry.state.Failures++
	}
	if storeErr := s.d.storage.Put(bucketProtocolJobs, key, entry.state); storeErr != nil {
		err = errors.Join(err, storeErr)
	}
	switch {
	case err == nil:
		entry.next = entry.state.LastSuccess.Add(entry.spec.Every)
	case errors.Is(err, fetcher.ErrJobSkipped):
		entry.next = time.Now().Add(entry.spec.Every)
	case errors.Is(err, fetcher.ErrConfigChanged):
		entry.next = time.Now().Add(time.Second)
	default:
		retry := time.Minute * time.Duration(1<<min(max(entry.state.Failures-1, 0), 6))
		entry.next = time.Now().Add(retry)
	}
	run.err = err
	entry.active = nil
	close(run.done)
	entry.mu.Unlock()
	if err != nil && !errors.Is(err, fetcher.ErrJobSkipped) && !errors.Is(err, context.Canceled) {
		s.d.Logger.Warn().Err(err).Str("job", key).Str("source", string(source)).Msg("protocol job failed")
	}
	if source == fetcher.TriggerManual {
		s.signal(entry)
	}
}
func (s *protocolScheduler) close() {
	s.gate.Lock()
	s.stopped = true
	s.gate.Unlock()
	s.wg.Wait()
}
func (d *Downloader) TriggerProtocolJob(ctx context.Context, protocol, name string) error {
	if d.protocolJobs == nil {
		return ErrProtocolJobNotFound
	}
	err := d.protocolJobs.trigger(ctx, protocol+"/"+name, fetcher.TriggerManual)
	if errors.Is(err, fetcher.ErrJobSkipped) {
		return nil
	}
	return err
}
