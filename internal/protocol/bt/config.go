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
	// External hands new torrent and magnet downloads to an rtorrent instance
	// instead of the built-in engine.
	External externalConfig `json:"external"`
}

// externalConfig is the rtorrent hand-off. Each task remembers the engine it
// started with, so changing this switch never moves an existing task.
type externalConfig struct {
	Enable bool `json:"enable"`
	// URL is rtorrent's XML-RPC endpoint: http://host/RPC2, https://host/RPC2,
	// scgi://host:port, or scgi:///path/to/rpc.socket for a Unix socket.
	URL string `json:"url"`
	// Username and Password are sent as HTTP basic auth, for http and https.
	Username string `json:"username"`
	Password string `json:"password"`
	// Directory is the download folder as rtorrent sees it. Empty uses the
	// task's own folder, which fits when both run on the same machine.
	Directory string `json:"directory"`
}
