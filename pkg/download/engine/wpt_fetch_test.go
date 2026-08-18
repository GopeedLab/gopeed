package engine_test

import (
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
)

//go:embed testdata/wpt_harness.js
var wptHarness string

//go:embed testdata/wpt/fetch/api/resources/utils.js
var wptFetchUtils string

//go:embed testdata/wpt/fetch/api/request/request-error.js
var wptRequestError string

//go:embed testdata/wpt/fetch/api/cors/resources/not-cors-safelisted.json
var wptNoCORSSafelistedHeaders []byte

//go:embed testdata/wpt/fetch/api/headers/*.any.js
var wptHeaders embed.FS

//go:embed testdata/wpt/fetch/api/request/*.any.js testdata/wpt/fetch/api/response/*.any.js
var wptRequestResponse embed.FS

func TestWPTHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(wptNoCORSSafelistedHeaders)
	}))
	t.Cleanup(server.Close)

	tests := []string{
		"header-setcookie.any.js",
		"headers-basic.any.js",
		"headers-casing.any.js",
		"headers-combine.any.js",
		"headers-errors.any.js",
		"headers-forbidden-override.any.js",
		"headers-normalize.any.js",
		"headers-no-cors.any.js",
		"headers-record.any.js",
		"headers-structure.any.js",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			setup := ""
			if name == "headers-no-cors.any.js" {
				setup = fmt.Sprintf("globalThis.location = new URL(%q);", server.URL+"/fetch/api/headers/test.any.js")
			}
			runWPTFile(t, wptHeaders, path.Join("testdata/wpt/fetch/api/headers", name), setup)
		})
	}
}

func TestWPTRequestResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte("{}"))
	}))
	t.Cleanup(server.Close)
	setup := fmt.Sprintf("globalThis.location = new URL(%q);", server.URL+"/fetch/api/response/test.any.js")

	tests := []string{
		"request/forbidden-method.any.js",
		"request/request-constructor-init-body-override.any.js",
		"request/request-consume.any.js",
		"request/request-consume-empty.any.js",
		"request/request-disturbed.any.js",
		"request/request-error.any.js",
		"request/request-headers.any.js",
		"request/request-init-002.any.js",
		"request/request-init-contenttype.any.js",
		"request/request-init-priority.any.js",
		"request/request-init-stream.any.js",
		"request/request-keepalive.any.js",
		"request/request-clone-readable-stream-body.any.js",
		"request/request-structure.any.js",
		"response/response-consume-empty.any.js",
		"response/response-consume-stream.any.js",
		"response/response-init-001.any.js",
		"response/response-init-002.any.js",
		"response/response-init-contenttype.any.js",
		"response/response-from-stream.any.js",
		"response/response-headers-guard.any.js",
		"response/response-error.any.js",
		"response/response-stream-bad-chunk.any.js",
		"response/response-static-error.any.js",
		"response/response-static-json.any.js",
		"response/response-static-redirect.any.js",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			runWPTFile(t, wptRequestResponse, path.Join("testdata/wpt/fetch/api", name), setup)
		})
	}
}

func runWPTFile(t *testing.T, files embed.FS, name, setup string) {
	t.Helper()
	source, err := files.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	runtime := engine.NewEngine(nil)
	t.Cleanup(runtime.Close)
	value, err := runtime.RunString(setup + "\n" + wptHarness + "\n" + wptFetchUtils + "\n" + wptRequestError + "\n" + string(source) + "\n__wptFinish();")
	if err != nil {
		t.Fatal(err)
	}
	if value == nil {
		t.Fatal("WPT harness returned no result")
	}
}
