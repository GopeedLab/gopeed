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
	// UploadLimit is the upload speed limit in bytes per second, 0 means unlimited.
	UploadLimit int64 `json:"uploadLimit"`
	// DisableDHT and DisablePEX turn off peer discovery for public torrents, private torrents never use it.
	DisableDHT bool `json:"disableDht"`
	DisablePEX bool `json:"disablePex"`
}
