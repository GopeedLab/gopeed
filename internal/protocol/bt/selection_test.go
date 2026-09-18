package bt

import (
	"context"
	"crypto/sha1"
	"fmt"
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
	for _, mode := range []string{"partial", "part only", "both paths", "all", "already removed", "remove error"} {
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
			storageOpts := storage.NewFileClientOpts{
				ClientBaseDir:   dir,
				TorrentDirMaker: func(baseDir string, _ *metainfo.Info, _ metainfo.Hash) string { return baseDir },
			}
			// VerifyDataContext can return before storage finishes marking the piece
			// complete. Disable automatic part-file promotion so it cannot rename
			// the .part fixtures that this test creates to exercise cleanup.
			storageOpts.UsePartFiles.Set(false)
			store := storage.NewFileOpts(storageOpts)
			defer store.Close()
			config := torrent.TestingConfig(t)
			config.DisableTCP = true
			config.DisableUTP = true
			c, err := torrent.NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			spec, err := torrent.TorrentSpecFromMetaInfoErr(&metainfo.MetaInfo{InfoBytes: bencode.MustMarshal(&info)})
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
			if mode == "part only" || mode == "both paths" || mode == "all" {
				if err := os.WriteFile(skipPath+".part", []byte("skip"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "part only" || mode == "already removed" || mode == "remove error" {
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
				if _, err := os.Stat(skipPath + ".part"); err != nil {
					t.Fatalf("selected part file was removed: %v", err)
				}
				data, err := os.ReadFile(skipPath)
				if err != nil || string(data) != "skip" {
					t.Fatalf("selected file changed: %q, %v", data, err)
				}
			} else if mode != "remove error" {
				for _, removed := range []string{skipPath, skipPath + ".part", filepath.Dir(skipPath)} {
					if _, err := os.Stat(removed); !os.IsNotExist(err) {
						t.Fatalf("unselected file or empty directory remains: %s: %v", removed, err)
					}
				}
			}
		})
	}
}

func TestRemoveUnselectedFilePrunesOnlyEmptyParents(t *testing.T) {
	for _, preserveSibling := range []bool{false, true} {
		t.Run(fmt.Sprint(preserveSibling), func(t *testing.T) {
			root := t.TempDir()
			relative := filepath.Join("bundle", "nested", "deep", "skip.txt")
			name := filepath.Join(root, relative)
			if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(name+".part", []byte("partial"), 0644); err != nil {
				t.Fatal(err)
			}
			sibling := filepath.Join(root, "bundle", "keep.txt")
			if preserveSibling {
				if err := os.WriteFile(sibling, []byte("keep"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := removeUnselectedFile(root, "bundle", relative); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, "bundle", "nested")); !os.IsNotExist(err) {
				t.Fatalf("empty ancestors remain: %v", err)
			}
			if preserveSibling {
				data, err := os.ReadFile(sibling)
				if err != nil || string(data) != "keep" {
					t.Fatalf("sibling changed: %q, %v", data, err)
				}
			}
			if info, err := os.Stat(filepath.Join(root, "bundle")); err != nil || !info.IsDir() {
				t.Fatalf("torrent root directory was removed: %v", err)
			}
			if _, err := os.Stat(root); err != nil {
				t.Fatalf("download directory was removed: %v", err)
			}
			// A file directly inside the torrent root must not cause that root to be pruned.
			rootFile := filepath.Join("bundle", "root.txt")
			if err := os.WriteFile(filepath.Join(root, rootFile)+".part", []byte("partial"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := removeUnselectedFile(root, "bundle", rootFile); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, "bundle")); err != nil {
				t.Fatalf("torrent root directory was removed after root-level cleanup: %v", err)
			}
			// Repeating cleanup after the file and its subdirectories are gone is harmless.
			if err := removeUnselectedFile(root, "bundle", relative); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCleanupMatchesNamedAndRootlessStorage(t *testing.T) {
	for _, version := range []int64{1, 2} {
		for _, layout := range []string{"rootless", "wrapped", "file-tree-root"} {
			if version == 1 && layout == "file-tree-root" {
				continue
			}
			t.Run(fmt.Sprintf("v%d/%s", version, layout), func(t *testing.T) {
				dir := t.TempDir()
				info := metainfo.Info{Name: metainfo.NoName, PieceLength: 16384}
				root := dir
				if layout == "wrapped" {
					info.Name = "bundle"
					root = filepath.Join(dir, "bundle")
				}
				if version == 1 {
					info.Files = []metainfo.FileInfo{{Path: []string{"keep.txt"}}, {Path: []string{"nested", "deep", "skip.txt"}}}
				} else {
					info.MetaVersion = 2
					info.FileTree = metainfo.FileTree{Dir: map[string]metainfo.FileTree{
						"keep.txt": {},
						"nested":   {Dir: map[string]metainfo.FileTree{"deep": {Dir: map[string]metainfo.FileTree{"skip.txt": {}}}}},
					}}
				}
				if layout == "file-tree-root" {
					info.FileTree = metainfo.FileTree{Dir: map[string]metainfo.FileTree{"tree-root": info.FileTree}}
					root = filepath.Join(dir, "tree-root")
				}
				// Keep the library's default FilePathMaker here to independently verify
				// that cleanup agrees with its real disk layout, including NoName.
				store := storage.NewFileOpts(storage.NewFileClientOpts{
					ClientBaseDir:   dir,
					TorrentDirMaker: func(baseDir string, _ *metainfo.Info, _ metainfo.Hash) string { return baseDir },
				})
				defer store.Close()
				torrentStorage, err := store.OpenTorrent(context.Background(), &info, metainfo.Hash{1})
				if err != nil {
					t.Fatal(err)
				}
				defer torrentStorage.Close()
				skip := filepath.Join(root, "nested", "deep", "skip.txt")
				if _, err := os.Stat(skip); err != nil {
					t.Fatalf("unexpected storage layout: %v", err)
				}
				if err := os.WriteFile(skip+".part", []byte("partial"), 0644); err != nil {
					t.Fatal(err)
				}
				files := info.UpvertedFiles()
				boundary, name := torrentFileLayout(&info, files[1])
				if err := removeUnselectedFile(dir, boundary, name); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(root, "nested")); !os.IsNotExist(err) {
					t.Fatalf("empty subtree remains: %v", err)
				}
				if _, err := os.Stat(filepath.Join(root, "keep.txt")); err != nil {
					t.Fatalf("selected file removed: %v", err)
				}
				// Also verify the boundary when the protected directory becomes empty.
				if err := os.Remove(filepath.Join(root, "keep.txt")); err != nil {
					t.Fatal(err)
				}
				for _, file := range files {
					boundary, name := torrentFileLayout(&info, file)
					if err := removeUnselectedFile(dir, boundary, name); err != nil {
						t.Fatal(err)
					}
				}
				if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
					t.Fatalf("cleanup boundary was removed: %v", err)
				}
			})
		}
	}
}
