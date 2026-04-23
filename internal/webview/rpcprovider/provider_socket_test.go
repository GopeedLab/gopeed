package rpcprovider

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func TestProviderIsAvailableOverUnixSocket(t *testing.T) {
	socketPath := filepath.Join("/tmp", "gopeed-rpcwebview-"+t.Name()+".sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req enginewebview.RPCRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Method != enginewebview.MethodIsAvailable {
				t.Fatalf("unexpected unix method: %s", req.Method)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{
				"result": enginewebview.IsAvailableResult{Available: true},
				"error":  nil,
			}); err != nil {
				t.Fatal(err)
			}
		}),
	}
	defer server.Close()
	go server.Serve(listener)

	provider := New(enginewebview.RPCConfig{
		Network: "unix",
		Address: socketPath,
	})
	if !provider.IsAvailable() {
		t.Fatal("expected unix provider to be available")
	}
}
