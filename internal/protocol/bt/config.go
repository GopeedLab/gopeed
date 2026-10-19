package bt

import (
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
)

type trackerSubscriptionStatus struct {
	Success       bool      `json:"success"`
	TrackerCount  int       `json:"trackerCount"`
	LastFetchTime time.Time `json:"lastFetchTime"`
}

type state struct {
	TrackerSubscriptionStatuses map[string]trackerSubscriptionStatus `json:"trackerSubscriptionStatuses"`
	TrackerSources              map[string][]string                  `json:"trackerSources"`
	LastTrackerUpdateTime       time.Time                            `json:"lastTrackerUpdateTime"`
}

type config struct {
	fetcher.ConfigState[state]
	TrackerSubscribeURLs []string `json:"trackerSubscribeUrls"`
	AutoUpdateTrackers   bool     `json:"autoUpdateTrackers"`
	CustomTrackers       []string `json:"customTrackers"`
	ListenPort           int      `json:"listenPort"`
	// SeedKeep is always keep seeding after downloading is complete, unless manually stopped.
	SeedKeep bool `json:"seedKeep"`
	// SeedRatio is the ratio of uploaded data to downloaded data to seed.
	SeedRatio float64 `json:"seedRatio"`
	// SeedTime is the time in seconds to seed after downloading is complete.
	SeedTime int64 `json:"seedTime"`
	// DisableDHT and DisablePEX turn off DHT and PEX peer discovery. Like ListenPort,
	// they are read when the shared client is built.
	DisableDHT bool `json:"disableDht"`
	DisablePEX bool `json:"disablePex"`
}
