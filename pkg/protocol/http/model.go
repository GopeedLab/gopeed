package http

type ReqExtra struct {
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type OptsExtra struct {
	Connections int `json:"connections"`
	// AutoTorrent when task download complete, and it is a .torrent file, it will be auto create a new task for the torrent file
	// nil means use global config, true/false means explicit setting
	AutoTorrent *bool `json:"autoTorrent"`
	// DeleteTorrentAfterDownload when true, deletes the .torrent file after creating BT task
	// nil means use global config, true/false means explicit setting
	DeleteTorrentAfterDownload *bool `json:"deleteTorrentAfterDownload"`
	// AutoExtract when task download complete, and it is an archive file, it will be auto extracted
	// nil means use global config, true/false means explicit setting
	AutoExtract *bool `json:"autoExtract"`
	// ArchivePassword is the password for extracting password-protected archives
	ArchivePassword string `json:"archivePassword"`
	// DeleteAfterExtract when true, deletes the archive file after successful extraction
	DeleteAfterExtract bool `json:"deleteAfterExtract"`
}

// Stats for download
type Stats struct {
	Connections []*StatsConnection `json:"connections"`
	Sha256      string             `json:"sha256"`
	Crc32       string             `json:"crc32"`
	// FileSize is the actual size of the downloaded file
	FileSize int64 `json:"fileSize"`
	// ExpectedSize is the expected size from the server
	ExpectedSize int64 `json:"expectedSize"`
	// IntegrityVerified is true if integrity check was performed
	IntegrityVerified bool `json:"integrityVerified"`
}

type StatsConnection struct {
	Downloaded int64 `json:"downloaded"`
	Completed  bool  `json:"completed"`
	Failed     bool  `json:"failed"`
	RetryTimes int   `json:"retryTimes"`
}
