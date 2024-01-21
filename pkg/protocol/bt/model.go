package bt

type ReqExtra struct {
	Trackers []string `json:"trackers"`
}

// Stats for download
type Stats struct {
	// bt stats
	// health indicators of torrents, from large to small, ConnectedSeeders are also the key to the health of seed resources
	TotalPeers       int `json:"totalPeers"`
	ActivePeers      int `json:"activePeers"`
	ConnectedSeeders int `json:"connectedSeeders"`
}
