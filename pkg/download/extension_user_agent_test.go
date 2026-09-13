package download

import (
	"bytes"
	"fmt"
	httpProtocol "github.com/GopeedLab/gopeed/internal/protocol/http"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExtensionHTTPUserAgent(t *testing.T) {
	video, err := os.ReadFile("engine/testdata/ffmpeg/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	audio, err := os.ReadFile("engine/testdata/ffmpeg/audio.mp4")
	if err != nil {
		t.Fatal(err)
	}
	for _, configured := range []string{" Configured-UA/1.0 ", "", "<default>"} {
		for _, header := range []string{"default", "custom", "empty"} {
			t.Run(fmt.Sprintf("configured=%q/header=%s", configured, header), func(t *testing.T) {
				expected := strings.TrimSpace(configured)
				if configured == "<default>" {
					expected = httpProtocol.DefaultUserAgent
				}
				headers := `{}`
				if header == "custom" {
					expected = "Explicit-UA/2.0"
					headers = `{"uSeR-aGeNt":"Explicit-UA/2.0"}`
				}
				if header == "empty" {
					expected = ""
					headers = `{"user-agent":""}`
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.UserAgent() != expected {
						t.Errorf("%s UA=%q, want %q", r.URL.Path, r.UserAgent(), expected)
					}
					if strings.HasPrefix(r.URL.Path, "/redirect/") {
						http.Redirect(w, r, strings.TrimPrefix(r.URL.Path, "/redirect"), http.StatusFound)
						return
					}
					data := video
					if strings.HasSuffix(r.URL.Path, "audio") {
						data = audio
					}
					http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(data))
				}))
				defer server.Close()
				d, cleanup, err := newTestExtensionEngineDownloader()
				if err != nil {
					t.Fatal(err)
				}
				defer cleanup()
				cfg, err := d.GetConfig()
				if err != nil {
					t.Fatal(err)
				}
				cfg.ProtocolConfig["http"] = map[string]any{"userAgent": configured}
				if configured == "<default>" {
					cfg.ProtocolConfig["http"] = map[string]any{"connections": 4}
				}
				if err = d.PutConfig(cfg); err != nil {
					t.Fatal(err)
				}
				runtime, err := newTestExtensionEngine(t, d)
				if err != nil {
					t.Fatal(err)
				}
				defer runtime.Close()
				// The existing engine keeps its UA snapshot when the downloader config changes.
				cfg, err = d.GetConfig()
				if err != nil {
					t.Fatal(err)
				}
				cfg.ProtocolConfig["http"] = map[string]any{"userAgent": "Next-engine-UA"}
				if err = d.PutConfig(cfg); err != nil {
					t.Fatal(err)
				}
				result, err := runtime.Eval(fmt.Sprintf(`(async()=>{
      const headers=%s, url=%q;
      await (await fetch(url+'/redirect/fetch',{headers})).arrayBuffer();
      await new Promise((resolve,reject)=>{
        const xhr=new XMLHttpRequest();xhr.open('GET',url+'/redirect/xhr');
        for(const key of Object.keys(headers))xhr.setRequestHeader(key,headers[key]);
        xhr.onload=resolve;xhr.onerror=()=>reject(new Error('XHR failed'));xhr.send();
      });
      // Profile defaults must not replace the configured or explicitly empty UA.
      __gopeed_setFingerprint('chrome');
      await (await fetch(url+'/fingerprint',{headers})).arrayBuffer();
      const output=gopeed.runtime.ffmpeg.merge({video:{url:url+'/redirect/video',headers},audio:{url:url+'/redirect/audio',headers}});
      return (await new Response(output).arrayBuffer()).byteLength > 0;
    })()`, headers, server.URL))
				if err != nil {
					t.Fatal(err)
				}
				if result != true {
					t.Fatalf("merge result=%v", result)
				}
			})
		}
	}
}
