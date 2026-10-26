package bt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
)

var defaultTrackerSubscribeURLs = []string{
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_http.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_https.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_ip.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_udp.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_ws.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt",
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best_ip.txt",
	"https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/all.txt",
	"https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/best.txt",
	"https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/http.txt",
}

func validTracker(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return false
	}
	switch u.Scheme {
	case "udp", "http", "https", "ws", "wss":
		return true
	}
	return false
}
func uniqueTrackers(groups ...[]string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, group := range groups {
		for _, raw := range group {
			value := strings.TrimSpace(raw)
			if value != "" && !seen[value] {
				seen[value] = true
				result = append(result, value)
			}
		}
	}
	return result
}

// effectiveTrackers derives the client list without persisting duplicate lists.
func (c *config) effectiveTrackers() []string {
	groups := [][]string{c.CustomTrackers}
	for _, source := range c.TrackerSubscribeURLs {
		groups = append(groups, c.State.TrackerSources[source])
	}
	return uniqueTrackers(groups...)
}
func (fm *FetcherManager) NormalizeConfig(raw json.RawMessage) (any, error) {
	c := fm.DefaultConfig().(*config)
	if err := json.Unmarshal(raw, c); err != nil {
		return nil, err
	}
	if c.ListenPort < 0 || c.ListenPort > 65535 || c.SeedRatio < 0 || c.SeedTime < 0 {
		return nil, errors.New("invalid BT configuration")
	}
	if len(c.TrackerSubscribeURLs) > 64 {
		return nil, errors.New("too many tracker subscriptions")
	}
	c.TrackerSubscribeURLs = uniqueTrackers(c.TrackerSubscribeURLs)
	for _, source := range c.TrackerSubscribeURLs {
		u, err := url.Parse(source)
		if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
			return nil, errors.New("invalid tracker subscription URL")
		}
	}
	c.CustomTrackers = uniqueTrackers(c.CustomTrackers)
	for _, tracker := range c.CustomTrackers {
		if !validTracker(tracker) {
			return nil, fmt.Errorf("invalid tracker URL: %s", tracker)
		}
	}
	if c.State.TrackerSources == nil {
		c.State.TrackerSources = map[string][]string{}
	}
	for source := range c.State.TrackerSources {
		if !slices.Contains(c.TrackerSubscribeURLs, source) {
			delete(c.State.TrackerSources, source)
		}
	}
	if c.State.TrackerSubscriptionStatuses == nil {
		c.State.TrackerSubscriptionStatuses = map[string]trackerSubscriptionStatus{}
	}
	for source := range c.State.TrackerSubscriptionStatuses {
		if !slices.Contains(c.TrackerSubscribeURLs, source) {
			delete(c.State.TrackerSubscriptionStatuses, source)
		}
	}
	return c, nil
}
func (fm *FetcherManager) CronJobs(host fetcher.ProtocolHost) []fetcher.CronJob {
	return []fetcher.CronJob{{
		Name:    "update-trackers",
		Every:   24 * time.Hour,
		Timeout: 90 * time.Second,
		Run: func(ctx context.Context, source fetcher.TriggerSource) error {
			return fm.updateTrackers(ctx, host, source)
		},
	}}
}
func parseTrackers(data []byte) ([]string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 64*1024)
	result := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if validTracker(line) {
			result = append(result, line)
		}
		if len(result) > 20000 {
			return nil, errors.New("too many trackers")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("subscription contains no valid trackers")
	}
	return uniqueTrackers(result), nil
}
func (fm *FetcherManager) updateTrackers(ctx context.Context, host fetcher.ProtocolHost, source fetcher.TriggerSource) error {
	snapshot, err := fetcher.ReadConfig[config](host)
	if err != nil {
		return err
	}
	if (source == fetcher.TriggerScheduled && !snapshot.AutoUpdateTrackers) || len(snapshot.TrackerSubscribeURLs) == 0 {
		return fetcher.ErrJobSkipped
	}
	fetched := map[string][]string{}
	statuses := map[string]trackerSubscriptionStatus{}
	var failures []error
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, 4)
	for _, address := range snapshot.TrackerSubscribeURLs {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				return
			}
			data, err := host.Fetch(ctx, address, 2<<20, func(body []byte) error { _, err := parseTrackers(body); return err })
			var trackers []string
			if err == nil {
				trackers, err = parseTrackers(data)
			}
			mu.Lock()
			defer mu.Unlock()
			statuses[address] = trackerSubscriptionStatus{Success: err == nil, TrackerCount: len(trackers), LastFetchTime: time.Now().UTC()}
			if err != nil {
				failures = append(failures, fmt.Errorf("subscription %s: %w", address, err))
			} else {
				fetched[address] = trackers
			}
		}(address)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		failures = append(failures, err)
	}
	if len(statuses) == 0 {
		return errors.Join(failures...)
	}
	err = fetcher.UpdateState[config, state](host, func(current *config, state *state) error {
		if !slices.Equal(current.TrackerSubscribeURLs, snapshot.TrackerSubscribeURLs) {
			return fetcher.ErrConfigChanged
		}
		if state.TrackerSources == nil {
			state.TrackerSources = map[string][]string{}
		}
		if state.TrackerSubscriptionStatuses == nil {
			state.TrackerSubscriptionStatuses = map[string]trackerSubscriptionStatus{}
		}
		for address, status := range statuses {
			state.TrackerSubscriptionStatuses[address] = status
		}
		for address, trackers := range fetched {
			state.TrackerSources[address] = trackers
		}
		if len(failures) == 0 {
			state.LastTrackerUpdateTime = time.Now().UTC()
		}
		return nil
	})
	if err != nil {
		return err
	}
	current, err := fetcher.ReadConfig[config](host)
	if err != nil {
		return err
	}
	fm.applyTrackers(current.effectiveTrackers())
	return errors.Join(failures...)
}
func (fm *FetcherManager) applyTrackers(trackers []string) {
	lock.Lock()
	defer lock.Unlock()
	if client == nil {
		return
	}
	announce := make([][]string, 0, len(trackers))
	for _, tracker := range trackers {
		announce = append(announce, []string{tracker})
	}
	for _, t := range client.Torrents() {
		info := t.Info()
		if info == nil || (info.Private != nil && *info.Private) {
			continue
		}
		t.AddTrackers(announce)
	}
}
