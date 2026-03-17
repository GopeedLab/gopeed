package ed2k

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/pkg/base"
)

const testLink = "ed2k://|file|Taylor,%20Elizabeth%20-%20Prohibido%20morir%20aqui%20[66672]%20(r1.0).epub|434885|23A8CEFF57A7A32D562D649ED7893796|/"

func TestFetcher_Resolve(t *testing.T) {
	f := (&FetcherManager{}).Build()
	f.Setup(controller.NewController())

	err := f.Resolve(&base.Request{URL: testLink}, &base.Options{Path: t.TempDir()})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	meta := f.Meta()
	if meta.Res == nil {
		t.Fatal("Resolve() resource is nil")
	}
	if got, want := meta.Res.Hash, "23A8CEFF57A7A32D562D649ED7893796"; got != want {
		t.Fatalf("Resolve() hash = %s, want %s", got, want)
	}
	if got, want := meta.Res.Size, int64(434885); got != want {
		t.Fatalf("Resolve() size = %d, want %d", got, want)
	}
	if got, want := meta.Res.Files[0].Name, "Taylor, Elizabeth - Prohibido morir aqui [66672] (r1.0).epub"; got != want {
		t.Fatalf("Resolve() name = %q, want %q", got, want)
	}
}

func TestFetcherManager_ParseName(t *testing.T) {
	got := (&FetcherManager{}).ParseName(testLink)
	want := "Taylor, Elizabeth - Prohibido morir aqui [66672] (r1.0).epub"
	if got != want {
		t.Fatalf("ParseName() = %q, want %q", got, want)
	}
}
