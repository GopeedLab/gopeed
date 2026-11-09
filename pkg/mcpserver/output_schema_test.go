package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/protocol/bt"
	"github.com/GopeedLab/gopeed/pkg/protocol/ed2k"
	phttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestToolSchemaPropertiesAreObjects(t *testing.T) {
	downloader := download.NewDownloader(&download.DownloaderConfig{Storage: download.NewMemStorage()})
	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	recorder := httptest.NewRecorder()
	NewHandler(downloader).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Result struct {
			Tools []struct {
				Name         string         `json:"name"`
				InputSchema  map[string]any `json:"inputSchema"`
				OutputSchema map[string]any `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Tools) == 0 {
		t.Fatal("tools/list returned no tools")
	}
	for _, tool := range response.Result.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			checkSchemaProperties(t, tool.InputSchema, "inputSchema")
			checkSchemaProperties(t, tool.OutputSchema, "outputSchema")
		})
	}
}

// Check the serialized definitions, including nested file requests and union
// branches. Boolean additionalProperties remains valid; boolean entries inside
// properties are what clients with object-only schema parsers cannot consume.
func checkSchemaProperties(t *testing.T, value any, path string) {
	t.Helper()
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "properties" {
				properties, ok := child.(map[string]any)
				if !ok {
					t.Fatalf("%s.properties = %#v, want object", path, child)
				}
				for name, property := range properties {
					if _, ok := property.(map[string]any); !ok {
						t.Errorf("%s.properties.%s = %#v, want schema object", path, name, property)
					}
				}
			}
			checkSchemaProperties(t, child, path+"."+key)
		}
	case []any:
		for i, child := range value {
			checkSchemaProperties(t, child, fmt.Sprintf("%s[%d]", path, i))
		}
	}
}

func TestToolOutputSchemasPreserveDynamicValues(t *testing.T) {
	statsSchema := toolOutputSchema[getTaskStatsOutput]()
	for _, test := range []struct {
		name  string
		stats any
	}{
		{"unavailable", nil},
		{"typed nil", (*fetcher.Stats)(nil)},
		{"http", &fetcher.Stats{Snapshot: &phttp.Stats{Connections: []*phttp.StatsConnection{{Downloaded: 123}}}}},
		{"bittorrent", &fetcher.Stats{Snapshot: &bt.StatsSnapshot{}, Runtime: &bt.StatsRuntime{}}},
		{"ed2k", &fetcher.Stats{Snapshot: &ed2k.StatsSnapshot{}, Runtime: &ed2k.StatsRuntime{}}},
		{"restored snapshot", &fetcher.Stats{Snapshot: json.RawMessage(`{"segmentsTotal":3,"segmentsDone":2,"segmentsFailed":0}`)}},
		{"empty object", map[string]any{}},
		{"nested JSON", map[string]any{"values": []any{nil, true, "text", 1.5, map[string]any{"key": "value"}}}},
		{"array", []any{1, "two", false}},
		{"string", "protocol value"},
		{"number", 1.5},
		{"boolean", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			validateToolOutput(t, statsSchema, &getTaskStatsOutput{Stats: test.stats})
		})
	}

	// Output extras can be empty, null, or extension-specific. In particular,
	// they need not satisfy the required settings in the download input schema.
	for _, extra := range []any{nil, map[string]any{}, &phttp.ReqExtra{}, &bt.ReqExtra{},
		map[string]any{"custom": []any{true, nil, 42}}, "custom value"} {
		req := &base.Request{URL: "https://example.com/file", Extra: extra}
		res := &base.Resource{Files: []*base.FileInfo{{Name: "file", Req: req}}}
		validateToolOutput(t, toolOutputSchema[resolveTaskOutput](), &resolveTaskOutput{Res: res})
		validateToolOutput(t, toolOutputSchema[getTaskOutput](), &getTaskOutput{Task: taskDetails{
			Req: req, Res: res, Opts: &base.Options{Extra: extra},
		}})
	}

	// Supplying an explicit schema must retain validation of the output envelope.
	resolved, err := statsSchema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []map[string]any{{}, {"stats": nil, "unexpected": true}} {
		if err := resolved.Validate(invalid); err == nil {
			t.Errorf("accepted invalid output envelope: %#v", invalid)
		}
	}
}

func validateToolOutput(t *testing.T, schema *jsonschema.Schema, output any) {
	t.Helper()
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolved.Validate(value); err != nil {
		t.Fatalf("output %s does not match schema: %v", data, err)
	}
}
