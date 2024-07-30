package bt

import (
	"encoding/base64"
	"encoding/json"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
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

	doResolve(t, buildConfigFetcher(&base.DownloaderProxyConfig{
		Enable: true,
		System: false,
		Scheme: "socks5",
		Host:   proxyListener.Addr().String(),
		Usr:    usr,
		Pwd:    pwd,
	}))
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
	fb := new(FetcherManager)
	fetcher := fb.Build()
	newController := controller.NewController()
	newController.GetConfig = func(v any) {
		json.Unmarshal([]byte(test.ToJson(fb.DefaultConfig())), v)
	}
	fetcher.Setup(newController)
	return fetcher
}

func buildConfigFetcher(proxyConfig *base.DownloaderProxyConfig) fetcher.Fetcher {
	fetcher := new(FetcherManager).Build()
	newController := controller.NewController()
	mockCfg := config{
		Trackers: []string{
			"udp://tracker.birkenwald.de:6969/announce",
			"udp://tracker.bitsearch.to:1337/announce",
		}}
	newController.GetConfig = func(v any) {
		json.Unmarshal([]byte(test.ToJson(mockCfg)), v)
	}
	newController.ProxyConfig = proxyConfig
	fetcher.Setup(newController)
	return fetcher
}
