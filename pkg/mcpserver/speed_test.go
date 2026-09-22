package mcpserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetDownloadSpeedTool(t *testing.T) {
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

	// 1. Query all tasks speed (empty list)
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

	// 2. Query non-existent task ID -> should return error
	notFoundResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_download_speed",
		Arguments: map[string]any{
			"id": "non-existent-task-id",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !notFoundResult.IsError {
		t.Fatal("get_download_speed should return error for non-existent task ID")
	}
}
