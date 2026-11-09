package http

// DefaultUserAgent is shared by HTTP downloads and extension HTTP requests.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

type config struct {
	UserAgent      string `json:"userAgent"`
	Connections    int    `json:"connections"`
	UseServerCtime bool   `json:"useServerCtime"`
	// MaxRetries is the number of retries each connection may make on its
	// own while no other connection of the download is healthy; any valid
	// response restores it for every unfinished connection. A negative value
	// uses the default (5); zero gives no such retries.
	MaxRetries int `json:"maxRetries"`
	// MaxRetryDelaySeconds caps the delay before a retry, which grows by one
	// second per retry. Values below 1 use the default (5).
	MaxRetryDelaySeconds int `json:"maxRetryDelaySeconds"`
	// UnlimitedRetries keeps retrying failed connections until the download
	// completes or is paused; MaxRetries is then ignored.
	UnlimitedRetries bool `json:"unlimitedRetries"`
}
