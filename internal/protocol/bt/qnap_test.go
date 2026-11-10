package bt

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
)

// fakeQpkgConf writes a qpkg.conf and points the detection at it.
func fakeQpkgConf(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "qpkg.conf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	old := qpkgConfPath
	qpkgConfPath = path
	t.Cleanup(func() { qpkgConfPath = old })
}

func skipUnlessLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the QNAP detection only runs on Linux")
	}
}

const qpkgConfWithRtorrent16 = `[QsyncServer]
Name = QsyncServer
Enable = TRUE
Install_Path = /share/CACHEDEV1_DATA/.qpkg/QsyncServer

[rtorrent16]
Name = rtorrent16
Display_Name = Rtorrent 16
Enable = TRUE
Install_Path = /share/CACHEDEV1_DATA/.qpkg/rtorrent16

[Other]
Enable = FALSE
`

func TestDetectLocalRtorrentOnQNAP(t *testing.T) {
	skipUnlessLinux(t)
	fakeQpkgConf(t, qpkgConfWithRtorrent16)
	if got, want := detectLocalRtorrent(), "scgi:///share/CACHEDEV1_DATA/.qpkg/rtorrent16/var/run/rpc.sock"; got != want {
		t.Fatalf("detected %q, want %q", got, want)
	}
}

func TestDetectLocalRtorrentNeedsEnabledPackage(t *testing.T) {
	skipUnlessLinux(t)
	for name, content := range map[string]string{
		"disabled":         strings.Replace(qpkgConfWithRtorrent16, "Enable = TRUE\nInstall_Path = /share/CACHEDEV1_DATA/.qpkg/rtorrent16", "Enable = FALSE\nInstall_Path = /share/CACHEDEV1_DATA/.qpkg/rtorrent16", 1),
		"not installed":    "[QsyncServer]\nEnable = TRUE\nInstall_Path = /share/x\n",
		"relative path":    "[rtorrent16]\nEnable = TRUE\nInstall_Path = rtorrent16\n",
		"enable elsewhere": "[rtorrent16]\nInstall_Path = /share/x\n[Other]\nEnable = TRUE\n",
	} {
		t.Run(name, func(t *testing.T) {
			fakeQpkgConf(t, content)
			if got := detectLocalRtorrent(); got != "" {
				t.Fatalf("detected %q, want nothing", got)
			}
		})
	}
	qpkgConfPath = filepath.Join(t.TempDir(), "missing.conf")
	t.Cleanup(func() { qpkgConfPath = "/etc/config/qpkg.conf" })
	if got := detectLocalRtorrent(); got != "" {
		t.Fatalf("detected %q without a qpkg.conf", got)
	}
}

func TestDetectLocalRtorrentReadsQuotedValues(t *testing.T) {
	skipUnlessLinux(t)
	fakeQpkgConf(t, "[rtorrent16]\n  Enable = \"TRUE\"\r\nInstall_Path='/share/MD0_DATA/.qpkg/rtorrent16'\r\n")
	if got := detectLocalRtorrent(); got != "scgi:///share/MD0_DATA/.qpkg/rtorrent16/var/run/rpc.sock" {
		t.Fatalf("detected %q", got)
	}
}

func TestExternalTargetFollowsDetectionUntilTheUserDecides(t *testing.T) {
	skipUnlessLinux(t)
	fakeQpkgConf(t, qpkgConfWithRtorrent16)
	detected := "scgi:///share/CACHEDEV1_DATA/.qpkg/rtorrent16/var/run/rpc.sock"
	on, off := true, false
	for _, tt := range []struct {
		name     string
		cfg      externalConfig
		on       bool
		endpoint string
	}{
		{"automatic", externalConfig{}, true, detected},
		{"switched off", externalConfig{Enable: &off}, false, detected},
		{"switched on", externalConfig{Enable: &on}, true, detected},
		{"own address", externalConfig{URL: "http://other/RPC2"}, true, "http://other/RPC2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotOn, gotEndpoint := tt.cfg.target()
			if gotOn != tt.on || gotEndpoint != tt.endpoint {
				t.Fatalf("target() = %v %q, want %v %q", gotOn, gotEndpoint, tt.on, tt.endpoint)
			}
		})
	}
	fakeQpkgConf(t, "")
	if gotOn, _ := (&externalConfig{}).target(); gotOn {
		t.Fatal("automatic must be off without a detected rtorrent")
	}
}

// With nothing set, a torrent goes to Rtorrent 16's socket on a QNAP NAS.
func TestExternalAutomaticOnQNAP(t *testing.T) {
	skipUnlessLinux(t)
	install, err := os.MkdirTemp("", "q")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(install) })
	sock := filepath.Join(install, "var", "run", "rpc.sock")
	if len(sock) > 100 {
		t.Skip("the temp folder path is too long for a Unix socket")
	}
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		t.Fatal(err)
	}
	rt := test.NewFakeRtorrent(t)
	rt.ServeSCGIUnix(t, sock)
	fakeQpkgConf(t, "[rtorrent16]\nEnable = TRUE\nInstall_Path = "+install+"\n")

	tc := newExternalTestConfig("", "")
	c := *tc.cfg.Load()
	c.External = externalConfig{}
	tc.cfg.Store(&c)
	torrentPath, _, hash := test.WriteTorrent(t, "auto.bin", map[string][]byte{"": make([]byte, 3000)})
	f := buildExternalFetcher(t, tc)
	if err := f.Resolve(&base.Request{URL: torrentPath}); err != nil {
		t.Fatal(err)
	}
	f.Meta().Opts = &base.Options{Path: t.TempDir()}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if d := rt.Download(hash); d == nil || !d.Started {
		t.Fatal("the detected rtorrent did not get the torrent")
	}
	assertNoBuiltInClient(t)

	// Switched off by the user: the next torrent stays with the built-in engine.
	off := false
	c.External.Enable = &off
	tc.cfg.Store(&c)
	if on, _ := buildExternalFetcher(t, tc).externalTarget(); on {
		t.Fatal("the user's off must win over the detection")
	}
}

func TestLocalRtorrentJobUpdatesStateOnlyOnChange(t *testing.T) {
	skipUnlessLinux(t)
	fakeQpkgConf(t, qpkgConfWithRtorrent16)
	stored := new(FetcherManager).DefaultConfig().(*config)
	writes := 0
	host := fetcher.ProtocolHost{
		Read: func(v any) error {
			data, _ := json.Marshal(stored)
			return json.Unmarshal(data, v)
		},
		UpdateState: func(change func(config, state json.RawMessage) (any, error)) error {
			cfg, _ := json.Marshal(stored)
			st, _ := json.Marshal(stored.State)
			next, err := change(cfg, st)
			if err != nil {
				return err
			}
			data, _ := json.Marshal(next)
			writes++
			return json.Unmarshal(data, &stored.State)
		},
	}
	job := localRtorrentJob(host)
	if err := job.Run(context.Background(), fetcher.TriggerScheduled); err != nil {
		t.Fatal(err)
	}
	if stored.State.LocalRtorrent != "scgi:///share/CACHEDEV1_DATA/.qpkg/rtorrent16/var/run/rpc.sock" || writes != 1 {
		t.Fatalf("state %q after %d writes", stored.State.LocalRtorrent, writes)
	}
	if err := job.Run(context.Background(), fetcher.TriggerScheduled); !errors.Is(err, fetcher.ErrJobSkipped) || writes != 1 {
		t.Fatalf("an unchanged detection must skip without a write: %v, %d writes", err, writes)
	}
	fakeQpkgConf(t, strings.ReplaceAll(qpkgConfWithRtorrent16, "Enable = TRUE", "Enable = FALSE"))
	if err := job.Run(context.Background(), fetcher.TriggerScheduled); err != nil || stored.State.LocalRtorrent != "" {
		t.Fatalf("after disabling: %v, state %q", err, stored.State.LocalRtorrent)
	}
}
