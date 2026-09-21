package download

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	httpProtocol "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func TestExtensionExtraMutationHTTP(t *testing.T) {
	for _, resolved := range []bool{false, true} {
		for _, invalid := range []bool{false, true} {
			t.Run(fmt.Sprintf("resolved=%v/invalid=%v", resolved, invalid), func(t *testing.T) {
				var updatedRequests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-New") == "new" {
						updatedRequests.Add(1)
						if r.Header.Get("X-Old") != "" {
							t.Error("extra mutation retained the old header")
						}
					}
					http.ServeContent(w, r, "test.txt", time.Time{}, strings.NewReader("extra mutation payload"))
				}))
				defer server.Close()
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Clear()
				extDir := t.TempDir()
				manifest := `{"name":"http-methods","title":"HTTP methods test","version":"0.0.1","scripts":[{"event":"onStart","match":{"labels":["test"]},"entry":"index.js"}]}`
				mutation := `await ctx.task.meta.req.setMethod('GET');
                  await ctx.task.meta.req.setBody('');
                  await ctx.task.meta.req.delHeader('x-old');
                  await ctx.task.meta.req.putHeader('X-New', 'new');`
				if invalid {
					mutation = `try { await ctx.task.meta.req.setHeaders({'X-New':123}); }
                      catch (err) { await ctx.task.meta.req.putLabel('rejected','true'); } ` + mutation
				}
				script := `gopeed.events.onStart(async ctx => { ` + mutation + ` });`
				for name, content := range map[string]string{"manifest.json": manifest, "index.js": script} {
					if err := os.WriteFile(filepath.Join(extDir, name), []byte(content), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := d.InstallExtensionByFolder(extDir, false); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				d.Listener(func(event *Event) {
					if event.Key == EventKeyFinally {
						done <- event.Err
					}
				})
				req := &base.Request{URL: server.URL + "/test.txt", Labels: map[string]string{"test": "true"}, Extra: &httpProtocol.ReqExtra{Header: map[string]string{"X-Old": "old"}}}
				opts := &base.Options{Path: t.TempDir()}
				var id string
				var err error
				if resolved {
					rr, resolveErr := d.Resolve(req, opts)
					if resolveErr != nil {
						t.Fatal(resolveErr)
					}
					id, err = d.Create(rr.ID)
				} else {
					id, err = d.CreateDirect(req, opts)
				}
				if err != nil {
					t.Fatal(err)
				}
				select {
				case err = <-done:
				case <-time.After(10 * time.Second):
					t.Fatal("download timed out")
				}
				if invalid && d.GetTask(id).Meta.Req.Labels["rejected"] != "true" {
					t.Fatal("invalid header was not rejected")
				}
				if err != nil {
					t.Fatal(err)
				}
				got, ok := d.GetTask(id).Meta.Req.Extra.(*httpProtocol.ReqExtra)
				if !ok || got.Header["X-New"] != "new" || len(got.Header) != 1 {
					t.Fatalf("unexpected protocol extra: %#v", d.GetTask(id).Meta.Req.Extra)
				}
				if !resolved && updatedRequests.Load() == 0 {
					t.Fatal("download did not use the extension headers")
				}
				data, err := os.ReadFile(filepath.Join(opts.Path, "test.txt"))
				if err != nil || string(data) != "extra mutation payload" {
					t.Fatalf("download content = %q, error = %v", data, err)
				}
			})
		}
	}
}
