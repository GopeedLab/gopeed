package hls

import (
	_ "embed"
	"net/url"
	"testing"
)

//go:embed testdata/olemovienews.m3u8
var realWorldPlaylist string

// TestParseMediaRealWorld guards parsing against a real site playlist
// (olemovienews.com VOD, 138 TS segments).
func TestParseMediaRealWorld(t *testing.T) {
	base, err := url.Parse("https://europe.olemovienews.com/ts2/20230125/afAxfjit/mp4/afAxfjit.mp4/index-v1-a1.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	media, err := ParseMedia(realWorldPlaylist, base)
	if err != nil {
		t.Fatal(err)
	}
	if media.Live {
		t.Error("real VOD playlist should not be live")
	}
	if media.IsFMP4 {
		t.Error("playlist should be detected as TS")
	}
	if len(media.Segments) != 138 {
		t.Errorf("want 138 segments, got %d", len(media.Segments))
	}
	first := media.Segments[0]
	if first.URI != "https://europe.olemovienews.com/ts2/20230125/afAxfjit/mp4/afAxfjit.mp4/seg-1-v1-a1.ts" {
		t.Errorf("segment URI resolution wrong: %s", first.URI)
	}
	if media.Segments[0].Sequence != 1 {
		t.Errorf("first sequence = %d, want 1", media.Segments[0].Sequence)
	}
}
