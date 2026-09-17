package rpcprovider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestPageEventStreamHandshakeFailureCanRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"http_error", http.StatusUnauthorized, `{"ready":true}`},
		{"invalid_json", http.StatusOK, `not json`},
		{"not_ready", http.StatusOK, `{"ready":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					w.WriteHeader(tc.status)
					_, _ = io.WriteString(w, tc.body)
					return
				}
				_, _ = io.WriteString(w, "{\"ready\":true}\n{\"event\":\"load\",\"data\":{\"url\":\"https://example.test\"}}\n")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer server.Close()
			p := &page{client: NewClient(wv.RPCConfig{Network: "tcp", Address: strings.TrimPrefix(server.URL, "http://")}), id: "retry-page"}
			if off, err := p.On("load", func(wv.Event) { t.Error("failed registration retained its listener") }); err == nil || off != nil {
				t.Fatal("failed handshake accepted")
			}
			got := make(chan wv.Event, 1)
			off, err := p.On("load", func(e wv.Event) { got <- e })
			if err != nil {
				t.Fatalf("retry: %v", err)
			}
			defer func() {
				p.eventCancel()
				<-p.eventDone
			}()
			defer off()
			select {
			case e := <-got:
				if e.Data["url"] != "https://example.test" {
					t.Fatal(e)
				}
			case <-time.After(time.Second):
				t.Fatal("retry did not receive load notification")
			}
			if requests.Load() != 2 {
				t.Fatalf("unexpected request count: %d", requests.Load())
			}
		})
	}
}

func TestPageEventStreamDisconnectAndClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request wv.RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		switch request.Method {
		case wv.MethodPageEvents:
			// End the stream without a terminal event, as if the host dropped it.
			_, _ = io.WriteString(w, "{\"ready\":true}\n")
		case wv.MethodPageClose:
			_ = json.NewEncoder(w).Encode(wv.RPCResponse{})
		default:
			t.Errorf("unexpected method: %s", request.Method)
		}
	}))
	defer server.Close()
	p := &page{client: NewClient(wv.RPCConfig{Network: "tcp", Address: strings.TrimPrefix(server.URL, "http://")}), id: "disconnected-page"}
	closed := make(chan wv.Event, 1)
	if _, err := p.On("closed", func(e wv.Event) { closed <- e }); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.eventDone:
	case <-time.After(time.Second):
		t.Fatal("disconnected stream did not stop")
	}
	if _, err := p.On("load", func(wv.Event) {}); err == nil || !strings.Contains(err.Error(), "disconnected") {
		t.Fatalf("registration hid stream failure: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-closed:
		if event.Data["reason"] != "api" {
			t.Fatal(event)
		}
	default:
		t.Fatal("API close did not release listeners after disconnect")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-closed:
		t.Fatalf("duplicate close notification: %+v", event)
	default:
	}
}

func TestPageEventStreamConnectionFailureCanRetry(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	p := &page{client: NewClient(wv.RPCConfig{Network: "tcp", Address: strings.TrimPrefix(server.URL, "http://")}), id: "offline-page"}
	for i := 0; i < 2; i++ {
		if off, err := p.On("load", func(wv.Event) {}); err == nil || off != nil {
			t.Fatal("registration to unavailable host succeeded")
		}
	}
	if p.eventCancel != nil {
		t.Fatal("failed connection retained an active stream")
	}
}
