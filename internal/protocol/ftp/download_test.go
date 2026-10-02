package ftp_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/GopeedLab/gopeed/internal/logger"
	iftp "github.com/GopeedLab/gopeed/internal/protocol/ftp"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	pftp "github.com/GopeedLab/gopeed/pkg/protocol/ftp"
)

// syncBuffer is a log sink that the downloader's goroutines can share.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// taskEvents records the first done or error event of every task, so the
// test never reads a task's status while the downloader writes it.
type taskEvents struct {
	mu    sync.Mutex
	final map[string]download.EventKey
}

func (e *taskEvents) listen(ev *download.Event) {
	if ev.Key != download.EventKeyDone && ev.Key != download.EventKeyError {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, seen := e.final[ev.Task.ID]; !seen {
		e.final[ev.Task.ID] = ev.Key
	}
}

func (e *taskEvents) wait(t *testing.T, id string) download.EventKey {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.Lock()
		key, ok := e.final[id]
		e.mu.Unlock()
		if ok {
			return key
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task %s did not finish", id)
	return ""
}

// TestPasswordNeverLogged runs FTP tasks through the downloader with its core
// logger captured: a success, a missing file and a wrong password, all behind
// an HTTP proxy that FTP must skip with a warning.
func TestPasswordNeverLogged(t *testing.T) {
	srv := iftp.StartFTPServer(t, iftp.ServerOptions{User: "u", Pass: "p@ss"})
	want := srv.WriteFile(t, "secret.bin", iftp.RandomBytes(300000, 21))

	d := download.NewDownloader(&download.DownloaderConfig{Storage: download.NewMemStorage(), StorageDir: t.TempDir()})
	logs := &syncBuffer{}
	d.Logger = &logger.Logger{Logger: zerolog.New(logs)}
	events := &taskEvents{final: map[string]download.EventKey{}}
	d.Listener(events.listen)
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	// Close, not Clear: Clear resets the extension list without a lock while
	// the last task's done hooks may still read it.
	defer d.Close()
	cfg, err := d.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Proxy = &base.DownloaderProxyConfig{Enable: true, Scheme: "http", Host: "127.0.0.1:1", Usr: "proxyuser", Pwd: "proxysecret"}
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	create := func(rawURL string) string {
		id, err := d.CreateDirect(&base.Request{URL: rawURL}, &base.Options{Path: dir, Extra: &pftp.OptsExtra{Connections: 2}})
		if err != nil {
			if strings.Contains(err.Error(), "p@ss") || strings.Contains(err.Error(), "p%40ss") {
				t.Fatalf("the password leaked into an error: %v", err)
			}
			t.Fatal(err)
		}
		return id
	}
	root := "ftp://u:p%40ss@" + srv.Addr + "/"

	okID := create(root + "secret.bin")
	if key := events.wait(t, okID); key != download.EventKeyDone {
		t.Fatalf("download ended with %q, logs:\n%s", key, logs.String())
	}
	ok := d.GetTask(okID)
	if ok.Protocol != "ftp" {
		t.Fatalf("protocol %q, want ftp", ok.Protocol)
	}
	if got := iftp.FileSHA256(t, ok.Meta.SingleFilepath()); got != want {
		t.Fatalf("sha256 %s, want %s", got, want)
	}
	missingID := create(root + "missing.bin")
	if key := events.wait(t, missingID); key != download.EventKeyError {
		t.Fatalf("a missing file must fail, got %q", key)
	}
	wrongID := create("ftp://u:wr0ng%40pw@" + srv.Addr + "/secret.bin")
	if key := events.wait(t, wrongID); key != download.EventKeyError {
		t.Fatalf("a wrong password must fail, got %q", key)
	}
	missing, wrong := d.GetTask(missingID), d.GetTask(wrongID)
	for _, task := range []*download.Task{ok, missing, wrong} {
		if name := task.Name(); strings.Contains(name, "p@ss") || strings.Contains(name, "p%40ss") || strings.Contains(name, "wr0ng") {
			t.Fatalf("task name %q shows the password", name)
		}
	}

	out := logs.String()
	if !strings.Contains(out, "task id") || !strings.Contains(out, "HTTP proxy") {
		t.Fatalf("the captured log misses the expected lines, so it proves nothing:\n%s", out)
	}
	for _, secret := range []string{"p@ss", "p%40ss", "wr0ng", "proxysecret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("%q appears in the log:\n%s", secret, out)
		}
	}
}
