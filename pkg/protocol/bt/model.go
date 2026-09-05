package bt

import "github.com/GopeedLab/gopeed/pkg/base"

type ReqExtra struct {
	Trackers []string `json:"trackers"`
}

// StatsSnapshot contains cumulative BitTorrent data that remains meaningful
// after the torrent is stopped or the application restarts.
type StatsSnapshot struct {
	// Total seed bytes
	SeedBytes int64 `json:"seedBytes"`
	// Seed ratio
	SeedRatio float64 `json:"seedRatio"`
	// Total seed time
	SeedTime int64 `json:"seedTime"`
	// PieceMap contains verified local piece completion in compact bitset-v1
	// format. It is nil while metadata or storage completion is unavailable.
	PieceMap *base.PieceMap `json:"pieceMap"`
}

// StatsRuntime contains live swarm telemetry. It is never persisted because
// peer connections and live swarm counters cease to be valid once the fetcher
// stops.
type StatsRuntime struct {
	// Health indicators of torrents, from large to small. ConnectedSeeders are
	// also the key to the health of seed resources.
	TotalPeers        int `json:"totalPeers"`
	ActivePeers       int `json:"activePeers"`
	ConnectedSeeders  int `json:"connectedSeeders"`
	ConnectedLeechers int `json:"connectedLeechers"`
	// Peers contains currently connected BitTorrent peers.
	Peers []*base.PeerStats `json:"peers"`
}
