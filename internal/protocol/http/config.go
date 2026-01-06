package http

type config struct {
	UserAgent          string `json:"userAgent"`
	Connections        int    `json:"connections"`
	UseServerCtime     bool   `json:"useServerCtime"`
	AutoExtract        bool   `json:"autoExtract"`        // AutoExtract enables automatic extraction of archives after download
	DeleteAfterExtract bool   `json:"deleteAfterExtract"` // DeleteAfterExtract deletes the archive after successful extraction
	AutoTorrent        bool   `json:"autoTorrent"`        // AutoTorrent enables automatic creation of torrent tasks for downloaded .torrent files
}
