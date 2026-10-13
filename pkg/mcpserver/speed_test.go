package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetDownloadSpeedTool(t *testing.T) {
	// Serve a test payload that takes a little bit of time or streams bytes
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.WriteHeader(http.StatusOK)
		chunk := make([]byte, 8192)
		for i := 0; i < 128; i++ {
			_, _ = w.Write(chunk)
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer testServer.Close()

	downloader := download.NewDownloader(&download.DownloaderConfig{
		Storage:    download.NewMemStorage(),
		StorageDir: t.TempDir(),
	})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer downloader.Close()

	httpServer := httptest.NewServer(NewHandler(downloader))
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "gopeed-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	// 1. Zero tasks test (returns 0 totalSpeed and 0 activeTaskCount)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_download_speed",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("get_download_speed returned error: %+v", result.Content)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content type = %T", result.StructuredContent)
	}
	if activeCount, ok := structured["activeTaskCount"].(float64); !ok || activeCount != 0 {
		t.Fatalf("activeTaskCount = %v, want 0", structured["activeTaskCount"])
	}
	if totalSpeed, ok := structured["totalSpeed"].(float64); !ok || totalSpeed != 0 {
		t.Fatalf("totalSpeed = %v, want 0", structured["totalSpeed"])
	}

	// 2. Start a real download task
	taskID, err := downloader.CreateDirect(&base.Request{
		URL: testServer.URL,
	}, &base.Options{
		Path: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = downloader.Delete(&download.TaskFilter{IDs: []string{taskID}}, true)
	}()

	// Query get_download_speed while downloading
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_download_speed",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("get_download_speed returned error: %+v", result.Content)
	}
	structured, ok = result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content type = %T", result.StructuredContent)
	}
	if _, ok := structured["activeTaskCount"].(float64); !ok {
		t.Fatalf("activeTaskCount missing or wrong type: %v", structured["activeTaskCount"])
	}
	if _, ok := structured["totalSpeed"].(float64); !ok {
		t.Fatalf("totalSpeed missing or wrong type: %v", structured["totalSpeed"])
	}
}
