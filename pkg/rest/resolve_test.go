package rest

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

func TestResolveMagnetAgainAfterRelease(t *testing.T) {
	seedConfig := torrent.TestingConfig(t)
	seedConfig.DisableUTP = true
	seedConfig.DisableIPv6 = true
	seedConfig.DisableWebtorrent = true
	seedConfig.Seed = true
	seedConfig.KeepAliveTimeout = time.Minute
	seedConfig.MaxAllocPeerRequestDataPerConn = 1 << 20
	const payload = "magnet preview test"
	path := filepath.Join(seedConfig.DataDir, "preview.txt")
	if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	info := metainfo.Info{PieceLength: 16 * 1024}
	if err := info.BuildFromFilePath(path); err != nil {
		t.Fatal(err)
	}
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	meta := &metainfo.MetaInfo{InfoBytes: infoBytes}
	seed, err := torrent.NewClient(seedConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Close()
	if _, err := seed.AddTorrent(meta); err != nil {
		t.Fatal(err)
	}
	addr := seed.ListenAddrs()[0].(*net.TCPAddr)
	var peer [6]byte
	copy(peer[:4], addr.IP.To4())
	binary.BigEndian.PutUint16(peer[4:], uint16(addr.Port))
	// A re-added torrent must find the peer before the old announce interval expires.
	trackerBody, err := bencode.Marshal(map[string]any{"interval": 3600, "peers": string(peer[:])})
	if err != nil {
		t.Fatal(err)
	}
	tracker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(trackerBody)
	}))
	defer tracker.Close()
	meta.Announce = tracker.URL
	magnet := meta.Magnet(nil, &info).String()
	body, err := json.Marshal(model.ResolveTask{Req: &base.Request{URL: magnet}})
	if err != nil {
		t.Fatal(err)
	}

	port, err := Start(&model.StartConfig{Storage: model.StorageMem, StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	config, err := Downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	config.ProtocolConfig["bt"] = map[string]any{"disableDht": true, "autoUpdateTrackers": false}
	if err := Downloader.PutConfig(config); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	request := func(method, path string, body []byte) json.RawMessage {
		t.Helper()
		req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d/api/v1%s", port, path), bytes.NewReader(body))
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		var result model.Result[json.RawMessage]
		if resp.StatusCode != http.StatusOK || json.Unmarshal(raw, &result) != nil || result.Code != model.CodeOk {
			t.Fatalf("%s %s returned HTTP %d: %s", method, path, resp.StatusCode, raw)
		}
		return result.Data
	}
	resolve := func() string {
		t.Helper()
		raw := request(http.MethodPost, "/resolve", body)
		var result download.ResolveResult
		if err := json.Unmarshal(raw, &result); err != nil || result.ID == "" || result.Res == nil || result.Res.Hash != meta.HashInfoBytes().HexString() {
			t.Fatalf("unexpected resolved resource: %s", raw)
		}
		return result.ID
	}
	for range 2 {
		id := resolve()
		request(http.MethodDelete, "/resolve/"+id, nil)
	}
	id := resolve()

	done := make(chan *download.Event, 1)
	Downloader.Listener(func(event *download.Event) {
		if event.Key == download.EventKeyDone || event.Key == download.EventKeyError {
			done <- event
		}
	})
	dir := t.TempDir()
	create, err := json.Marshal(model.CreateTask{Rid: id, Opts: &base.Options{Path: dir}})
	if err != nil {
		t.Fatal(err)
	}
	request(http.MethodPost, "/tasks", create)
	select {
	case event := <-done:
		if event.Key != download.EventKeyDone {
			t.Fatalf("download failed: %v", event.Err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("download did not finish after resolving the magnet again")
	}
	data, err := os.ReadFile(filepath.Join(dir, "preview.txt"))
	if err != nil || string(data) != payload {
		t.Fatalf("downloaded payload = %q, err = %v", data, err)
	}
}
