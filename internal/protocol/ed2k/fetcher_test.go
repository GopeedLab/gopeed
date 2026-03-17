package ed2k

import (
	"testing"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/pkg/base"
)

const testLink = "ed2k://|file|cn_windows_10_multi-edition_vl_version_1709_updated_sept_2017_x64_dvd_100090774.iso|4630972416|8867C5E54405FF9452225B66EFEE690A|/"

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
	if got, want := meta.Res.Hash, "8867C5E54405FF9452225B66EFEE690A"; got != want {
		t.Fatalf("Resolve() hash = %s, want %s", got, want)
	}
	if got, want := meta.Res.Size, int64(4630972416); got != want {
		t.Fatalf("Resolve() size = %d, want %d", got, want)
	}
	if got, want := meta.Res.Files[0].Name, "cn_windows_10_multi-edition_vl_version_1709_updated_sept_2017_x64_dvd_100090774.iso"; got != want {
		t.Fatalf("Resolve() name = %q, want %q", got, want)
	}
}

func TestFetcherManager_ParseName(t *testing.T) {
	got := (&FetcherManager{}).ParseName(testLink)
	want := "cn_windows_10_multi-edition_vl_version_1709_updated_sept_2017_x64_dvd_100090774.iso"
	if got != want {
		t.Fatalf("ParseName() = %q, want %q", got, want)
	}
}
