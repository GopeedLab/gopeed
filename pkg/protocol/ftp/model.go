package ftp

// TLS modes for the "tls" option of an ftp:// task. ftps:// URLs always use
// implicit TLS and ftpes:// URLs always use explicit TLS.
const (
	TLSNone     = "none"
	TLSExplicit = "explicit"
	TLSImplicit = "implicit"
)

type OptsExtra struct {
	// Connections is the maximum number of parallel logins. Servers often cap
	// logins per user, so the fetcher lowers it when the server refuses one.
	Connections int `json:"connections"`
	// TLS upgrades a plain ftp:// URL: "none" (default), "explicit" (AUTH TLS)
	// or "implicit". It is ignored for ftps:// and ftpes:// URLs.
	TLS string `json:"tls"`
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

// Stats for download. It has the same shape as the HTTP protocol's stats, so
// clients can show FTP connections the way they show HTTP connections.
type Stats struct {
	Connections []*StatsConnection `json:"connections"`
}

// StatsConnection describes one login that took part in the download.
type StatsConnection struct {
	Downloaded int64 `json:"downloaded"`
	// Total is the resource size divided by the number of connections,
	// rounded up to a whole byte, as in the HTTP protocol's stats. It is zero
	// when the resource size is unknown.
	Total      int64 `json:"total"`
	Completed  bool  `json:"completed"`
	Failed     bool  `json:"failed"`
	RetryTimes int   `json:"retryTimes"`
}
