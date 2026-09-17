package rpcprovider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wv "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func TestPageEventStream(t *testing.T) {
	events := make(chan wv.Event, 8)
	disconnected := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request wv.RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "page.events" {
			t.Errorf("unexpected method %s", request.Method)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Error("missing authorization")
		}
		defer close(disconnected)
		enc := json.NewEncoder(w)
		_ = enc.Encode(map[string]any{"ready": true})
		w.(http.Flusher).Flush()
		for {
			select {
			case e := <-events:
				_ = enc.Encode(e)
				w.(http.Flusher).Flush()
				if e.Name == "closed" {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	}))
	defer server.Close()
	p := &page{client: NewClient(wv.RPCConfig{Network: "tcp", Address: strings.TrimPrefix(server.URL, "http://"), Token: "test"}), id: "page-1"}
	got := make(chan wv.Event, 8)
	off, err := p.On("load", func(e wv.Event) { got <- e })
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.On("closed", func(e wv.Event) { got <- e })
	if err != nil {
		t.Fatal(err)
	}
	events <- wv.Event{Name: "load", Data: map[string]any{"url": "https://example.test"}}
	select {
	case e := <-got:
		if e.Name != "load" {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("no load event")
	}
	off()
	off()
	events <- wv.Event{Name: "load"}
	events <- wv.Event{Name: "closed", Data: map[string]any{"reason": "user"}}
	select {
	case e := <-got:
		if e.Name != "closed" || e.Data["reason"] != "user" {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("no closed event")
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("stream retained after close")
	}
	if _, err = p.On("load", func(wv.Event) {}); err == nil {
		t.Fatal("registration after close")
	}
}
