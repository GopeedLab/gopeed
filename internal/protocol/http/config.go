package http

// DefaultUserAgent is shared by HTTP downloads and extension HTTP requests.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

type config struct {
	UserAgent      string `json:"userAgent"`
	Connections    int    `json:"connections"`
	UseServerCtime bool   `json:"useServerCtime"`
	// MaxRetries is the number of retries each connection may make on its
	// own while no other connection of the download is healthy; any valid
	// response restores it for every unfinished connection. The default is 5;
	// -1 is unlimited and zero gives no such retries.
	MaxRetries int `json:"maxRetries"`
}
