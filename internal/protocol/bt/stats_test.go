package bt

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	protocolbt "github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/RoaringBitmap/roaring"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

func TestStatsSeparatesSnapshotAndRuntime(t *testing.T) {
	frame := (&Fetcher{
		meta: &fetcher.FetcherMeta{Res: &base.Resource{Size: 256}},
		data: &fetcherData{SeedBytes: 128, SeedTime: 64},
	}).Stats()
	if frame.Snapshot != nil {
		t.Fatalf("snapshot = %T, want nil until piece completion is known", frame.Snapshot)
	}
	runtime, ok := frame.Runtime.(*protocolbt.StatsRuntime)
	if !ok {
		t.Fatalf("runtime type = %T", frame.Runtime)
	}
	if len(runtime.Peers) != 0 || runtime.TotalPeers != 0 {
		t.Fatalf("unexpected runtime: %+v", runtime)
	}
}

func TestApplyBTPieceRunsPublishesVerifiedCompletionOnly(t *testing.T) {
	pieceMap := base.NewPieceMap(4, 1024)
	ready := applyBTPieceRuns(pieceMap, []torrent.PieceStateRun{
		{PieceState: torrent.PieceState{Completion: torrentCompletion(true, true)}, Length: 1},
		{PieceState: torrent.PieceState{Completion: torrentCompletion(true, false), Partial: true}, Length: 2},
		{PieceState: torrent.PieceState{Completion: torrentCompletion(false, false), Hashing: true}, Length: 1},
	})
	if ready {
		t.Fatal("piece map became ready while one completion is unknown")
	}
	want := []bool{true, false, false, false}
	for index, expected := range want {
		got, err := pieceMap.Get(index)
		if err != nil || got != expected {
			t.Fatalf("piece %d = %v, %v; want %v", index, got, err, expected)
		}
	}

	knownMap := base.NewPieceMap(2, 1024)
	if !applyBTPieceRuns(knownMap, []torrent.PieceStateRun{
		{PieceState: torrent.PieceState{Completion: torrentCompletion(true, true)}, Length: 1},
		{PieceState: torrent.PieceState{Completion: torrentCompletion(true, false)}, Length: 1},
	}) {
		t.Fatal("fully known completion map was not ready")
	}
	if applyBTPieceRuns(base.NewPieceMap(2, 1024), []torrent.PieceStateRun{
		{PieceState: torrent.PieceState{Completion: torrentCompletion(true, true)}, Length: 1},
	}) {
		t.Fatal("incomplete piece runs were published")
	}
}

func torrentCompletion(ok, complete bool) storage.Completion {
	return storage.Completion{Ok: ok, Complete: complete}
}

func TestNormalizeBTPeerMetadata(t *testing.T) {
	if got := normalizeBTPeerSource(torrent.PeerSourceDhtGetPeers); got != "dht" {
		t.Fatalf("DHT source = %q", got)
	}
	if got := normalizeBTPeerSource(torrent.PeerSourcePex); got != "pex" {
		t.Fatalf("PEX source = %q", got)
	}
	transports := map[string]string{
		"udp4":   "utp",
		"udp6":   "utp",
		"tcp4":   "tcp",
		"tcp6":   "tcp",
		"webrtc": "webrtc",
		"unix":   "unknown",
	}
	for network, want := range transports {
		if got := normalizeBTTransport(network); got != want {
			t.Errorf("normalizeBTTransport(%q) = %q, want %q", network, got, want)
		}
	}
}

func TestBTPeerCompletion(t *testing.T) {
	if got := btPeerCompletion(1, 4); got == nil || *got != 0.25 {
		t.Fatalf("completion = %v, want 0.25", got)
	}
	if got := btPeerCompletion(5, 4); got == nil || *got != 1 {
		t.Fatalf("clamped completion = %v, want 1", got)
	}
	if got := btPeerCompletion(-1, 4); got == nil || *got != 0 {
		t.Fatalf("negative completion = %v, want 0", got)
	}
	if got := btPeerCompletion(0, 0); got != nil {
		t.Fatalf("completion without metadata = %v, want nil", *got)
	}
}

func TestBTPeerRelevanceUsesWantedMissingPieces(t *testing.T) {
	missing := btMissingWantedPieces([]torrent.PieceStateRun{
		{
			PieceState: torrent.PieceState{
				Completion: torrentCompletion(true, true),
				Priority:   torrent.PiecePriorityNormal,
			},
			Length: 2,
		},
		{
			PieceState: torrent.PieceState{
				Completion: torrentCompletion(true, false),
				Priority:   torrent.PiecePriorityNormal,
			},
			Length: 2,
		},
		{
			PieceState: torrent.PieceState{
				Completion: torrentCompletion(true, false),
				Priority:   torrent.PiecePriorityNone,
			},
			Length: 2,
		},
	}, 6)
	if missing == nil || !missing.Equals(roaring.BitmapOf(2, 3)) {
		t.Fatalf("missing wanted pieces = %v, want {2, 3}", missing)
	}

	if got := btPeerRelevance(roaring.BitmapOf(0, 2, 4), missing); got == nil || *got != 0.5 {
		t.Fatalf("relevance = %v, want 0.5", got)
	}
	if got := btPeerRelevance(roaring.BitmapOf(0, 1, 4), missing); got == nil || *got != 0 {
		t.Fatalf("zero relevance = %v, want 0", got)
	}
	if got := btPeerRelevance(roaring.BitmapOf(2, 3), roaring.New()); got != nil {
		t.Fatalf("relevance with no missing pieces = %v, want nil", *got)
	}
	if got := btMissingWantedPieces(nil, 6); got != nil {
		t.Fatalf("incomplete piece states = %v, want nil", got)
	}
}

func TestBTPeerRelevanceTracksRemainingDownload(t *testing.T) {
	missing := roaring.BitmapOf(90, 91, 92, 93, 94, 95, 96, 97, 98, 99)
	if got := btPeerRelevance(missing.Clone(), missing); got == nil || *got != 1 {
		t.Fatalf("peer covering every missing piece = %v, want 1", got)
	}

	oneRelevantPiece := roaring.BitmapOf(90)
	if got := btPeerRelevance(oneRelevantPiece, missing); got == nil || *got != 0.1 {
		t.Fatalf("peer covering one of ten missing pieces = %v, want 0.1", got)
	}

	missingAfterTransfer := missing.Clone()
	missingAfterTransfer.Remove(90)
	if got := btPeerRelevance(oneRelevantPiece, missingAfterTransfer); got == nil || *got != 0 {
		t.Fatalf("peer after transferring its only relevant piece = %v, want 0", got)
	}
}
