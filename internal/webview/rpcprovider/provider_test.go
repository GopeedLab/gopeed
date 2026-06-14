package rpcprovider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

func TestProviderLifecycleOverHTTP(t *testing.T) {
	var calls []enginewebview.RPCRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != enginewebview.RPCEndpointPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var req enginewebview.RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		calls = append(calls, req)

		var body any
		switch req.Method {
		case enginewebview.MethodIsAvailable:
			body = map[string]any{
				"result": enginewebview.IsAvailableResult{Available: true},
				"error":  nil,
			}
		case enginewebview.MethodPageOpen:
			body = map[string]any{
				"result": enginewebview.PageOpenResult{PageID: "page-open-1"},
				"error":  nil,
			}
		case enginewebview.MethodPageAddInitScript, enginewebview.MethodPageGoto:
			body = map[string]any{
				"result": enginewebview.EmptyResult{},
				"error":  nil,
			}
		case enginewebview.MethodPageExecute:
			body = map[string]any{
				"result": map[string]any{"title": "Example"},
				"error":  nil,
			}
		case enginewebview.MethodPageClose:
			body = map[string]any{
				"result": enginewebview.EmptyResult{},
				"error":  nil,
			}
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	cfg := enginewebview.RPCConfig{
		Network: "tcp",
		Address: server.Listener.Addr().String(),
		Token:   "secret-token",
	}
	provider := New(cfg)
	if !provider.IsAvailable() {
		t.Fatal("expected provider to be available")
	}

	opener, ok := provider.(enginewebview.Opener)
	if !ok {
		t.Fatal("expected provider to implement runtime webview opener")
	}
	opened, err := opener.Open(enginewebview.OpenOptions{
		Headless:  true,
		Width:     1280,
		Height:    720,
		Title:     "Gopeed",
		UserAgent: "UA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.AddInitScript("window.__TEST__ = true"); err != nil {
		t.Fatal(err)
	}
	if err := opened.Goto("https://example.com", enginewebview.GotoOptions{
		TimeoutMS: 3210,
		WaitUntil: "domcontentloaded",
	}); err != nil {
		t.Fatal(err)
	}
	value, err := opened.Execute("document.title")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := value.(map[string]any)
	if !ok || got["title"] != "Example" {
		t.Fatalf("unexpected evaluate result: %#v", value)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}

	expectedMethods := []string{
		enginewebview.MethodIsAvailable,
		enginewebview.MethodIsAvailable,
		enginewebview.MethodPageOpen,
		enginewebview.MethodPageAddInitScript,
		enginewebview.MethodPageGoto,
		enginewebview.MethodPageExecute,
		enginewebview.MethodPageClose,
	}
	if len(calls) != len(expectedMethods) {
		t.Fatalf("unexpected call count: got %d want %d", len(calls), len(expectedMethods))
	}
	for i, method := range expectedMethods {
		if calls[i].Method != method {
			t.Fatalf("unexpected method at %d: got %s want %s", i, calls[i].Method, method)
		}
	}
	gotoParamsJSON, err := json.Marshal(calls[4].Params)
	if err != nil {
		t.Fatal(err)
	}
	var gotoParams enginewebview.PageGotoParams
	if err := json.Unmarshal(gotoParamsJSON, &gotoParams); err != nil {
		t.Fatal(err)
	}
	if gotoParams.WaitUntil != "domcontentloaded" {
		t.Fatalf("unexpected waitUntil: %q", gotoParams.WaitUntil)
	}
}

func TestProviderLaunchUnavailableWithoutEndpoint(t *testing.T) {
	provider := New(enginewebview.RPCConfig{})
	if provider.IsAvailable() {
		t.Fatal("expected empty provider to be unavailable")
	}
	opener, ok := provider.(enginewebview.Opener)
	if !ok {
		t.Fatal("expected provider to implement runtime webview opener")
	}
	if _, err := opener.Open(enginewebview.OpenOptions{}); err == nil || err.Error() != enginewebview.ErrUnavailable.Error() {
		t.Fatalf("unexpected launch error: %v", err)
	}
}

func TestPageSetCookieParamsOmitsZeroExpires(t *testing.T) {
	params := enginewebview.PageSetCookieParams{
		PageID: "page-1",
		Cookie: enginewebview.Cookie{
			Name:   "visible",
			Value:  "1",
			Domain: "example.com",
			Path:   "/",
		},
	}
	payload, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) == "" {
		t.Fatal("expected non-empty json payload")
	}
	if got := string(payload); strings.Contains(got, `"expires"`) {
		t.Fatalf("unexpected zero expires in payload: %s", got)
	}

	params.Cookie.Expires = time.Now().UTC().Truncate(time.Second)
	payload, err = json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(payload); !strings.Contains(got, `"expires"`) {
		t.Fatalf("expected expires in payload: %s", got)
	}
}
