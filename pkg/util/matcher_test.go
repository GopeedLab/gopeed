package util

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern string
		urls    []string
		want    bool
	}{
		{"*://*/*", []string{"https://www.google.com/", "http://example.org/foo/bar.html"}, true},
		{"https://*/*", []string{"https://www.google.com", "https://example.org/foo/bar.html"}, true},
		{"*://www.google.com", []string{"https://www.google.com/", "https://www.google.com"}, true},
		{"*://*.google.com/", []string{"https://a.www.google.com/", "https://c.www.google.com/", "https://www.google.com/"}, true},
		{"https://*/foo*", []string{"https://www.google.com/foo", "https://example.com/foo/bar.html"}, true},
		{"https://www.google.com/*/b/*", []string{"https://www.google.com/a/b", "https://www.google.com/a/b/c"}, true},
		{"https://*.google.com/foo*bar", []string{"https://www.google.com/foo/baz/bar", "https://docs.google.com/foobar"}, true},
		{"https://www.google.com/*abc*", []string{"https://www.google.com/abc", "https://www.google.com/123abc", "https://www.google.com/abc456", "https://www.google.com/123abc456"}, true},
		{"https://example.org/foo/bar.html", []string{"https://example.org/foo/bar.html"}, true},
		{"http://127.0.0.1/*", []string{"http://127.0.0.1/", "http://127.0.0.1/foo/bar.html"}, true},
		{"*://mail.google.com/*", []string{"http://mail.google.com/foo/baz/bar", "https://mail.google.com/foobar"}, true},
		{"https://www.google.com/", []string{"http://www.google.com/"}, false},
		{"www.google.com/", []string{"http://www.google.com/", "https://www.google.com/"}, false},
		{"www.google.com/*c", []string{"https://www.google.com/a", "https://www.google.com/b"}, false},
		{"https://*.example.org/*", []string{"https://www.google.com", "https://docs.google.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			for _, url := range tt.urls {
				if got := Match(tt.pattern, url); got != tt.want {
					t.Errorf("Match() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
