package download

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	btProtocol "github.com/GopeedLab/gopeed/internal/protocol/bt"
	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// State tests initialize config without starting network clients or cron loops.
func protocolConfigDownloader(t *testing.T, storage Storage, managers ...fetcher.FetcherManager) *Downloader {
	t.Helper()
	if storage == nil {
		storage = NewMemStorage()
	}
	d := NewDownloader(&DownloaderConfig{Storage: storage, StorageDir: t.TempDir(), FetchManagers: managers})
	d.cfg.DownloaderStoreConfig = &base.DownloaderStoreConfig{}
	if err := storage.Setup([]string{bucketConfig}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.cancelBackground(); _ = storage.Close() })
	if err := d.initProtocolConfigs(); err != nil {
		t.Fatal(err)
	}
	return d
}
func protocolObject(t *testing.T, d *Downloader, name string) map[string]any {
	t.Helper()
	cfg, err := d.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg.ProtocolConfig[name])
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func configureBT(t *testing.T, d *Downloader, urls ...string) {
	t.Helper()
	cfg, _ := d.GetConfig()
	cfg.ProtocolConfig["bt"] = map[string]any{"autoUpdateTrackers": false, "trackerSubscribeUrls": urls, "customTrackers": []string{"udp://custom.example:80/announce"}}
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
}
func seedBTState(t *testing.T, d *Downloader) {
	t.Helper()
	if err := d.updateProtocolState("bt", func(_, _ json.RawMessage) (any, error) {
		return map[string]any{
			"trackerSources":              map[string][]string{"https://example.com/list": {"udp://cached.example:80/announce"}},
			"trackerSubscriptionStatuses": map[string]any{"https://example.com/list": map[string]any{"success": true, "trackerCount": 1, "lastFetchTime": time.Date(2026, 10, 19, 1, 2, 3, 0, time.UTC)}},
			"lastTrackerUpdateTime":       time.Date(2026, 10, 19, 1, 2, 3, 0, time.UTC),
		}, nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestFullConfigValidatesProtocolsAndPreservesExplicitValues(t *testing.T) {
	d := protocolConfigDownloader(t, nil, &httpProtocol.FetcherManager{}, &btProtocol.FetcherManager{})
	configureBT(t, d)
	cfg, _ := d.GetConfig()
	cfg.ProtocolConfig["http"] = map[string]any{"connections": 0, "useServerCtime": false}
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got := protocolObject(t, d, "http")
	if got["connections"] != float64(0) || got["useServerCtime"] != false {
		t.Fatal(got)
	}
	for _, invalid := range []any{map[string]any{"connections": "bad"}, map[string]any{"useServerCtime": "bad"}} {
		cfg, _ := d.GetConfig()
		cfg.ProtocolConfig["http"] = invalid
		if err := d.PutConfig(cfg); err == nil {
			t.Fatalf("invalid protocol config accepted: %v", invalid)
		}
	}
	cfg, _ = d.GetConfig()
	cfg.DownloadDir = "must-not-publish"
	cfg.ProtocolConfig["bt"] = map[string]any{"listenPort": -1}
	if err := d.PutConfig(cfg); err == nil {
		t.Fatal("invalid BT config accepted")
	}
	saved, _ := d.GetConfig()
	if saved.DownloadDir == "must-not-publish" {
		t.Fatal("invalid full PUT partially published")
	}
	cfg, _ = d.GetConfig()
	cfg.ProtocolConfig["unknown"] = map[string]any{"state": map[string]any{"injected": true}}
	if err := d.PutConfig(cfg); err == nil {
		t.Fatal("unregistered protocol accepted")
	}
	// Returned configs are detached snapshots.
	snapshot, _ := d.GetConfig()
	snapshot.ProtocolConfig["http"] = map[string]any{"connections": 999}
	if !reflect.DeepEqual(protocolObject(t, d, "http"), got) {
		t.Fatal("snapshot changed live config")
	}
}

func TestProtocolConfigIgnoresUnknownFields(t *testing.T) {
	for _, mode := range []string{"stored", "put"} {
		t.Run(mode, func(t *testing.T) {
			d := protocolConfigDownloader(t, nil, &btProtocol.FetcherManager{})
			cfg, _ := d.GetConfig()
			cfg.ProtocolConfig["bt"] = map[string]any{
				"listenPort":         6881,
				"autoUpdateTrackers": false,
				"trackers":           []string{"udp://obsolete.example:80/announce"},
				"state":              map[string]any{"obsolete": true},
			}
			if mode == "stored" {
				if err := d.storage.Put(bucketConfig, "config", cfg); err != nil {
					t.Fatal(err)
				}
				var restored base.DownloaderStoreConfig
				if exists, err := d.storage.Get(bucketConfig, "config", &restored); err != nil || !exists {
					t.Fatalf("load stored config: %v %v", exists, err)
				}
				d.cfg.DownloaderStoreConfig = &restored
				if err := d.initProtocolConfigs(); err != nil {
					t.Fatalf("load config containing removed fields: %v", err)
				}
			} else if err := d.PutConfig(cfg); err != nil {
				t.Fatalf("save config containing unknown fields: %v", err)
			}
			actual := protocolObject(t, d, "bt")
			if actual["listenPort"] != float64(6881) || actual["autoUpdateTrackers"] != false {
				t.Fatalf("known fields changed: %v", actual)
			}
			if _, exists := actual["trackers"]; exists {
				t.Fatal("removed field retained")
			}
			if len(actual["customTrackers"].([]any)) != 0 {
				t.Fatal("removed trackers migrated into custom trackers")
			}
			if _, exists := actual["state"].(map[string]any)["obsolete"]; exists {
				t.Fatal("unknown state field retained")
			}
		})
	}
}

func TestBTStateSurvivesOmittedNullForgedAndStaleFullConfig(t *testing.T) {
	for _, mode := range []string{"omitted", "null", "forged", "wrong-type", "stale", "omitted-protocol", "omitted-timestamp", "case-variant", "unicode-variant"} {
		t.Run(mode, func(t *testing.T) {
			d := protocolConfigDownloader(t, nil, &btProtocol.FetcherManager{})
			configureBT(t, d, "https://example.com/list")
			stale, _ := d.GetConfig()
			seedBTState(t, d)
			expected := protocolObject(t, d, "bt")["state"]
			cfg, _ := d.GetConfig()
			bt := protocolObject(t, d, "bt")
			switch mode {
			case "omitted":
				delete(bt, "state")
			case "null":
				bt["state"] = nil
			case "forged":
				bt["state"] = map[string]any{"lastTrackerUpdateTime": "2000-01-01T00:00:00Z", "trackerSources": map[string]any{}, "unknown": true}
			case "wrong-type":
				bt["state"] = "ignore this before decoding"
			case "stale":
				cfg = stale
				bt = map[string]any{"autoUpdateTrackers": false, "trackerSubscribeUrls": []string{"https://example.com/list"}, "customTrackers": []string{"udp://custom.example:80/announce"}, "state": map[string]any{}}
			case "omitted-protocol":
				delete(cfg.ProtocolConfig, "bt")
			case "case-variant":
				bt["STATE"] = "ignore this alias"
			case "unicode-variant":
				bt["ſtate"] = "ignore Unicode case folding"
			case "omitted-timestamp":
				bt["state"] = map[string]any{"trackerSources": map[string]any{}}
			}
			if mode != "omitted-protocol" {
				bt["listenPort"] = 6881
				cfg.ProtocolConfig["bt"] = bt
			}
			if err := d.PutConfig(cfg); err != nil {
				t.Fatal(err)
			}
			actual := protocolObject(t, d, "bt")
			if !reflect.DeepEqual(actual["state"], expected) {
				t.Fatalf("backend state replaced: %v", actual)
			}
			if _, exists := actual["trackers"]; exists {
				t.Fatal("derived tracker list persisted")
			}
			if mode != "omitted-protocol" && actual["listenPort"] != float64(6881) {
				t.Fatal("writable config did not change")
			}
		})
	}
}

func TestBTConfigRemovalPrunesOnlyRemovedState(t *testing.T) {
	d := protocolConfigDownloader(t, nil, &btProtocol.FetcherManager{})
	configureBT(t, d, "https://example.com/list")
	seedBTState(t, d)
	configureBT(t, d, "https://new.example/list")
	state := protocolObject(t, d, "bt")["state"].(map[string]any)
	if len(state["trackerSources"].(map[string]any)) != 0 || len(state["trackerSubscriptionStatuses"].(map[string]any)) != 0 {
		t.Fatal("removed subscription retained state")
	}
}

type counterState struct {
	Count int `json:"count"`
	Other int `json:"other"`
}
type counterConfig struct {
	fetcher.ConfigState[counterState]
	Writable int `json:"writable"`
}
type counterManager struct{ httpProtocol.FetcherManager }

func (*counterManager) Name() string       { return "counter" }
func (*counterManager) DefaultConfig() any { return &counterConfig{Writable: 7} }
func counterHost(d *Downloader) fetcher.ProtocolHost {
	return fetcher.ProtocolHost{UpdateState: func(change func(config, state json.RawMessage) (any, error)) error {
		return d.updateProtocolState("counter", change)
	}}
}

func TestProtocolUpdateStateCommitsOnlyStateAndUsesLatestConfig(t *testing.T) {
	d := protocolConfigDownloader(t, nil, &counterManager{})
	host := counterHost(d)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fetcher.UpdateState[counterConfig, counterState](host, func(cfg *counterConfig, state *counterState) error { cfg.Writable = 999; state.Count++; return nil }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	actual := protocolObject(t, d, "counter")
	if actual["writable"] != float64(7) || actual["state"].(map[string]any)["count"] != float64(20) {
		t.Fatal(actual)
	}
	// Rejecting a callback must not publish its partial mutation.
	err := fetcher.UpdateState[counterConfig, counterState](host, func(_ *counterConfig, state *counterState) error { state.Count = 1000; return errors.New("reject") })
	if err == nil || !reflect.DeepEqual(protocolObject(t, d, "counter"), actual) {
		t.Fatal("failed callback published state")
	}
	if err := d.updateProtocolState("counter", func(_, _ json.RawMessage) (any, error) { return map[string]any{"count": "bad"}, nil }); err == nil {
		t.Fatal("invalid state accepted")
	}
}

func TestProtocolStateAndFullPutDoNotLoseConcurrentUpdates(t *testing.T) {
	d := protocolConfigDownloader(t, nil, &counterManager{})
	snapshot, _ := d.GetConfig()
	snapshot.ProtocolConfig["counter"] = map[string]any{"writable": 21}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := d.PutConfig(snapshot); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := fetcher.UpdateState[counterConfig, counterState](counterHost(d), func(cfg *counterConfig, state *counterState) error { state.Count++; return nil }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	actual := protocolObject(t, d, "counter")
	if actual["writable"] != float64(21) || actual["state"].(map[string]any)["count"] != float64(20) {
		t.Fatal(actual)
	}
}

func TestBTSubscriptionStatusesSurviveStorageRestart(t *testing.T) {
	dir := t.TempDir()
	storage := NewBoltStorage(dir)
	d := protocolConfigDownloader(t, storage, &btProtocol.FetcherManager{})
	configureBT(t, d, "https://example.com/list")
	seedBTState(t, d)
	expected := protocolObject(t, d, "bt")["state"]
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := NewBoltStorage(dir)
	if err := reopened.Setup([]string{bucketConfig}); err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var persisted base.DownloaderStoreConfig
	if exists, err := reopened.Get(bucketConfig, "config", &persisted); err != nil || !exists {
		t.Fatalf("reload persisted config: %v %v", exists, err)
	}
	raw, _ := json.Marshal(persisted.ProtocolConfig["bt"])
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg["state"], expected) {
		t.Fatalf("state lost after reopening storage: %+v", cfg)
	}
}

type rejectedConfigStorage struct{ Storage }

func (*rejectedConfigStorage) Put(string, string, any) error {
	return errors.New("storage unavailable")
}

func TestProtocolStateAndPutStorageFailureDoNotPublish(t *testing.T) {
	d := protocolConfigDownloader(t, nil, &counterManager{})
	before := protocolObject(t, d, "counter")
	d.storage = &rejectedConfigStorage{Storage: d.storage}
	err := fetcher.UpdateState[counterConfig, counterState](counterHost(d), func(_ *counterConfig, state *counterState) error { state.Count = 99; return nil })
	if err == nil || !reflect.DeepEqual(protocolObject(t, d, "counter"), before) {
		t.Fatal("failed state persistence published a change")
	}
	cfg, _ := d.GetConfig()
	cfg.ProtocolConfig["counter"] = map[string]any{"writable": 99}
	if err := d.PutConfig(cfg); err == nil || !reflect.DeepEqual(protocolObject(t, d, "counter"), before) {
		t.Fatal("failed config persistence published a change")
	}
}
