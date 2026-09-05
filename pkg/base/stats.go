package base

// PeerStats is the common row returned by protocols that expose connected
// peers. Speeds contain payload bytes per second rather than protocol overhead.
type PeerStats struct {
	Address       string `json:"address"`
	Client        string `json:"client"`
	DownloadSpeed int64  `json:"downloadSpeed"`
	UploadSpeed   int64  `json:"uploadSpeed"`
	PieceCount    int    `json:"pieceCount"`
	// Completion is the known remote data completion ratio in the range 0–1.
	// It is omitted when a protocol cannot determine the remote total.
	Completion *float64 `json:"completion,omitempty"`
	// Relevance is the ratio of locally wanted, missing pieces that this peer
	// claims to have. It is omitted when the local missing set is unavailable or empty.
	Relevance *float64 `json:"relevance,omitempty"`
	Source    string   `json:"source"`
	// Transport is the active connection transport, for example tcp or utp.
	// It is intentionally separate from Source, which describes peer discovery.
	Transport string `json:"transport"`
}
