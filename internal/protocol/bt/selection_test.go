package bt

import (
	"context"
	"crypto/sha1"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
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

func TestWaitCleansUnselectedFilesFromSharedPiece(t *testing.T) {
	for _, mode := range []string{"partial", "all", "already removed", "remove error"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			keepPath := filepath.Join(dir, "bundle", "keep.txt")
			skipPath := filepath.Join(dir, "bundle", "nested", "skip.txt")
			if err := os.MkdirAll(filepath.Dir(skipPath), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(keepPath, []byte("keep"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(skipPath, []byte("skip"), 0644); err != nil {
				t.Fatal(err)
			}
			// Both files occupy the same piece, so downloading keep.txt also writes skip.txt.
			hash := sha1.Sum([]byte("keepskip"))
			info := metainfo.Info{Name: "bundle", PieceLength: 16, Pieces: hash[:], Files: []metainfo.FileInfo{
				{Length: 4, Path: []string{"keep.txt"}},
				{Length: 4, Path: []string{"nested", "skip.txt"}},
			}}
			store := storage.NewFileOpts(storage.NewFileClientOpts{
				ClientBaseDir:   dir,
				TorrentDirMaker: func(baseDir string, _ *metainfo.Info, _ metainfo.Hash) string { return baseDir },
			})
			defer store.Close()
			config := torrent.TestingConfig(t)
			config.DisableTCP = true
			config.DisableUTP = true
			c, err := torrent.NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			spec, err := torrent.TorrentSpecFromMetaInfoErr(&metainfo.MetaInfo{InfoBytes: bencode.MustMarshal(info)})
			if err != nil {
				t.Fatal(err)
			}
			spec.Storage = store
			tor, _, err := c.AddTorrentSpec(spec)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := tor.VerifyDataContext(ctx); err != nil {
				t.Fatal(err)
			}
			f := &Fetcher{torrent: tor, meta: &fetcher.FetcherMeta{Opts: &base.Options{Path: dir, SelectFiles: []int{0}}}, data: &fetcherData{}, torrentDropCtx: ctx}
			f.torrentReady.Store(true)
			f.updateRes()
			if mode == "all" {
				f.meta.Opts.SelectFiles = []int{0, 1}
			}
			if !f.isDone() {
				t.Fatal("selected file is not complete")
			}
			if mode == "already removed" || mode == "remove error" {
				if err := os.Remove(skipPath); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "remove error" {
				if err := os.Mkdir(skipPath, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(skipPath, "child"), []byte("preserve"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err = f.Wait()
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
			if mode == "remove error" {
				if err == nil {
					t.Fatal("cleanup failure was silently ignored")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(keepPath)
			if err != nil || string(data) != "keep" {
				t.Fatalf("selected file changed: %q, %v", data, err)
			}
			if mode == "all" {
				data, err := os.ReadFile(skipPath)
				if err != nil || string(data) != "skip" {
					t.Fatalf("selected file changed: %q, %v", data, err)
				}
			} else if mode != "remove error" {
				if _, err := os.Stat(skipPath); !os.IsNotExist(err) {
					t.Fatalf("unselected file remains: %v", err)
				}
			}
		})
	}
}
