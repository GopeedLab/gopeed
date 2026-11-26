package rest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func newTestLoginPage() (*webLoginPageHandler, *webSessionStore) {
	sessions := newWebSessionStore()
	auth := &model.WebAuth{Username: "gopeed", Password: "secret"}
	return newWebLoginPageHandler(auth, sessions), sessions
}

func TestWebLoginPageRendersForm(t *testing.T) {
	page, _ := newTestLoginPage()
	req := httptest.NewRequest(http.MethodGet, webLoginPagePath, nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rec := httptest.NewRecorder()

	page.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`name="username"`,
		`autocomplete="username"`,
		`type="password"`,
		`autocomplete="current-password"`,
		`action="/login"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("login page missing %q", want)
		}
	}
}

func TestWebLoginPageSubmit(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		next       string
		wantStatus int
		wantLoc    string
		wantCookie bool
	}{
		{"success", "gopeed", "secret", "", http.StatusSeeOther, "/", true},
		{"success with next", "gopeed", "secret", "/tasks", http.StatusSeeOther, "/tasks", true},
		{"open redirect blocked", "gopeed", "secret", "//evil.example.com", http.StatusSeeOther, "/", true},
		{"wrong password", "gopeed", "nope", "", http.StatusUnauthorized, "", false},
		{"wrong username", "admin", "secret", "", http.StatusUnauthorized, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, sessions := newTestLoginPage()
			form := url.Values{"username": {tt.username}, "password": {tt.password}, "next": {tt.next}}
			req := httptest.NewRequest(http.MethodPost, webLoginPagePath, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			page.handlePost(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantLoc != "" {
				if loc := rec.Header().Get("Location"); loc != tt.wantLoc {
					t.Errorf("Location = %q, want %q", loc, tt.wantLoc)
				}
			}
			cookies := rec.Result().Cookies()
			found := false
			for _, cookie := range cookies {
				if cookie.Name == webAuthCookieName {
					found = true
					if !tt.wantCookie {
						t.Fatalf("unexpected session cookie")
					}
					if !sessions.valid(cookie.Value) {
						t.Errorf("session %q should be valid", cookie.Value)
					}
					if !cookie.HttpOnly {
						t.Errorf("session cookie should be HttpOnly")
					}
				}
			}
			if tt.wantCookie && !found {
				t.Fatalf("missing %s cookie", webAuthCookieName)
			}
			if !tt.wantCookie && rec.Code == http.StatusUnauthorized {
				if !strings.Contains(rec.Body.String(), "Incorrect username or password") {
					t.Errorf("missing credentials error message")
				}
			}
		})
	}
}

func TestWebLoginPageRedirectsWhenAuthenticated(t *testing.T) {
	page, sessions := newTestLoginPage()
	sessionID, _, err := sessions.create()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, webLoginPagePath+"?next=/extensions", nil)
	req.AddCookie(&http.Cookie{Name: webAuthCookieName, Value: sessionID})
	rec := httptest.NewRecorder()

	page.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/extensions" {
		t.Errorf("Location = %q, want /extensions", loc)
	}
}

func TestWebLoginLogout(t *testing.T) {
	page, sessions := newTestLoginPage()
	sessionID, _, err := sessions.create()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, webLogoutPath, nil)
	req.AddCookie(&http.Cookie{Name: webAuthCookieName, Value: sessionID})
	rec := httptest.NewRecorder()

	page.handleLogout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != webLoginPagePath {
		t.Errorf("Location = %q, want %q", loc, webLoginPagePath)
	}
	var cleared bool
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == webAuthCookieName && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Errorf("logout should clear %s cookie", webAuthCookieName)
	}
}

func TestIsHTMLNavigation(t *testing.T) {
	tests := []struct {
		name   string
		method string
		accept string
		want   bool
	}{
		{"browser navigation", http.MethodGet, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", true},
		{"head navigation", http.MethodHead, "text/html", true},
		{"api call", http.MethodGet, "application/json", false},
		{"asset", http.MethodGet, "*/*", false},
		{"form post", http.MethodPost, "text/html", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			req.Header.Set("Accept", tt.accept)
			if got := isHTMLNavigation(req); got != tt.want {
				t.Errorf("isHTMLNavigation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeNextPath(t *testing.T) {
	tests := map[string]string{
		"":                      "/",
		"/":                     "/",
		"/tasks?filter=all":     "/tasks?filter=all",
		"//evil.example.com":    "/",
		"https://evil.example":  "/",
		"tasks":                 "/",
		"/login?next=/settings": "/login?next=/settings",
	}
	for in, want := range tests {
		if got := safeNextPath(in); got != want {
			t.Errorf("safeNextPath(%q) = %q, want %q", in, got, want)
		}
	}
}
