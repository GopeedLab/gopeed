package hls

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
)

// The validators below check real media structure, not just byte
// concatenation: TS packets are verified on the 188 byte grid with
// continuity-counter rules, PAT/PMT presence and PES start codes; fMP4
// output is verified by walking the ISO-BMFF box hierarchy.

func be32(b []byte) int {
	return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
}

// validateTS asserts structural validity of a merged MPEG-2 TS stream.
func validateTS(t *testing.T, data []byte) {
	t.Helper()
	if len(data) == 0 || len(data)%188 != 0 {
		t.Fatalf("TS size %d is not a positive multiple of 188", len(data))
	}
	type pidState struct{ cc int }
	ccByPID := map[int]int{}
	patSeen, pmtSeen, pesSeen := false, false, false
	for off := 0; off < len(data); off += 188 {
		pkt := data[off : off+188]
		if pkt[0] != 0x47 {
			t.Fatalf("bad TS sync byte at offset %d", off)
		}
		pid := (int(pkt[1]&0x1F) << 8) | int(pkt[2])
		pusi := pkt[1]&0x40 != 0
		afc := (pkt[3] >> 4) & 0x3
		if afc == 0 {
			continue // reserved
		}
		idx := 4
		if afc&0x2 != 0 { // adaptation field present
			afLen := int(pkt[4])
			if afLen > 0 && pkt[5]&0x80 != 0 {
				// discontinuity_indicator: restart the CC expectation
				delete(ccByPID, pid)
			}
			idx += 1 + afLen
			if afc == 2 {
				continue // adaptation only, no payload: no CC rule
			}
		}
		if idx > 188 {
			t.Fatalf("adaptation field overruns packet at offset %d", off)
		}
		if pid == 0x1FFF {
			continue // null packet
		}
		payload := pkt[idx:]
		if pusi && len(payload) >= 3 && pid != 0 && payload[0] == 0 && payload[1] == 0 && payload[2] == 1 {
			pesSeen = true
		}
		if pid == 0 {
			if ptr := int(payload[0]); 1+ptr < len(payload) && payload[1+ptr] == 0x00 {
				patSeen = true // program_association_section table_id
			}
		} else if pusi && len(payload) >= 2 && payload[0] == 0 && payload[1] == 0x02 {
			pmtSeen = true // program_map_section table_id
		}
		cc := int(pkt[3] & 0x0F)
		prev, ok := ccByPID[pid]
		if !ok {
			ccByPID[pid] = cc
			continue
		}
		// CC increments by one per payload-bearing packet; a repeated value
		// models the single legal duplicate packet.
		if cc != (prev+1)%16 && cc != prev {
			t.Fatalf("continuity error on PID %d: %d after %d at offset %d", pid, cc, prev, off)
		}
		ccByPID[pid] = cc
	}
	if !patSeen {
		t.Error("no PAT found in merged TS")
	}
	if !pmtSeen {
		t.Error("no PMT found in merged TS")
	}
	if !pesSeen {
		t.Error("no PES start code found in merged TS")
	}
}

// walkBoxes iterates the top-level ISO-BMFF boxes of data[off:end].
func walkBoxes(t *testing.T, data []byte, off, end int) []string {
	t.Helper()
	var types []string
	for off+8 <= end {
		size := be32(data[off:])
		typ := string(data[off+4 : off+8])
		if size < 8 || off+size > end {
			t.Fatalf("invalid box %q at offset %d (size %d)", typ, off, size)
		}
		types = append(types, typ)
		off += size
	}
	return types
}

// validateFMP4 asserts the merged fMP4 output is init section followed by
// well-formed moof+mdat fragments.
func validateFMP4(t *testing.T, merged []byte, wantFragments int) {
	t.Helper()
	types := walkBoxes(t, merged, 0, len(merged))
	hasFtyp, hasMoov := false, false
	fragments := 0
	expectFragment := false
	for i, typ := range types {
		switch typ {
		case "ftyp":
			if i != 0 {
				t.Errorf("ftyp not at stream start")
			}
			hasFtyp = true
		case "moov":
			if i != 1 {
				t.Errorf("moov must follow ftyp, found at index %d", i)
			}
			hasMoov = true
		case "styp":
			expectFragment = true
		case "moof":
			if !hasMoov {
				t.Errorf("fragment before moov")
			}
			expectFragment = true
		case "mdat":
			if !expectFragment {
				t.Errorf("mdat outside of a fragment at index %d", i)
			}
			expectFragment = false
			fragments++
		case "mfra":
			// optional trailer
		default:
			t.Errorf("unexpected top-level box %q", typ)
		}
	}
	if !hasFtyp || !hasMoov {
		t.Error("merged fMP4 missing ftyp/moov init section")
	}
	if fragments != wantFragments {
		t.Errorf("want %d fragments in merged fMP4, got %d", wantFragments, fragments)
	}
}

func mediaFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/media/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func serveFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("testdata/media"))
	mux.Handle("/", fs)
	return httptest.NewServer(mux)
}

func TestHLSRealMedia_TS(t *testing.T) {
	seg0 := mediaFixture(t, "sample-00.ts")
	seg1 := mediaFixture(t, "sample-01.ts")
	playlist := "#EXTM3U\n" +
		"#EXTINF:0.5,\nsample-00.ts\n" +
		"#EXTINF:0.5,\nsample-01.ts\n" +
		"#EXT-X-ENDLIST\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist))
	})
	mux.Handle("/", http.FileServer(http.Dir("testdata/media")))
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	if err := f.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	got := readOutput(t, f)
	if !bytes.Equal(got, append(append([]byte{}, seg0...), seg1...)) {
		t.Fatal("merged TS is not the exact concatenation of the segments")
	}
	validateTS(t, got)
}

func TestHLSRealMedia_FMP4(t *testing.T) {
	init := mediaFixture(t, "fmp4-init.mp4")
	frag1 := mediaFixture(t, "fmp4-seg1.m4s")
	frag2 := mediaFixture(t, "fmp4-seg2.m4s")
	frag3 := mediaFixture(t, "fmp4-seg3.m4s")
	playlist := "#EXTM3U\n" +
		"#EXT-X-MAP:URI=\"fmp4-init.mp4\"\n" +
		"#EXTINF:0.5,\nfmp4-seg1.m4s\n" +
		"#EXTINF:0.5,\nfmp4-seg2.m4s\n" +
		"#EXTINF:0.5,\nfmp4-seg3.m4s\n" +
		"#EXT-X-ENDLIST\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist))
	})
	mux.Handle("/", http.FileServer(http.Dir("testdata/media")))
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	if err := f.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	got := readOutput(t, f)
	want := append(append(append(append([]byte{}, init...), frag1...), frag2...), frag3...)
	if !bytes.Equal(got, want) {
		t.Fatal("merged fMP4 is not the exact concatenation of init + fragments")
	}
	validateFMP4(t, got, 3)
}

func TestHLSRealMedia_AES128(t *testing.T) {
	keys := [][]byte{[]byte("key-A-0123456789"), []byte("key-B-9876543210")}
	iv := []byte("iv-1337-01234567")
	plain := [][]byte{mediaFixture(t, "sample-00.ts"), mediaFixture(t, "sample-01.ts")}

	playlist := "#EXTM3U\n" +
		"#EXT-X-MEDIA-SEQUENCE:0\n" +
		"#EXT-X-KEY:METHOD=AES-128,URI=\"key-0.bin\",IV=0x" + hexEncode(iv) + "\n" +
		"#EXTINF:0.5,\nenc-0.ts\n" +
		"#EXT-X-KEY:METHOD=AES-128,URI=\"key-1.bin\"\n" +
		"#EXTINF:0.5,\nenc-1.ts\n" +
		"#EXT-X-ENDLIST\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/video.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(playlist))
	})
	for i, key := range keys {
		k := key
		mux.HandleFunc(fmt.Sprintf("/key-%d.bin", i), func(w http.ResponseWriter, r *http.Request) {
			w.Write(k)
		})
		// Segment 0 has an explicit IV, segment 1 derives it from its media
		// sequence number (1).
		enc := encryptSegment(plain[i], k, map[bool][]byte{true: iv, false: sequenceIV(int64(i))}[i == 0])
		mux.HandleFunc(fmt.Sprintf("/enc-%d.ts", i), func(w http.ResponseWriter, r *http.Request) {
			w.Write(enc)
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	f := newTestFetcher(t, testConfig())
	if err := f.Resolve(&base.Request{URL: server.URL + "/video.m3u8"}, &base.Options{Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if f.meta.Res.Size != 0 {
		t.Errorf("encrypted plan must report indeterminate size, got %d", f.meta.Res.Size)
	}
	if err := f.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Wait(); err != nil {
		t.Fatal(err)
	}
	got := readOutput(t, f)
	if !bytes.Equal(got, append(append([]byte{}, plain[0]...), plain[1]...)) {
		t.Fatal("decrypted output mismatch")
	}
	validateTS(t, got)
	if p := f.Progress(); p.TotalDownloaded() != int64(len(got)) {
		t.Errorf("final progress %d != merged size %d", p.TotalDownloaded(), len(got))
	}
}
