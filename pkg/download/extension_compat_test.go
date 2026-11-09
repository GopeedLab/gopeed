package download

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestExtensionDeprecatedRequestAPIs(t *testing.T) {
	for _, event := range []ActivationEvent{EventOnResolve, EventOnStart, EventOnError, EventOnDone} {
		for _, withLabels := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/labels=%v", event, withLabels), func(t *testing.T) {
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Clear()

				extDir := t.TempDir()
				manifest := fmt.Sprintf(`{"name":"legacy-request","title":"Legacy request APIs","version":"0.0.1","scripts":[{"event":%q,"match":{"urls":["*://example.com/*"]},"entry":"index.js"}]}`, event)
				request := "ctx.task.meta.req"
				urlMutation := ""
				if event == EventOnResolve {
					request = "ctx.req"
				} else if event == EventOnStart || event == EventOnError {
					urlMutation = `await ctx.task.setUrl('https://example.com/legacy');
  if (req.url !== 'https://example.com/legacy') throw new Error('legacy URL did not write through');
  await req.setUrl('https://example.com/new');
  if (req.url !== 'https://example.com/new') throw new Error('new URL did not write through');
  await ctx.task.setUrl('https://example.com/final');`
				}
				script := fmt.Sprintf(`gopeed.events.%s(async ctx => {
  const req = %s;
  await req.delLabel('missing');
  await req.putLabel('initial', 'yes');
  if (req.labels.initial !== 'yes') throw new Error('putLabel did not initialize labels');
  await req.setLabels({keep: 'yes', remove: 'yes'});
  if (req.labels.initial !== undefined || req.labels.old !== undefined) throw new Error('setLabels did not replace labels');
  await req.putLabel('added', 'yes');
  await req.delLabel('remove');
  if (req.labels.keep !== 'yes' || req.labels.added !== 'yes' || req.labels.remove !== undefined) throw new Error('label mutation failed');
  %s
  await req.putLabel('completed', 'yes');
});`, event, request, urlMutation)
				for name, content := range map[string]string{"manifest.json": manifest, "index.js": script} {
					if err := os.WriteFile(filepath.Join(extDir, name), []byte(content), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := d.InstallExtensionByFolder(extDir, false); err != nil {
					t.Fatal(err)
				}
				req := &base.Request{URL: "https://example.com/original"}
				if withLabels {
					req.Labels = map[string]string{"old": "yes"}
				}
				task := &Task{Protocol: "http", Meta: &fetcher.FetcherMeta{Req: req, Opts: &base.Options{}}}
				switch event {
				case EventOnResolve:
					if _, err := d.triggerOnResolve(req); err != nil {
						t.Fatal(err)
					}
				case EventOnStart:
					d.triggerOnStart(task)
				case EventOnError:
					d.triggerOnError(task, errors.New("retry"))
				case EventOnDone:
					d.triggerOnDone(task)
				}
				if !reflect.DeepEqual(req.Labels, map[string]string{"keep": "yes", "added": "yes", "completed": "yes"}) {
					t.Fatalf("legacy handler did not update the original request: %#v", req.Labels)
				}
				expectedURL := "https://example.com/original"
				if urlMutation != "" {
					expectedURL = "https://example.com/final"
				}
				if req.URL != expectedURL {
					t.Fatalf("URL = %q, want %q", req.URL, expectedURL)
				}
			})
		}
	}
}
