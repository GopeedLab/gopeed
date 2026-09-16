package bt

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/anacrolix/torrent"
)

func TestStartUsesSelectionSubmittedAfterResolve(t *testing.T) {
	config := torrent.TestingConfig(t)
	config.DataDir = t.TempDir()
	config.DisableTCP = true
	config.DisableUTP = true
	c, err := torrent.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	tor, err := c.AddTorrentFromFile("testdata/test.torrent")
	if err != nil {
		t.Fatal(err)
	}
	f := &Fetcher{torrent: tor, meta: &fetcher.FetcherMeta{Opts: &base.Options{}}, data: &fetcherData{}}
	f.torrentReady.Store(true)
	f.updateRes()
	// Resolve initializes all files; creation replaces the selection before Start.
	f.meta.Opts = &base.Options{SelectFiles: []int{2}}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	for i, file := range tor.Files() {
		want := torrent.PiecePriorityNone
		if i == 2 {
			want = torrent.PiecePriorityNormal
		}
		if file.Priority() != want {
			t.Errorf("file %d priority = %v, want %v", i, file.Priority(), want)
		}
	}
	if len(f.data.Progress) != 1 {
		t.Fatalf("progress tracks %d files, want 1", len(f.data.Progress))
	}
	if err := f.Pause(); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if tor.Files()[0].Priority() != torrent.PiecePriorityNone {
		t.Fatal("resume selected an unselected file")
	}
}
