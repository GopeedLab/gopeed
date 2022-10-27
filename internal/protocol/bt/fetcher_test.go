package bt

import (
	"github.com/monkeyWie/gopeed/internal/controller"
	"github.com/monkeyWie/gopeed/internal/fetcher"
	"github.com/monkeyWie/gopeed/pkg/base"
	"reflect"
	"testing"
)

func TestFetcher_Resolve_Torrent(t *testing.T) {
	fetcher := buildFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "./testdata/ubuntu-22.04-live-server-amd64.iso.torrent",
	})
	if err != nil {
		panic(err)
	}

	want := &base.Resource{
		Req:   res.Req,
		Name:  "ubuntu-22.04-live-server-amd64.iso",
		Size:  1466714112,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: "ubuntu-22.04-live-server-amd64.iso",
				Size: 1466714112,
			},
		},
		Extra: map[string]any{
			"infoHash": "8a55cfbd5ca5d11507364765936c4f9e55b253ed",
		},
	}
	if !reflect.DeepEqual(want, res) {
		t.Errorf("Resolve() got = %v, want %v", res, want)
	}
}

func buildFetcher() fetcher.Fetcher {
	fetcher := new(FetcherBuilder).Build()
	fetcher.Setup(controller.NewController())
	return fetcher
}
