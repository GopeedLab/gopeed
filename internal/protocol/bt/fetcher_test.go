package bt

import (
	"encoding/base64"
	"encoding/json"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/GopeedLab/gopeed/pkg/util"
	"net/url"
	"os"
	"reflect"
	"testing"
)

func TestFetcher_Resolve_Torrent(t *testing.T) {
	doResolve(t, buildFetcher())
}

func TestFetcher_Resolve_DataUri_Torrent(t *testing.T) {
	fetcher := buildFetcher()
	buf, err := os.ReadFile("./testdata/ubuntu-22.04-live-server-amd64.iso.torrent")
	if err != nil {
		t.Fatal(err)
	}
	// convert to data uri
	dataUri := "data:application/x-bittorrent;base64," + base64.StdEncoding.EncodeToString(buf)
	err = fetcher.Resolve(&base.Request{
		URL: dataUri,
	})
	if err != nil {
		panic(err)
	}

	want := &base.Resource{
		Name:  "ubuntu-22.04-live-server-amd64.iso",
		Size:  1466714112,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: "ubuntu-22.04-live-server-amd64.iso",
				Size: 1466714112,
			},
		},
		Hash: "8a55cfbd5ca5d11507364765936c4f9e55b253ed",
	}
	if !reflect.DeepEqual(want, fetcher.Meta().Res) {
		t.Errorf("Resolve() got = %v, want %v", fetcher.Meta().Res, want)
	}
}

func TestFetcher_Config(t *testing.T) {
	doResolve(t, buildConfigFetcher(nil))
}

func TestFetcher_ResolveWithProxy(t *testing.T) {
	usr, pwd := "admin", "123"
	proxyListener := test.StartSocks5Server(usr, pwd)
	defer proxyListener.Close()

	doResolve(t, buildConfigFetcher(util.BuildProxyUrl("socks5", proxyListener.Addr().String(), usr, pwd)))
}

func doResolve(t *testing.T, fetcher fetcher.Fetcher) {
	err := fetcher.Resolve(&base.Request{
		URL: "./testdata/ubuntu-22.04-live-server-amd64.iso.torrent",
		Extra: bt.ReqExtra{
			Trackers: []string{
				"udp://tracker.birkenwald.de:6969/announce",
				"udp://tracker.bitsearch.to:1337/announce",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	want := &base.Resource{
		Name:  "ubuntu-22.04-live-server-amd64.iso",
		Size:  1466714112,
		Range: true,
		Files: []*base.FileInfo{
			{
				Name: "ubuntu-22.04-live-server-amd64.iso",
				Size: 1466714112,
			},
		},
		Hash: "8a55cfbd5ca5d11507364765936c4f9e55b253ed",
	}
	if !reflect.DeepEqual(want, fetcher.Meta().Res) {
		t.Errorf("Resolve() got = %v, want %v", fetcher.Meta().Res, want)
	}
}

func buildFetcher() fetcher.Fetcher {
	fetcher := new(FetcherBuilder).Build()
	fetcher.Setup(controller.NewController())
	return fetcher
}

func buildConfigFetcher(proxyUrl *url.URL) fetcher.Fetcher {
	fetcher := new(FetcherBuilder).Build()
	newController := controller.NewController()
	mockCfg := config{
		Trackers: []string{
			"udp://tracker.birkenwald.de:6969/announce",
			"udp://tracker.bitsearch.to:1337/announce",
		}}
	newController.GetConfig = func(v any) bool {
		if err := json.Unmarshal([]byte(test.ToJson(mockCfg)), v); err != nil {
			return false
		}
		return true
	}
	newController.ToUrl = proxyUrl
	fetcher.Setup(newController)
	return fetcher
}
