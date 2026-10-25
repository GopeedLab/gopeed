package download

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestDownloader_Extension_MagnetMatch(t *testing.T) {
	const magnetURL = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"
	for _, pattern := range []string{"magnet:*", "magnet:?*"} {
		t.Run(pattern, func(t *testing.T) {
			d := NewDownloader(&DownloaderConfig{
				Storage:    NewMemStorage(),
				StorageDir: t.TempDir(),
				TempDir:    t.TempDir(),
			})
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			defer d.Clear()

			extDir := t.TempDir()
			manifest := fmt.Sprintf(`{"name":"magnet-match","author":"test","title":"Magnet match","version":"1.0.0","scripts":[{"event":"onResolve","match":{"urls":[%q]},"entry":"index.js"}]}`, pattern)
			script := `gopeed.events.onResolve(ctx => {
  ctx.res = {files: [{name: "resolved.txt", size: 1, req: {url: "https://example.com/resolved.txt"}}]};
});`
			for name, content := range map[string]string{"manifest.json": manifest, "index.js": script} {
				if err := os.WriteFile(filepath.Join(extDir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := d.InstallExtensionByFolder(extDir, true); err != nil {
				t.Fatal(err)
			}

			for _, req := range []*base.Request{
				{URL: magnetURL},
				{URL: "https://example.com/rewritten", RawURL: magnetURL},
			} {
				// Exercise extension activation without falling back to a live
				// BitTorrent metadata lookup when the pattern fails to match.
				res, err := d.triggerOnResolve(req)
				if err != nil {
					t.Fatal(err)
				}
				if res == nil || len(res.Files) != 1 || res.Files[0].Name != "resolved.txt" {
					t.Fatalf("magnet extension did not resolve request %+v: %+v", req, res)
				}
			}
		})
	}
}
