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
	// LocalRtorrent is the endpoint of a local rtorrent found without any
	// setting (Rtorrent 16 on a QNAP NAS), or empty. See qnap.go.
	LocalRtorrent string `json:"localRtorrent"`
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
	// Enable is the user's switch. Unset (null) means automatic: on when a
	// local rtorrent is detected (qnap.go), off otherwise.
	Enable *bool `json:"enable"`
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

// target says whether new torrents go to rtorrent and at which address. An
// address the user set wins over a detected one.
func (e *externalConfig) target() (bool, string) {
	detected := ""
	if e.Enable == nil || e.URL == "" {
		detected = detectLocalRtorrent()
	}
	on := detected != ""
	if e.Enable != nil {
		on = *e.Enable
	}
	endpoint := e.URL
	if endpoint == "" {
		endpoint = detected
	}
	return on, endpoint
}
