package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
)

func TestFetcher_RestoredStartSkipsAbsentPrefetch(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789abcdef"), 64*1024)
	var firstRequest atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstRequest.CompareAndSwap(0, time.Now().UnixNano())
		http.ServeContent(w, r, "restored.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer server.Close()

	dir := t.TempDir()
	size := int64(len(data))
	half := size / 2
	file := filepath.Join(dir, "restored.bin")
	if err := os.WriteFile(file, data[:half], 0600); err != nil {
		t.Fatal(err)
	}
	conn := &connection{ID: 0, Role: rolePrimary, State: connNotStarted, Chunk: newChunk(0, size-1), Downloaded: half}
	conn.Chunk.Downloaded = half
	rangeMode := true
	saved := &fetcherData{Connections: []*connection{conn}, Range: &rangeMode, ResourceSize: &size, FileSize: &size}
	meta := &fetcher.FetcherMeta{
		Req:  &base.Request{URL: server.URL + "/restored.bin"},
		Res:  &base.Resource{Size: size, Range: true, Files: []*base.FileInfo{{Name: "restored.bin", Size: size}}},
		Opts: &base.Options{Path: dir, Name: "restored.bin", Extra: &fhttp.OptsExtra{Connections: 1}},
	}
	fm := new(FetcherManager)
	_, restore := fm.Restore()
	f := restore(meta, saved).(*Fetcher)
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		if err := json.Unmarshal([]byte(test.ToJson(fm.DefaultConfig())), v); err != nil {
			t.Error(err)
		}
	}
	f.Setup(ctl)
	defer f.Close()
	started := time.Now()
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, data) {
		t.Fatal("restored download differs from the complete server content")
	}
	if firstRequest.Load() == 0 {
		t.Fatal("restored download did not request the remaining content")
	}
	delay := time.Duration(firstRequest.Load() - started.UnixNano())
	t.Logf("first restored request after %s", delay)
	if delay > 2*time.Second {
		t.Errorf("restored download waited %s before its first request", delay)
	}
}
