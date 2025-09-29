package bt

import (
	"encoding/base64"
	"encoding/json"
	gohttp "net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
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
	t.Run("Resolve Single File", func(t *testing.T) {
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
			t.Errorf("Resolve Single File Resolve() got = %v, want %v", fetcher.Meta().Res, want)
		}
	})

	t.Run("Resolve Multi Files", func(t *testing.T) {
		err := fetcher.Resolve(&base.Request{
			URL: "./testdata/test.torrent",
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
			Name:  "test",
			Size:  107484864,
			Range: true,
			Files: []*base.FileInfo{
				{
					Name: "c.txt",
					Path: "path",
					Size: 98501754,
				},
				{
					Name: "b.txt",
					Size: 8904996,
				},
				{
					Name: "a.txt",
					Size: 78114,
				},
			},
			Hash: "ccbc92b0cd8deec16a2ef4be242a8c9243b1cedb",
		}
		if !reflect.DeepEqual(want, fetcher.Meta().Res) {
			t.Errorf("Resolve Multi Files Resolve() got = %v, want %v", fetcher.Meta().Res, want)
		}
	})

	t.Run("Resolve Unclean Torrent", func(t *testing.T) {
		err := fetcher.Resolve(&base.Request{
			URL: "./testdata/test.unclean.torrent",
		})
		if err != nil {
			t.Errorf("Resolve Unclean Torrent Resolve() got = %v, want nil", err)
		}
	})

	t.Run("Resolve file scheme Torrent", func(t *testing.T) {
		file, _ := filepath.Abs("./testdata/test.unclean.torrent")
		uri := "file:///" + file
		err := fetcher.Resolve(&base.Request{
			URL: uri,
		})
		if err != nil {
			t.Errorf("Resolve file scheme Torrent Resolve() got = %v, want nil", err)
		}
	})
}

func TestFetcherManager_ParseName(t *testing.T) {
	type args struct {
		u string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "broken url",
			args: args{
				u: "magnet://!@#%github.com",
			},
			want: "",
		},
		{
			name: "dn",
			args: args{
				u: "magnet:?xt=urn:btih:8a55cfbd5ca5d11507364765936c4f9e55b253ed&dn=ubuntu-22.04-live-server-amd64.iso",
			},
			want: "ubuntu-22.04-live-server-amd64.iso",
		},
		{
			name: "no dn",
			args: args{
				u: "magnet:?xt=urn:btih:8a55cfbd5ca5d11507364765936c4f9e55b253ed",
			},
			want: "8a55cfbd5ca5d11507364765936c4f9e55b253ed",
		},
		{
			name: "non standard magnet",
			args: args{
				u: "magnet:?xxt=abcd",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := &FetcherManager{}
			if got := fm.ParseName(tt.args.u); got != tt.want {
				t.Errorf("ParseName() = %v, want %v", got, tt.want)
			}
		})
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
	newController.GetProxy = func(requestProxy *base.RequestProxy) func(*gohttp.Request) (*url.URL, error) {
		return proxyConfig.ToHandler()
	}
	fetcher.Setup(newController)
	return fetcher
}
