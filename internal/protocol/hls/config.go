package hls

const (
	defaultSegmentConnections = 8
	defaultMaxRetries         = 5
	defaultTimeoutSeconds     = 30

	// maxPrefetchSizeSegments caps the HEAD probing done at resolve time.
	// Playlists larger than this skip size prefetch entirely.
	maxPrefetchSizeSegments = 500
)

type config struct {
	SegmentConnections    int  `json:"segmentConnections"`
	MaxRetries            int  `json:"maxRetries"`
	TimeoutSeconds        int  `json:"timeoutSeconds"`
	PrefetchContentLength bool `json:"prefetchContentLength"`
}

func (c *config) normalize() {
	if c.SegmentConnections <= 0 {
		c.SegmentConnections = defaultSegmentConnections
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = defaultMaxRetries
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = defaultTimeoutSeconds
	}
}
