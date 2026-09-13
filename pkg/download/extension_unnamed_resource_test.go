package download

import (
	"github.com/GopeedLab/gopeed/pkg/base"
	"os"
	"path/filepath"
	"testing"
)

func TestExtensionUnnamedResource(t *testing.T) {
	d, cleanup, err := newTestExtensionEngineDownloader()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	dir := t.TempDir()
	manifest := `{"name":"unnamed","title":"Unnamed test","author":"test","version":"1.0.0","scripts":[{"event":"onResolve","match":{"urls":["https://example.com/unnamed"]},"entry":"index.js"}]}`
	script := `gopeed.events.onResolve(async(ctx)=>{ctx.res={range:false,files:[{name:'video.mp4',req:{url:await gopeed.runtime.blob.createObjectURL(()=>new ReadableStream({start(c){c.enqueue(new Uint8Array([1,2,3]));c.close();}}))}}]};});`
	for name, data := range map[string]string{"manifest.json": manifest, "index.js": script} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = d.InstallExtensionByFolder(dir, true); err != nil {
		t.Fatal(err)
	}
	res, err := d.Resolve(&base.Request{URL: "https://example.com/unnamed"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Res.Name != "" || len(res.Res.Files) != 1 || res.Res.Files[0].Name != "video.mp4" {
		t.Fatalf("unexpected resource: %+v", res.Res)
	}
	if !d.blob.IsURL(res.Res.Files[0].Req.URL) {
		t.Fatal("extension result was discarded")
	}
}
