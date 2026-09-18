package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTempDirInitialization(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{false: "system-default", true: "explicit"}[configured], func(t *testing.T) {
			root := t.TempDir()
			// os.TempDir uses different environment variables across supported platforms.
			t.Setenv("TMPDIR", root)
			t.Setenv("TMP", root)
			t.Setenv("TEMP", root)
			cfg := &DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()}
			if configured {
				cfg.TempDir = root
			}
			existing := filepath.Join(root, "gopeed-media-existing")
			if err := os.Mkdir(existing, 0700); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(existing, "keep")
			if err := os.WriteFile(marker, []byte("other owner"), 0600); err != nil {
				t.Fatal(err)
			}
			d := NewDownloader(cfg)
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			if d.cfg.TempDir != root {
				t.Fatalf("TempDir=%q, want %q", d.cfg.TempDir, root)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatal("initialization removed existing data", err)
			}
			owned, err := os.MkdirTemp(d.cfg.TempDir, "gopeed-media-")
			if err != nil {
				t.Fatal(err)
			}
			if err := d.tempFiles.Track(owned); err != nil {
				t.Fatal(err)
			}
			if err := d.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(owned); !os.IsNotExist(err) {
				t.Fatal("shutdown retained owned data", err)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatal("shutdown removed another owner's data", err)
			}
		})
	}
}
