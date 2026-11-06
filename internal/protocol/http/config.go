package http

// DefaultUserAgent is shared by HTTP downloads and extension HTTP requests.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

type config struct {
	UserAgent      string `json:"userAgent"`
	Connections    int    `json:"connections"`
	UseServerCtime bool   `json:"useServerCtime"`
	// MaxRetries is the number of retry rounds a download gets once all of
	// its unfinished connections have failed. A negative value uses the
	// default (5); zero fails the download as soon as that happens.
	MaxRetries int `json:"maxRetries"`
	// MaxRetryDelaySeconds caps the delay before a retry, which grows by one
	// second per retry. Values below 1 use the default (5).
	MaxRetryDelaySeconds int `json:"maxRetryDelaySeconds"`
	// UnlimitedRetries keeps retrying failed connections until the download
	// completes or is paused; MaxRetries is then ignored.
	UnlimitedRetries bool `json:"unlimitedRetries"`
}
