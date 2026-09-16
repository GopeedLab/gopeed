package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/tempfiles"
)

type prefetchSetupBody struct {
	io.Reader
	closed bool
}

func (b *prefetchSetupBody) Close() error {
	b.closed = true
	return nil
}

func TestPrefetchSetupFailureClosesResponse(t *testing.T) {
	for _, closedScope := range []bool{false, true} {
		name := "invalid_temp_directory"
		if closedScope {
			name = "downloader_already_closed"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			scope := &tempfiles.Scope{}
			dir := root
			if closedScope {
				if err := scope.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				dir = filepath.Join(root, "file")
				if err := os.WriteFile(dir, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			body := &prefetchSetupBody{Reader: strings.NewReader("unused response")}
			f := &Fetcher{
				ctl:         &controller.Controller{TempDir: dir, TempFiles: scope},
				resolveResp: &http.Response{Body: body},
			}
			f.asyncPrefetch()
			if f.prefetchErr == nil {
				t.Fatal("expected temporary-file initialization to fail")
			}
			if !body.closed {
				t.Error("prefetch setup failure left the response body open")
			}
			if f.resolveResp != nil {
				t.Error("prefetch retained its failed resolve response")
			}
			if !f.prefetchDone.Load() {
				t.Error("prefetch did not report completion")
			}
			if closedScope {
				entries, err := os.ReadDir(root)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 0 {
					t.Errorf("late prefetch left temporary files: %v", entries)
				}
			}
		})
	}
}
