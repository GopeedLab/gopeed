package bt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
)

func trackerTestHost(t *testing.T, c *config, fetch func(context.Context, string, int64, func([]byte) error) ([]byte, error)) (fetcher.ProtocolHost, *sync.Mutex) {
	t.Helper()
	mu := &sync.Mutex{}
	return fetcher.ProtocolHost{
		Read: func(dst any) error {
			mu.Lock()
			defer mu.Unlock()
			raw, err := json.Marshal(c)
			if err != nil {
				return err
			}
			return json.Unmarshal(raw, dst)
		},
		UpdateState: func(change func(config, state json.RawMessage) (any, error)) error {
			mu.Lock()
			defer mu.Unlock()
			raw, _ := json.Marshal(c)
			currentState, _ := json.Marshal(c.State)
			next, err := change(raw, currentState)
			if err != nil {
				return err
			}
			raw, err = json.Marshal(next)
			if err != nil {
				return err
			}
			return json.Unmarshal(raw, &c.State)
		},
		Fetch: fetch,
	}, mu
}
func TestTrackerRefreshPartialSuccessUsesLatestConfig(t *testing.T) {
	fm := &FetcherManager{}
	cfg := fm.DefaultConfig().(*config)
	cfg.TrackerSubscribeURLs = []string{"https://a.example/list", "https://b.example/list"}
	cfg.CustomTrackers = []string{"udp://custom.example:80/announce"}
	cfg.State.TrackerSources = map[string][]string{cfg.TrackerSubscribeURLs[0]: {"udp://old.example:80/announce"}, cfg.TrackerSubscribeURLs[1]: {"udp://cached.example:80/announce"}}
	host, mu := trackerTestHost(t, cfg, nil)
	host.Fetch = func(ctx context.Context, address string, _ int64, validate func([]byte) error) ([]byte, error) {
		if address == "https://b.example/list" {
			return nil, errors.New("offline")
		}
		// Simulate a frontend update while the network request is in flight.
		mu.Lock()
		cfg.CustomTrackers = []string{"udp://new-custom.example:80/announce"}
		mu.Unlock()
		data := []byte("udp://fresh.example:80/announce\n# ignored\nudp://fresh.example:80/announce\n")
		return data, validate(data)
	}
	err := fm.updateTrackers(context.Background(), host, fetcher.TriggerManual)
	if err == nil {
		t.Fatal("partial failure must be reported")
	}
	want := []string{"udp://new-custom.example:80/announce", "udp://fresh.example:80/announce", "udp://cached.example:80/announce"}
	if !reflect.DeepEqual(cfg.effectiveTrackers(), want) {
		t.Fatalf("got %v", cfg.effectiveTrackers())
	}
	success := cfg.State.TrackerSubscriptionStatuses["https://a.example/list"]
	failure := cfg.State.TrackerSubscriptionStatuses["https://b.example/list"]
	if !success.Success || success.TrackerCount != 1 || success.LastFetchTime.IsZero() || failure.Success || failure.TrackerCount != 0 || failure.LastFetchTime.IsZero() {
		t.Fatalf("unexpected subscription statuses: %+v", cfg.State.TrackerSubscriptionStatuses)
	}
	if !cfg.State.LastTrackerUpdateTime.IsZero() {
		t.Fatal("partial refresh marked fully successful")
	}
}
func TestTrackerRefreshDiscardsChangedSubscriptions(t *testing.T) {
	fm := &FetcherManager{}
	cfg := fm.DefaultConfig().(*config)
	cfg.TrackerSubscribeURLs = []string{"https://a.example/list"}
	host, mu := trackerTestHost(t, cfg, nil)
	host.Fetch = func(context.Context, string, int64, func([]byte) error) ([]byte, error) {
		mu.Lock()
		cfg.TrackerSubscribeURLs = []string{}
		mu.Unlock()
		return []byte("udp://fresh.example:80/announce"), nil
	}
	if err := fm.updateTrackers(context.Background(), host, fetcher.TriggerManual); !errors.Is(err, fetcher.ErrConfigChanged) {
		t.Fatal(err)
	}
	if len(cfg.State.TrackerSources) != 0 {
		t.Fatal("outdated source was committed")
	}
}
func TestTrackerRefreshManualWorksWhenAutomaticDisabled(t *testing.T) {
	fm := &FetcherManager{}
	cfg := fm.DefaultConfig().(*config)
	cfg.AutoUpdateTrackers = false
	cfg.TrackerSubscribeURLs = []string{"https://a.example/list"}
	calls := 0
	host, _ := trackerTestHost(t, cfg, func(context.Context, string, int64, func([]byte) error) ([]byte, error) {
		calls++
		return []byte("udp://fresh.example:80/announce"), nil
	})
	if err := fm.updateTrackers(context.Background(), host, fetcher.TriggerScheduled); !errors.Is(err, fetcher.ErrJobSkipped) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("scheduled job fetched while disabled")
	}
	if err := fm.updateTrackers(context.Background(), host, fetcher.TriggerManual); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || cfg.State.LastTrackerUpdateTime.IsZero() {
		t.Fatal("manual refresh did not succeed")
	}
}
func TestTrackerConfigPreservesExplicitSettings(t *testing.T) {
	value, err := (&FetcherManager{}).NormalizeConfig(json.RawMessage(`{
		"listenPort":6881, "trackerSubscribeUrls":[], "autoUpdateTrackers":false,
		"customTrackers":["udp://custom.example:80/announce"]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg := value.(*config)
	if cfg.AutoUpdateTrackers || len(cfg.TrackerSubscribeURLs) != 0 || cfg.ListenPort != 6881 {
		t.Fatalf("%+v", cfg)
	}
	if !reflect.DeepEqual(cfg.effectiveTrackers(), cfg.CustomTrackers) {
		t.Fatalf("derived trackers: %v", cfg.effectiveTrackers())
	}
}

func TestParseTrackerList(t *testing.T) {
	got, err := parseTrackers([]byte("\ufeffudp://tracker.example:80/announce\r\n\n<html>error</html>\nhttps://tracker.example/announce\n"))
	if err != nil || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := parseTrackers([]byte("<html>mirror error</html>")); err == nil {
		t.Fatal("invalid mirror body accepted")
	}
}
func TestTrackerDefaultsRegisterDailyJob(t *testing.T) {
	jobs := (&FetcherManager{}).CronJobs(fetcher.ProtocolHost{})
	if len(jobs) != 1 || jobs[0].Name != "update-trackers" || jobs[0].Every != 24*time.Hour {
		t.Fatal(jobs)
	}
}

func TestTrackerRefreshAllFailuresPersistStatusesAndKeepCache(t *testing.T) {
	fm := &FetcherManager{}
	cfg := fm.DefaultConfig().(*config)
	address := "https://custom.example/trackers.txt"
	cfg.TrackerSubscribeURLs = []string{address}
	cfg.State.TrackerSources[address] = []string{"udp://cached.example:80/announce"}
	cfg.State.TrackerSubscriptionStatuses[address] = trackerSubscriptionStatus{Success: true, TrackerCount: 1, LastFetchTime: time.Now().Add(-time.Hour)}
	host, _ := trackerTestHost(t, cfg, func(context.Context, string, int64, func([]byte) error) ([]byte, error) {
		return nil, errors.New("offline")
	})
	before := time.Now()
	if err := fm.updateTrackers(context.Background(), host, fetcher.TriggerManual); err == nil {
		t.Fatal("failure was not reported")
	}
	status, ok := cfg.State.TrackerSubscriptionStatuses[address]
	if !ok || status.Success || status.TrackerCount != 0 || status.LastFetchTime.Before(before) {
		t.Fatalf("failed fetch status not stored: %+v", status)
	}
	if !reflect.DeepEqual(cfg.State.TrackerSources[address], []string{"udp://cached.example:80/announce"}) {
		t.Fatal("last good cache lost")
	}
	if !reflect.DeepEqual(cfg.effectiveTrackers(), cfg.State.TrackerSources[address]) {
		t.Fatal("cached trackers not retained")
	}
}

func TestTrackerConfigPrunesRemovedSubscriptions(t *testing.T) {
	fm := &FetcherManager{}
	cfg := fm.DefaultConfig().(*config)
	address := "https://custom.example/trackers.txt"
	cfg.TrackerSubscribeURLs = []string{}
	cfg.State.TrackerSources[address] = []string{"udp://cached.example:80/announce"}
	cfg.State.TrackerSubscriptionStatuses[address] = trackerSubscriptionStatus{Success: true, TrackerCount: 1, LastFetchTime: time.Now()}
	raw, _ := json.Marshal(cfg)
	value, err := fm.NormalizeConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	next := value.(*config)
	if len(next.State.TrackerSources) != 0 || len(next.State.TrackerSubscriptionStatuses) != 0 || len(next.effectiveTrackers()) != 0 {
		t.Fatal("removed subscription still owns cached results")
	}
}

func TestTrackerRefreshTimeoutPersistsCompletedAttemptStatus(t *testing.T) {
	fm := &FetcherManager{}
	cfg := fm.DefaultConfig().(*config)
	address := "https://custom.example/trackers.txt"
	cfg.TrackerSubscribeURLs = []string{address}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	host, _ := trackerTestHost(t, cfg, func(context.Context, string, int64, func([]byte) error) ([]byte, error) {
		cancel()
		return nil, context.DeadlineExceeded
	})
	if err := fm.updateTrackers(ctx, host, fetcher.TriggerManual); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	status, ok := cfg.State.TrackerSubscriptionStatuses[address]
	if !ok || status.Success || status.TrackerCount != 0 || status.LastFetchTime.IsZero() {
		t.Fatalf("timeout status lost: %+v", status)
	}
}
