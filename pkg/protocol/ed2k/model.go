package ed2k

type Stats struct {
	State         string `json:"state"`
	Paused        bool   `json:"paused"`
	ActivePeers   int    `json:"activePeers"`
	TotalPeers    int    `json:"totalPeers"`
	DownloadRate  int    `json:"downloadRate"`
	Upload        int64  `json:"upload"`
	UploadRate    int    `json:"uploadRate"`
	TotalDone     int64  `json:"totalDone"`
	TotalReceived int64  `json:"totalReceived"`
	TotalWanted   int64  `json:"totalWanted"`
}
