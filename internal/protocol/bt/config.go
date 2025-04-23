package bt

type config struct {
	ListenPort int      `json:"listenPort"`
	Trackers   []string `json:"trackers"`
	// SeedKeep is always keep seeding after downloading is complete, unless manually stopped.
	SeedKeep bool `json:"seedKeep"`
	// SeedRatio is the ratio of uploaded data to downloaded data to seed.
	SeedRatio float64 `json:"seedRatio"`
	// SeedTime is the time in seconds to seed after downloading is complete.
	SeedTime int64 `json:"seedTime"`
}
