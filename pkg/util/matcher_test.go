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

func TestMatchMagnet(t *testing.T) {
	const magnetURL = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=example.zip"
	tests := []struct {
		name    string
		pattern string
		url     string
		want    bool
	}{
		{"scheme wildcard", "magnet:*", magnetURL, true},
		{"query wildcard", "magnet:?*", magnetURL, true},
		{"info hash wildcard", "magnet:?xt=urn:btih:*", magnetURL, true},
		{"exact URI", magnetURL, magnetURL, true},
		{"encoded name", "magnet:?*&dn=example%20file.zip", "magnet:?xt=urn:btih:abc&dn=example%20file.zip", true},
		{"literal query punctuation", "magnet:?*&dn=a+b[1].zip", "magnet:?xt=urn:btih:abc&dn=a+b[1].zip", true},
		{"literal question mark", "magnet:?*", "magnet:xt=urn:btih:abc", false},
		{"literal dot", "magnet:?*&dn=example.zip", "magnet:?xt=urn:btih:abc&dn=exampleXzip", false},
		{"different hash type", "magnet:?xt=urn:btmh:*", magnetURL, false},
		{"different hash", "magnet:?xt=urn:btih:other*", magnetURL, false},
		{"different scheme", "magnet:*", "https://example.org/file.zip", false},
		{"invalid URI", "magnet:*", "magnet:\x00", false},
		{"existing catch-all", "*://*/*", magnetURL, true},
		{"existing magnet pattern", "magnet://*", magnetURL, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.url); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.url, got, tt.want)
			}
		})
	}
}
