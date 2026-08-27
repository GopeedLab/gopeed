package ed2k

import "github.com/GopeedLab/gopeed/pkg/base"

type StatsSnapshot struct {
	Upload        int64          `json:"upload"`
	TotalDone     int64          `json:"totalDone"`
	TotalReceived int64          `json:"totalReceived"`
	TotalWanted   int64          `json:"totalWanted"`
	PieceMap      *base.PieceMap `json:"pieceMap"`
}

type StatsRuntime struct {
	State        string            `json:"state"`
	Paused       bool              `json:"paused"`
	ActivePeers  int               `json:"activePeers"`
	TotalPeers   int               `json:"totalPeers"`
	DownloadRate int               `json:"downloadRate"`
	UploadRate   int               `json:"uploadRate"`
	Peers        []*base.PeerStats `json:"peers"`
}
