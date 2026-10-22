package download

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestExtensionResolveSizeThroughTaskCreation(t *testing.T) {
	for _, mode := range []string{"http", "blob"} {
		for _, known := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/known=%v", mode, known), func(t *testing.T) {
				payload := []byte("0123456789")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body := payload
					if r.URL.Path == "/empty" {
						body = nil
					}
					if known {
						w.Header().Set("Content-Length", fmt.Sprint(len(body)))
					}
					w.(http.Flusher).Flush()
					w.Write(body)
				}))
				defer server.Close()
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir()})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Clear()
				d.cfg.MaxRunning = 0 // Inspect persisted queued tasks before HTTP resolution.
				extDir := t.TempDir()
				manifest := `{"name":"size-hint","author":"test","title":"Size hint","version":"1.0.0","scripts":[{"event":"onResolve","match":{"urls":["https://example.com/size-hint"]},"entry":"index.js"}]}`
				script := fmt.Sprintf(`gopeed.events.onResolve(async ctx=>{
  const mode=%q, origin=%q;
  const file=async(name,size,empty)=>({name,size,req:{url:mode==='http'?origin+(empty?'/empty':'/data'):
    await gopeed.runtime.blob.createObjectURL(()=>new ReadableStream({start(c){if(!empty)c.enqueue(new Uint8Array([48,49,50,51,52,53,54,55,56,57]));c.close();}}))}});
  ctx.res={files:[await file('under.bin',3,false),await file('over.bin',1048576,false),await file('empty.bin',99,true)]};
});`, mode, server.URL)
				for name, contents := range map[string]string{"manifest.json": manifest, "index.js": script} {
					if err := os.WriteFile(filepath.Join(extDir, name), []byte(contents), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := d.InstallExtensionByFolder(extDir, true); err != nil {
					t.Fatal(err)
				}
				rr, err := d.Resolve(&base.Request{URL: "https://example.com/size-hint"})
				if err != nil {
					t.Fatal(err)
				}
				output := t.TempDir()
				batch := &base.CreateTaskBatch{}
				for _, file := range rr.Res.Files {
					batch.Reqs = append(batch.Reqs, &base.CreateTaskBatchItem{Req: file.Req, Opts: &base.Options{Size: file.Size, Name: file.Name, Path: output}})
				}
				wire, err := json.Marshal(batch)
				if err != nil {
					t.Fatal(err)
				}
				var submitted base.CreateTaskBatch
				if err := json.Unmarshal(wire, &submitted); err != nil {
					t.Fatal(err)
				}
				ids, err := d.CreateDirectBatch(&submitted)
				if err != nil {
					t.Fatal(err)
				}
				for i, id := range ids {
					task := d.GetTask(id)
					if task.Meta.Res != nil {
						t.Fatal("a size estimate must not mark the task resolved")
					}
					status, err := d.RuntimeStatus(id)
					if err != nil || status.Total != rr.Res.Files[i].Size {
						t.Fatalf("queued size lost: status=%+v err=%v", status, err)
					}
					var saved Task
					if _, err := d.storage.Get(bucketTask, id, &saved); err != nil || saved.Meta.Opts.Size != status.Total {
						t.Fatal("queued estimate was not persisted", err)
					}
					if err := d.doStart(task); err != nil {
						t.Fatal(err)
					}
					waitForTaskTerminal(t, d, id, 5*time.Second)
					body, err := os.ReadFile(filepath.Join(output, rr.Res.Files[i].Name))
					if err != nil {
						t.Fatal(err)
					}
					want := payload
					if i == 2 {
						want = nil
					}
					if !bytes.Equal(body, want) {
						t.Fatalf("estimate changed output: %q", body)
					}
					status, err = d.RuntimeStatus(id)
					if err != nil || status.Total != int64(len(want)) || status.Downloaded != int64(len(want)) || task.Meta.Res.Files[0].Size != int64(len(want)) {
						t.Fatalf("final size not updated: %+v err=%v", status, err)
					}
				}
			})
		}
	}
}

func TestResolvedHTTPSizeEstimateAtCreation(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(fmt.Sprintf("known=%v", known), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if known {
					w.Header().Set("Content-Length", "10")
				}
				w.(http.Flusher).Flush()
				w.Write([]byte("0123456789"))
			}))
			defer server.Close()
			d := NewDownloader(&DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir()})
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			defer d.Clear()
			d.cfg.MaxRunning = 0
			rr, err := d.Resolve(&base.Request{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			id, err := d.CreateWithOptions(rr.ID, &base.Options{Size: 3, Path: t.TempDir(), Name: "output.bin"})
			if err != nil {
				t.Fatal(err)
			}
			want := int64(3)
			if known {
				want = 10
			}
			status, err := d.RuntimeStatus(id)
			if err != nil || status.Total != want || d.GetTask(id).Meta.Res.Files[0].Size != want {
				t.Fatalf("resolve-before-create size=%+v want=%d err=%v", status, want, err)
			}
			if err := d.doStart(d.GetTask(id)); err != nil {
				t.Fatal(err)
			}
			waitForTaskTerminal(t, d, id, 5*time.Second)
			task := d.GetTask(id)
			if task.Status != base.DownloadStatusDone || task.Meta.Res.Size != 10 {
				t.Fatalf("estimate changed completion: status=%s size=%d", task.Status, task.Meta.Res.Size)
			}
		})
	}
}
