package http

type Config struct {
	UserAgent      string `json:"userAgent"`
	Connections    int    `json:"connections"`
	UseServerCtime bool   `json:"useServerCtime"`
	AutoTorrent    bool   `json:"autoTorrent"`
}
