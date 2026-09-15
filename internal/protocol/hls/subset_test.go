package hls

import (
	"net/url"
	"strings"
	"testing"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// subsetCase is a media playlist plus the error substring the pipeline must
// reject it with; wantOK playlists must parse AND validate cleanly.
type subsetCase struct {
	name     string
	playlist string
	wantErr  string
}

func TestSubsetRejections(t *testing.T) {
	cases := []subsetCase{
		{
			name:     "gap",
			playlist: "#EXTM3U\n#EXTINF:4,\na.ts\n#EXT-X-GAP\n#EXTINF:4,\nb.ts\n#EXT-X-ENDLIST\n",
			wantErr:  "EXT-X-GAP",
		},
		{
			name:     "gap-before-first-segment",
			playlist: "#EXTM3U\n#EXT-X-GAP\n#EXTINF:4,\na.ts\n#EXT-X-ENDLIST\n",
			wantErr:  "EXT-X-GAP",
		},
		{
			name:     "discontinuity",
			playlist: "#EXTM3U\n#EXTINF:4,\na.ts\n#EXT-X-DISCONTINUITY\n#EXTINF:4,\nb.ts\n#EXT-X-ENDLIST\n",
			wantErr:  "EXT-X-DISCONTINUITY",
		},
		{
			name:     "live",
			playlist: "#EXTM3U\n#EXTINF:4,\na.ts\n",
			wantErr:  "live",
		},
		{
			name:     "keyformat",
			playlist: "#EXTM3U\n#EXT-X-KEY:METHOD=AES-128,URI=\"k\",KEYFORMAT=\"com.apple.streamingkeydelivery\"\n#EXTINF:4,\na.ts\n#EXT-X-ENDLIST\n",
			wantErr:  "KEYFORMAT",
		},
		{
			name:     "encrypted-map-without-iv",
			playlist: "#EXTM3U\n#EXT-X-KEY:METHOD=AES-128,URI=\"k\"\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:4,\na.m4s\n#EXT-X-ENDLIST\n",
			wantErr:  "explicit IV",
		},
		{
			name:     "map-change-byterange",
			playlist: "#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\",BYTERANGE=\"100@0\"\n#EXTINF:4,\na.m4s\n#EXT-X-MAP:URI=\"init.mp4\",BYTERANGE=\"100@100\"\n#EXTINF:4,\nb.m4s\n#EXT-X-ENDLIST\n",
			wantErr:  "changing init sections",
		},
		{
			name:     "map-change-uri",
			playlist: "#EXTM3U\n#EXT-X-MAP:URI=\"init1.mp4\"\n#EXTINF:4,\na.m4s\n#EXT-X-MAP:URI=\"init2.mp4\"\n#EXTINF:4,\nb.m4s\n#EXT-X-ENDLIST\n",
			wantErr:  "changing init sections",
		},
	}
	baseURL := mustParseURL(t, "https://cdn.example.com/vod/index.m3u8")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			media, parseErr := ParseMedia(tc.playlist, baseURL)
			if parseErr != nil {
				if !strings.Contains(parseErr.Error(), tc.wantErr) {
					t.Fatalf("parse error %q does not mention %q", parseErr, tc.wantErr)
				}
				return
			}
			err := validateSupported(media)
			if err == nil {
				t.Fatalf("playlist should be rejected with %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestSubsetRepeatedSameMapAllowed(t *testing.T) {
	playlist := "#EXTM3U\n" +
		"#EXT-X-MAP:URI=\"init.mp4\",BYTERANGE=\"100@0\"\n" +
		"#EXTINF:4,\na.m4s\n" +
		"#EXT-X-MAP:URI=\"init.mp4\",BYTERANGE=\"100@0\"\n" +
		"#EXTINF:4,\nb.m4s\n" +
		"#EXT-X-ENDLIST\n"
	media, err := ParseMedia(playlist, mustParseURL(t, "https://cdn.example.com/vod/index.m3u8"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateSupported(media); err != nil {
		t.Fatalf("identical repeated MAP must be allowed: %v", err)
	}
	if len(media.Segments) != 3 { // init + 2 segments, no duplicate init
		t.Fatalf("want 3 segments (init + 2), got %d", len(media.Segments))
	}
}

func TestMasterAudioRenditionDetection(t *testing.T) {
	masterURL := mustParseURL(t, "https://cdn.example.com/master.m3u8")

	separateAudio := "#EXTM3U\n" +
		"#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",NAME=\"en\",DEFAULT=YES,URI=\"audio/index.m3u8\"\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=1000000,AUDIO=\"aud\"\nvideo/index.m3u8\n"
	master, err := ParseMasterPlaylist(separateAudio, masterURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAudioRendition(master, PickBestVariant(master.Variants)); err == nil {
		t.Fatal("selected variant with a separate audio rendition must be rejected")
	}

	// A group the variant does not reference must not cause a false positive.
	unrelatedGroup := "#EXTM3U\n" +
		"#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",NAME=\"en\",DEFAULT=YES,URI=\"audio/index.m3u8\"\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=1000000\nvideo/index.m3u8\n"
	master, err = ParseMasterPlaylist(unrelatedGroup, masterURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAudioRendition(master, PickBestVariant(master.Variants)); err != nil {
		t.Fatalf("unreferenced audio group must not be rejected: %v", err)
	}

	// A rendition group without URIs (muxed tracks) is fine.
	groupWithoutURI := "#EXTM3U\n" +
		"#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",NAME=\"en\",DEFAULT=YES\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=1000000,AUDIO=\"aud\"\nvideo/index.m3u8\n"
	master, err = ParseMasterPlaylist(groupWithoutURI, masterURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAudioRendition(master, PickBestVariant(master.Variants)); err != nil {
		t.Fatalf("audio group without URI must not be rejected: %v", err)
	}
}
