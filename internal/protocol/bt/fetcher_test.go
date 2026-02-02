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
	}, nil)
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
		}, nil)
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
		}, nil)
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
		}, nil)
		if err != nil {
			t.Errorf("Resolve Unclean Torrent Resolve() got = %v, want nil", err)
		}
	})

	t.Run("Resolve file scheme Torrent", func(t *testing.T) {
		file, _ := filepath.Abs("./testdata/test.unclean.torrent")
		uri := "file:///" + file
		err := fetcher.Resolve(&base.Request{
			URL: uri,
		}, nil)
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

// TestFetcher_Patch tests the Patch functionality for BT fetcher.
// It tests modifying selected files after Resolve (without downloading).
func TestFetcher_Patch(t *testing.T) {
	f := buildFetcher()

	// Resolve a multi-file torrent
	err := f.Resolve(&base.Request{
		URL: "./testdata/test.torrent",
	}, nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify initial state: 3 files, all selected by default
	meta := f.Meta()
	if len(meta.Res.Files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(meta.Res.Files))
	}
	if len(meta.Opts.SelectFiles) != 3 {
		t.Fatalf("Expected 3 selected files, got %d", len(meta.Opts.SelectFiles))
	}

	// Total size of all files: 107484864
	totalSize := int64(107484864)
	if meta.Res.Size != totalSize {
		t.Fatalf("Expected total size %d, got %d", totalSize, meta.Res.Size)
	}

	t.Run("Patch with valid indices", func(t *testing.T) {
		// Select only file 0 and 2 (c.txt and a.txt)
		err := f.Patch(nil, &base.Options{
			SelectFiles: []int{0, 2},
		})
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		meta := f.Meta()
		if !reflect.DeepEqual(meta.Opts.SelectFiles, []int{0, 2}) {
			t.Errorf("Expected SelectFiles [0, 2], got %v", meta.Opts.SelectFiles)
		}

		// Size should be recalculated: c.txt (98501754) + a.txt (78114) = 98579868
		expectedSize := int64(98501754 + 78114)
		if meta.Res.Size != expectedSize {
			t.Errorf("Expected size %d, got %d", expectedSize, meta.Res.Size)
		}
	})

	t.Run("Patch with invalid indices are silently ignored", func(t *testing.T) {
		// Mix of valid (0, 1) and invalid (-1, 5, 100) indices
		err := f.Patch(nil, &base.Options{
			SelectFiles: []int{-1, 0, 5, 1, 100},
		})
		if err != nil {
			t.Fatalf("Patch should not return error for invalid indices: %v", err)
		}

		meta := f.Meta()
		// Only valid indices 0 and 1 should remain
		if !reflect.DeepEqual(meta.Opts.SelectFiles, []int{0, 1}) {
			t.Errorf("Expected SelectFiles [0, 1], got %v", meta.Opts.SelectFiles)
		}

		// Size should be: c.txt (98501754) + b.txt (8904996) = 107406750
		expectedSize := int64(98501754 + 8904996)
		if meta.Res.Size != expectedSize {
			t.Errorf("Expected size %d, got %d", expectedSize, meta.Res.Size)
		}
	})

	t.Run("Patch with all invalid indices results in empty selection", func(t *testing.T) {
		err := f.Patch(nil, &base.Options{
			SelectFiles: []int{-5, 10, 999},
		})
		if err != nil {
			t.Fatalf("Patch should not return error: %v", err)
		}

		meta := f.Meta()
		if len(meta.Opts.SelectFiles) != 0 {
			t.Errorf("Expected empty SelectFiles, got %v", meta.Opts.SelectFiles)
		}

		// Note: CalcSize with empty selectFiles calculates total size of all files
		// This is by design - empty selection in CalcSize means "all files"
		// But SelectFiles being empty means no files are selected for download
		if meta.Res.Size != totalSize {
			t.Errorf("Expected size %d (CalcSize with empty slice = all files), got %d", totalSize, meta.Res.Size)
		}
	})

	t.Run("Patch with nil opts does nothing", func(t *testing.T) {
		// First set a known state
		f.Patch(nil, &base.Options{SelectFiles: []int{1}})
		prevSelectFiles := f.Meta().Opts.SelectFiles

		// Patch with nil opts
		err := f.Patch(nil, nil)
		if err != nil {
			t.Fatalf("Patch with nil opts should not fail: %v", err)
		}

		// Should remain unchanged
		if !reflect.DeepEqual(f.Meta().Opts.SelectFiles, prevSelectFiles) {
			t.Errorf("SelectFiles should remain unchanged after nil opts Patch")
		}
	})

	t.Run("Patch progress array is resized", func(t *testing.T) {
		btFetcher := f.(*Fetcher)
		// Initialize progress array
		btFetcher.data.Progress = make(fetcher.Progress, 3)

		err := f.Patch(nil, &base.Options{
			SelectFiles: []int{0, 2},
		})
		if err != nil {
			t.Fatalf("Patch failed: %v", err)
		}

		if len(btFetcher.data.Progress) != 2 {
			t.Errorf("Expected Progress length 2, got %d", len(btFetcher.data.Progress))
		}
	})
}
