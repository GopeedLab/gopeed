package http

// DefaultUserAgent is shared by HTTP downloads and extension HTTP requests.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

type config struct {
	UserAgent      string `json:"userAgent"`
	Connections    int    `json:"connections"`
	UseServerCtime bool   `json:"useServerCtime"`
	// Adaptive is the default for tasks that do not set OptsExtra.Adaptive:
	// grow and shrink the connection count on measured throughput, with
	// Connections as the ceiling. Off unless the user turns it on.
	Adaptive bool `json:"adaptive"`
}
