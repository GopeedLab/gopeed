package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStreamableHTTPTools(t *testing.T) {
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Gopeed-Test"); got != "mcp-object" {
			t.Errorf("download request header = %q, want %q", got, "mcp-object")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer downloadServer.Close()

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

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(tools.Tools))
	var getTaskTool *mcp.Tool
	var resolveTaskTool *mcp.Tool
	var createTaskTool *mcp.Tool
	for _, tool := range tools.Tools {
		got = append(got, tool.Name)
		switch tool.Name {
		case "get_task":
			getTaskTool = tool
		case "resolve_task":
			resolveTaskTool = tool
		case "create_task":
			createTaskTool = tool
		}
	}
	want := []string{
		"continue_task",
		"create_task",
		"delete_task",
		"get_task",
		"get_task_stats",
		"get_task_status",
		"list_tasks",
		"pause_task",
		"resolve_task",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tool names = %v, want %v", got, want)
	}
	if getTaskTool == nil {
		t.Fatal("get_task tool not found")
	}
	if resolveTaskTool == nil || createTaskTool == nil {
		t.Fatal("download tools not found")
	}
	if instructions := session.InitializeResult().Instructions; instructions != serverInstructions {
		t.Fatalf("server instructions = %q, want %q", instructions, serverInstructions)
	}
	for _, tool := range []*mcp.Tool{resolveTaskTool, createTaskTool} {
		inputSchema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(inputSchema, []byte(`"extra":true`)) {
			t.Fatalf("%s input schema contains unconstrained boolean schema for extra: %s", tool.Name, inputSchema)
		}
		var schema map[string]any
		if err := json.Unmarshal(inputSchema, &schema); err != nil {
			t.Fatal(err)
		}
		reqExtra := nestedSchemaProperty(t, schema, "req", "extra")
		branches, ok := reqExtra["oneOf"].([]any)
		if reqExtra["type"] != "object" || !ok || len(branches) != 2 {
			t.Fatalf("%s req.extra schema = %#v, want an object with two oneOf branches", tool.Name, reqExtra)
		}
		optsExtra := nestedSchemaProperty(t, schema, "opts", "extra")
		if !schemaIncludesType(optsExtra, "object") || optsExtra["additionalProperties"] != false {
			t.Fatalf("%s opts.extra schema = %#v, want concrete HTTP task settings", tool.Name, optsExtra)
		}
	}
	outputSchema, err := json.Marshal(getTaskTool.OutputSchema)
	if err != nil {
		t.Fatal(err)
	}
	for _, property := range [][]byte{[]byte(`"task"`), []byte(`"id"`), []byte(`"req"`), []byte(`"res"`), []byte(`"opts"`)} {
		if !bytes.Contains(outputSchema, property) {
			t.Fatalf("get_task output schema %s does not contain property %s", outputSchema, property)
		}
	}

	resolveResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "resolve_task",
		Arguments: map[string]any{
			"req": map[string]any{
				"url": downloadServer.URL,
				"extra": map[string]any{
					"method": "GET",
					"header": map[string]any{"X-Gopeed-Test": "mcp-object"},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolveResult.IsError {
		t.Fatalf("resolve_task returned tool error: %+v", resolveResult.Content)
	}

	invalidExtraResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "resolve_task",
		Arguments: map[string]any{
			"req": map[string]any{
				"url": downloadServer.URL,
				"extra": map[string]any{
					"trackers": []any{"udp://tracker.example.com:80"},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !invalidExtraResult.IsError {
		t.Fatal("resolve_task accepted BitTorrent settings for an HTTP URL")
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_tasks",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("list_tasks returned tool error: %+v", result.Content)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured content type = %T", result.StructuredContent)
	}
	if tasks, ok := structured["tasks"].([]any); !ok || len(tasks) != 0 {
		t.Fatalf("tasks = %#v, want empty array", structured["tasks"])
	}
}

func nestedSchemaProperty(t *testing.T, schema map[string]any, path ...string) map[string]any {
	t.Helper()
	current := schema
	for _, name := range path {
		properties, ok := current["properties"].(map[string]any)
		if !ok {
			t.Fatalf("schema has no properties while resolving %v: %#v", path, current)
		}
		current, ok = properties[name].(map[string]any)
		if !ok {
			t.Fatalf("schema property %q is missing while resolving %v: %#v", name, path, properties)
		}
	}
	return current
}

func schemaIncludesType(schema map[string]any, want string) bool {
	switch value := schema["type"].(type) {
	case string:
		return value == want
	case []any:
		for _, item := range value {
			if item == want {
				return true
			}
		}
	}
	return false
}

func TestStreamableHTTPRejectsCrossOriginRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	req.Header.Set("Origin", "https://example.com")
	recorder := httptest.NewRecorder()

	downloader := download.NewDownloader(&download.DownloaderConfig{Storage: download.NewMemStorage()})
	NewHandler(downloader).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}
