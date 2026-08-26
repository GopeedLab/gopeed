package ed2k

import (
	"testing"

	ped2k "github.com/GopeedLab/gopeed/pkg/protocol/ed2k"
	"github.com/monkeyWie/goed2k"
	"github.com/monkeyWie/goed2k/protocol"
)

func TestStatsSeparatesSnapshotAndRuntimeWithoutHandle(t *testing.T) {
	frame := (&Fetcher{}).Stats()
	if frame.Snapshot != nil {
		t.Fatalf("snapshot = %T, want nil without a valid handle", frame.Snapshot)
	}
	runtime, ok := frame.Runtime.(*ped2k.StatsRuntime)
	if !ok {
		t.Fatalf("runtime type = %T", frame.Runtime)
	}
	if runtime.Peers == nil || len(runtime.Peers) != 0 {
		t.Fatalf("unexpected runtime peers: %+v", runtime.Peers)
	}
	if runtime.ServerIDClass != "unknown" {
		t.Fatalf("server ID class = %q, want unknown", runtime.ServerIDClass)
	}
}

func TestED2KServerIDClassUsesConnectedPrimaryServer(t *testing.T) {
	tests := []struct {
		name    string
		servers []goed2k.ServerSnapshot
		want    string
	}{
		{name: "no server", want: "unknown"},
		{
			name: "high ID",
			servers: []goed2k.ServerSnapshot{
				{Connected: true, HandshakeCompleted: true, ClientID: 12345},
				{Primary: true, Connected: true, HandshakeCompleted: true, ClientID: 0x7f000001},
			},
			want: "high",
		},
		{
			name: "low ID",
			servers: []goed2k.ServerSnapshot{
				{Primary: true, Connected: true, HandshakeCompleted: true, ClientID: 12345},
			},
			want: "low",
		},
		{
			name: "handshake pending",
			servers: []goed2k.ServerSnapshot{
				{Primary: true, Connected: true, ClientID: 0x7f000001},
			},
			want: "unknown",
		},
		{
			name: "missing client ID",
			servers: []goed2k.ServerSnapshot{
				{Primary: true, Connected: true, HandshakeCompleted: true},
			},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ed2kServerIDClass(tt.servers); got != tt.want {
				t.Fatalf("ed2kServerIDClass() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildED2KPieceMapPreservesIndexes(t *testing.T) {
	snapshots := []goed2k.PieceSnapshot{
		{Index: 0, State: goed2k.PieceSnapshotFinished},
		{Index: 1, State: goed2k.PieceSnapshotMissing},
		{Index: 2, State: goed2k.PieceSnapshotDownloading},
		{Index: 3, State: goed2k.PieceSnapshotFinished},
	}
	pieceMap := buildED2KPieceMap(snapshots)
	if pieceMap == nil {
		t.Fatal("buildED2KPieceMap returned nil")
	}
	want := []bool{true, false, false, true}
	for index, expected := range want {
		got, err := pieceMap.Get(index)
		if err != nil || got != expected {
			t.Fatalf("piece %d = %v, %v; want %v", index, got, err, expected)
		}
	}
	if pieceMap.CompletedPieces() != 2 {
		t.Fatalf("CompletedPieces() = %d, want 2", pieceMap.CompletedPieces())
	}
}

func TestBuildED2KPeers(t *testing.T) {
	remotePieces := protocol.NewBitField(4)
	remotePieces.SetBit(0)
	remotePieces.SetBit(2)
	endpoint, err := protocol.EndpointFromString("127.0.0.1", 4662)
	if err != nil {
		t.Fatalf("EndpointFromString: %v", err)
	}
	peers := buildED2KPeers([]goed2k.PeerInfo{{
		PayloadDownloadSpeed: 2048,
		PayloadUploadSpeed:   1024,
		RemotePieces:         remotePieces,
		Endpoint:             endpoint,
		ModName:              "aMule",
		StrModVersion:        "2.3.3",
		SourceFlag:           int(goed2k.PeerServer | goed2k.PeerDHT),
	}})
	if len(peers) != 1 {
		t.Fatalf("len(peers) = %d, want 1", len(peers))
	}
	peer := peers[0]
	if peer.Client != "aMule 2.3.3" || peer.PieceCount != 2 || peer.Source != "server|kad" || peer.Transport != "tcp" {
		t.Fatalf("unexpected peer: %+v", peer)
	}
}
