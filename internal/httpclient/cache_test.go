package httpclient

import (
	"testing"
	"time"
)

func TestBrowserSelectionCacheExpiresEntries(t *testing.T) {
	now := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC)
	cache := newBrowserSelectionCache(7*24*time.Hour, func() time.Time {
		return now
	})

	cache.store("https://example.com", BrowserChrome)
	if browser, ok := cache.load("https://example.com"); !ok || browser != BrowserChrome {
		t.Fatalf("load() = (%q, %t), want (%q, true)", browser, ok, BrowserChrome)
	}

	now = now.Add(7 * 24 * time.Hour)
	if browser, ok := cache.load("https://example.com"); ok {
		t.Fatalf("load() = (%q, true), want expired entry", browser)
	}
}

func TestBrowserSelectionCachePurgesExpiredEntriesOnStore(t *testing.T) {
	now := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC)
	cache := newBrowserSelectionCache(time.Hour, func() time.Time {
		return now
	})

	cache.store("https://expired.example", BrowserChrome)
	now = now.Add(time.Hour)
	cache.store("https://active.example", BrowserFirefox)

	if _, ok := cache.entries["https://expired.example"]; ok {
		t.Fatal("expired entry was not purged")
	}
	if selection, ok := cache.entries["https://active.example"]; !ok || selection.browser != BrowserFirefox {
		t.Fatalf("active entry = (%v, %t), want Firefox", selection, ok)
	}
}
