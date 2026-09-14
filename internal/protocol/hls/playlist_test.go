package hls

import (
	"net/url"
	"testing"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestParseMediaTS(t *testing.T) {
	content := "#EXTM3U\n" +
		"#EXT-X-TARGETDURATION:20\n" +
		"#EXT-X-VERSION:3\n" +
		"#EXT-X-MEDIA-SEQUENCE:1\n" +
		"#EXTINF:20.000,\n" +
		"seg-1.ts\n" +
		"#EXTINF:20.000,\n" +
		"seg-2.ts\n" +
		"#EXT-X-ENDLIST\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if media.Live {
		t.Error("VOD playlist should not be live")
	}
	if media.IsFMP4 {
		t.Error("TS playlist should not be fMP4")
	}
	if len(media.Segments) != 2 {
		t.Fatalf("want 2 segments, got %d", len(media.Segments))
	}
	want := "https://cdn.example.com/vod/seg-1.ts"
	if media.Segments[0].URI != want {
		t.Errorf("want uri %s, got %s", want, media.Segments[0].URI)
	}
	if media.Segments[0].Sequence != 1 {
		t.Errorf("want sequence 1, got %d", media.Segments[0].Sequence)
	}
	if media.Segments[0].Key != nil {
		t.Error("segment should be unencrypted")
	}
}

func TestParseMediaWithoutExtinf(t *testing.T) {
	// Some playlists list segment URIs without EXTINF lines.
	content := "#EXTM3U\nseg-1.ts\nseg-2.ts\n#EXT-X-ENDLIST\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/a/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(media.Segments) != 2 {
		t.Fatalf("want 2 segments, got %d", len(media.Segments))
	}
}

func TestParseMediaLive(t *testing.T) {
	content := "#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXTINF:6.0,\nseg-1.ts\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/live/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if !media.Live {
		t.Error("playlist without ENDLIST should be live")
	}
}

func TestParseMediaFMP4(t *testing.T) {
	content := "#EXTM3U\n" +
		"#EXT-X-VERSION:6\n" +
		"#EXT-X-MAP:URI=\"init.mp4\"\n" +
		"#EXTINF:5.0,\n" +
		"seg-1.m4s\n" +
		"#EXT-X-ENDLIST\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if !media.IsFMP4 {
		t.Error("playlist with MAP should be fMP4")
	}
	if len(media.Segments) != 2 {
		t.Fatalf("want 2 segments (init + 1), got %d", len(media.Segments))
	}
	if !media.Segments[0].IsInit {
		t.Error("first segment should be the init section")
	}
	if media.Segments[0].URI != "https://cdn.example.com/vod/init.mp4" {
		t.Errorf("init uri wrong: %s", media.Segments[0].URI)
	}
	if media.OutputExtension() != ".mp4" {
		t.Errorf("want .mp4, got %s", media.OutputExtension())
	}
}

func TestParseMediaMultiMapError(t *testing.T) {
	content := "#EXTM3U\n" +
		"#EXT-X-MAP:URI=\"init1.mp4\"\n" +
		"#EXTINF:5.0,\nseg-1.m4s\n" +
		"#EXT-X-DISCONTINUITY\n" +
		"#EXT-X-MAP:URI=\"init2.mp4\"\n" +
		"#EXTINF:5.0,\nseg-2.m4s\n" +
		"#EXT-X-ENDLIST\n"
	if _, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8")); err == nil {
		t.Error("multi-period fMP4 should be rejected")
	}
}

func TestParseMediaKeyRotation(t *testing.T) {
	content := "#EXTM3U\n" +
		"#EXT-X-KEY:METHOD=AES-128,URI=\"key1.bin\",IV=0x00000000000000000000000000000001\n" +
		"#EXTINF:5.0,\nseg-1.ts\n" +
		"#EXT-X-KEY:METHOD=AES-128,URI=\"key2.bin\"\n" +
		"#EXTINF:5.0,\nseg-2.ts\n" +
		"#EXT-X-KEY:METHOD=NONE\n" +
		"#EXTINF:5.0,\nseg-3.ts\n" +
		"#EXT-X-ENDLIST\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if media.Segments[0].Key == nil || media.Segments[0].Key.URI != "https://cdn.example.com/vod/key1.bin" {
		t.Fatalf("segment 1 key wrong: %+v", media.Segments[0].Key)
	}
	if len(media.Segments[0].Key.IV) != 16 || media.Segments[0].Key.IV[15] != 1 {
		t.Error("explicit IV not parsed")
	}
	if media.Segments[1].Key == nil || media.Segments[1].Key.URI != "https://cdn.example.com/vod/key2.bin" {
		t.Fatalf("segment 2 key wrong: %+v", media.Segments[1].Key)
	}
	if media.Segments[1].Key.IV != nil {
		t.Error("key without IV should leave IV nil (sequence-derived)")
	}
	if media.Segments[2].Key != nil {
		t.Error("segment 3 should be unencrypted after METHOD=NONE")
	}
}

func TestParseMediaSampleAESError(t *testing.T) {
	content := "#EXTM3U\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"skd://key\"\n#EXTINF:5.0,\nseg-1.ts\n"
	if _, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8")); err == nil {
		t.Error("SAMPLE-AES should be rejected")
	}
}

func TestParseMediaByterange(t *testing.T) {
	content := "#EXTM3U\n" +
		"#EXT-X-BYTERANGE:1000@0\nseg.ts\n" +
		"#EXT-X-BYTERANGE:2000\nseg.ts\n" +
		"#EXT-X-BYTERANGE:3000\nseg.ts\n" +
		"#EXT-X-ENDLIST\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	ranges := [][2]int64{{0, 1000}, {1000, 2000}, {3000, 3000}}
	for i, want := range ranges {
		br := media.Segments[i].Byterange
		if br == nil {
			t.Fatalf("segment %d byterange missing", i)
		}
		if br.Offset != want[0] || br.Length != want[1] {
			t.Errorf("segment %d: want offset %d length %d, got %d/%d", i, want[0], want[1], br.Offset, br.Length)
		}
	}
}

func TestParseMediaGap(t *testing.T) {
	content := "#EXTM3U\n#EXTINF:5.0,\nseg-1.ts\n#EXT-X-GAP\n#EXTINF:5.0,\nseg-2.ts\n#EXT-X-ENDLIST\n"
	media, err := ParseMedia(content, mustURL(t, "https://cdn.example.com/vod/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if media.Segments[0].Gap || !media.Segments[1].Gap {
		t.Error("gap flags wrong")
	}
}

func TestParseMediaInvalidHeader(t *testing.T) {
	if _, err := ParseMedia("not a playlist", mustURL(t, "https://cdn.example.com/a.m3u8")); err == nil {
		t.Error("invalid content should be rejected")
	}
}

func TestParseMasterPickBest(t *testing.T) {
	content := "#EXTM3U\n" +
		"#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",NAME=\"en\",DEFAULT=YES,URI=\"audio/index.m3u8\"\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360\n" +
		"360p/index.m3u8\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1920x1080,CODECS=\"avc1.640028,mp4a.40.2\"\n" +
		"1080p/index.m3u8?sign=abc\n" +
		"#EXT-X-ENDLIST\n"
	variants, err := ParseMaster(content, mustURL(t, "https://cdn.example.com/vod/master.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 2 {
		t.Fatalf("want 2 variants, got %d", len(variants))
	}
	best := PickBestVariant(variants)
	if best.Bandwidth != 2500000 {
		t.Errorf("want highest bandwidth variant, got %d", best.Bandwidth)
	}
	if best.URI != "https://cdn.example.com/vod/1080p/index.m3u8?sign=abc" {
		t.Errorf("variant uri wrong: %s", best.URI)
	}
}

func TestResolveURLQueryNotInherited(t *testing.T) {
	base := mustURL(t, "https://cdn.example.com/vod/index.m3u8?sign=abc")
	got := resolveURL(base, "seg-1.ts")
	if got != "https://cdn.example.com/vod/seg-1.ts" {
		t.Errorf("relative resolution should follow RFC3986 (no query inheritance), got %s", got)
	}
}

func TestDeriveOutputName(t *testing.T) {
	media := &Media{}
	if got := DeriveOutputName("https://cdn.example.com/ts2/a/index-v1-a1.m3u8", media); got != "index-v1-a1.ts" {
		t.Errorf("want index-v1-a1.ts, got %s", got)
	}
	fmp4 := &Media{IsFMP4: true}
	if got := DeriveOutputName("https://cdn.example.com/ts2/a/index.m3u8", fmp4); got != "index.mp4" {
		t.Errorf("want index.mp4, got %s", got)
	}
	if got := DeriveOutputName("https://cdn.example.com/play?token=x", media); got != "play.ts" {
		t.Errorf("want play.ts, got %s", got)
	}
	if got := DeriveOutputName("https://cdn.example.com/", media); got != "cdn.example.com.ts" {
		t.Errorf("want host fallback, got %s", got)
	}
}

func TestParseAttributesQuotedComma(t *testing.T) {
	attrs := parseAttributes(`URI="a,b.ts",IV=0xABCDEF00000000000000000000000001,NAME="x"`)
	if attrs["URI"] != "a,b.ts" {
		t.Errorf("quoted comma value wrong: %q", attrs["URI"])
	}
	if attrs["IV"] != "0xABCDEF00000000000000000000000001" {
		t.Errorf("hex value wrong: %q", attrs["IV"])
	}
	if attrs["NAME"] != "x" {
		t.Errorf("quoted value wrong: %q", attrs["NAME"])
	}
}
