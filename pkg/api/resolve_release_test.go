package api

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	restmodel "github.com/GopeedLab/gopeed/pkg/rest/model"
)

func TestCancelledTorrentPreviewCanBeResolvedAgain(t *testing.T) {
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	path, err := filepath.Abs("../../internal/protocol/bt/testdata/test.torrent")
	if err != nil {
		t.Fatal(err)
	}
	req := &restmodel.ResolveTask{Req: &base.Request{URL: path}}
	first := serviceTestRequest[*download.ResolveResult](t, service, http.MethodPost, "/api/v1/resolve", req)
	serviceTestRequest[any](t, service, http.MethodDelete, "/api/v1/resolve/"+first.ID, nil)
	// Release is idempotent, including an old window disposing after Create.
	serviceTestRequest[any](t, service, http.MethodDelete, "/api/v1/resolve/"+first.ID, nil)
	second := serviceTestRequest[*download.ResolveResult](t, service, http.MethodPost, "/api/v1/resolve", req)
	if second.Res.Hash != first.Res.Hash {
		t.Fatalf("different torrent after release: %v", second)
	}
}
