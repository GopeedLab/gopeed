package ffmpeg

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
)

// Generate tiny deterministic media with the exact embedded build under test;
// no system FFmpeg, network media or third-party fixture licensing is needed.
func fixture(t *testing.T, audio bool) []byte {
	t.Helper()
	initialize()
	if shared.err != nil {
		t.Fatal(shared.err)
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=black:s=32x32:r=10", "-t", "0.3", "-c:v", "libx264", "-preset", "ultrafast", "-movflags", "frag_keyframe+empty_moov", "-f", "mp4", "pipe:1"}
	if audio {
		args = []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "anullsrc=r=48000:cl=mono", "-t", "0.3", "-c:a", "aac", "-movflags", "frag_keyframe+empty_moov", "-f", "mp4", "pipe:1"}
	}
	var out, logs bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rc, err := wasm.Run(ctx, shared.runtime, shared.module, wasm.Args{Name: "ffmpeg", Args: args, Stdout: &out, Stderr: &logs})
	if err != nil || rc != 0 {
		t.Fatalf("fixture: %d %v %s", rc, err, logs.String())
	}
	return out.Bytes()
}

func TestTailMoovNeedsRange(t *testing.T) {
	initialize()
	if shared.err != nil {
		t.Fatal(shared.err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	var logs bytes.Buffer
	rc, err := wasm.Run(ctx, shared.runtime, shared.module, wasm.Args{Name: "ffmpeg", Args: []string{
		"-v", "error", "-f", "lavfi", "-i", "color=c=black:s=32x32:r=10", "-t", "0.3", "-c:v", "libx264", "-preset", "ultrafast", "/out/video.mp4",
	}, Stderr: &logs, Config: func(c wazero.ModuleConfig) wazero.ModuleConfig {
		return c.WithFSConfig(wazero.NewFSConfig().WithDirMount(dir, "/out"))
	}})
	if rc != 0 || err != nil {
		t.Fatalf("fixture %d %v %s", rc, err, logs.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "video.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	var video []byte
	for offset := 0; offset < len(raw); {
		length := int(binary.BigEndian.Uint32(raw[offset : offset+4]))
		if string(raw[offset+4:offset+8]) == "moov" {
			free := make([]byte, 2*httpBlockSize)
			binary.BigEndian.PutUint32(free, uint32(len(free)))
			copy(free[4:], "free")
			video = append(video, free...)
		}
		video = append(video, raw[offset:offset+length]...)
		offset += length
	}
	audio := fixture(t, true)
	for _, ranged := range []bool{true, false} {
		t.Run(fmtBool(ranged), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data := video
				if r.URL.Path == "/audio" {
					data = audio
				}
				if ranged {
					http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(data))
				} else {
					w.Write(data)
				}
			}))
			defer srv.Close()
			v, err := OpenHTTP(ctx, srv.Client(), HTTPSource{URL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			a, err := OpenHTTP(ctx, srv.Client(), HTTPSource{URL: srv.URL + "/audio"})
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			err = Run(ctx, v, a, io.Discard, "mp4", nil)
			if ranged && err != nil {
				t.Fatal(err)
			}
			if !ranged && err == nil {
				t.Fatal("non-seekable tail-moov input unexpectedly succeeded")
			}
		})
	}
}

func fmtBool(b bool) string {
	if b {
		return "range"
	}
	return "sequential"
}

func TestWebMToMP4(t *testing.T) {
	initialize()
	if shared.err != nil {
		t.Fatal(shared.err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inputs := make([]*StreamInput, 2)
	for i, name := range []string{"video.webm", "audio.webm"} {
		data, err := os.ReadFile(filepath.Join("..", "testdata", "ffmpeg", name))
		if err != nil {
			t.Fatal(err)
		}
		inputs[i] = NewStreamInput(ctx, "")
		defer inputs[i].Close()
		if _, err = inputs[i].Write(data); err != nil {
			t.Fatal(err)
		}
		inputs[i].End(nil)
	}

	var output, probe, logs bytes.Buffer
	if err := Run(ctx, inputs[0], inputs[1], &output, "mp4", nil); err != nil {
		t.Fatal(err)
	}
	rc, err := wasm.Run(ctx, shared.runtime, shared.module, wasm.Args{Name: "ffprobe", Args: []string{"-v", "error", "-show_entries", "stream=codec_name", "-of", "json", "pipe:0"}, Stdin: bytes.NewReader(output.Bytes()), Stdout: &probe, Stderr: &logs})
	if rc != 0 || err != nil {
		t.Fatalf("probe %d %v %s", rc, err, logs.String())
	}
	if !bytes.Contains(probe.Bytes(), []byte(`"vp9"`)) || !bytes.Contains(probe.Bytes(), []byte(`"opus"`)) {
		t.Fatalf("codecs changed: %s", probe.String())
	}
}

func TestWASMMerge(t *testing.T) {
	video, audio := fixture(t, false), fixture(t, true)
	for _, ranged := range []bool{false, true} {
		t.Run(map[bool]string{false: "sequential", true: "range"}[ranged], func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data := video
				if r.URL.Path == "/audio" {
					data = audio
				}
				if ranged {
					http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(data))
				} else {
					w.Write(data)
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			v, err := OpenHTTP(ctx, srv.Client(), HTTPSource{URL: srv.URL + "/video"})
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			a, err := OpenHTTP(ctx, srv.Client(), HTTPSource{URL: srv.URL + "/audio"})
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			var output bytes.Buffer
			if err = Run(ctx, v, a, &output, "", nil); err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(output.Bytes(), []byte("moof")) {
				t.Fatal("output is not fragmented MP4")
			}
			// Probe the actual result, rather than just checking its filename/header.
			var probe, logs bytes.Buffer
			rc, err := wasm.Run(ctx, shared.runtime, shared.module, wasm.Args{Name: "ffprobe", Args: []string{"-v", "error", "-show_streams", "-of", "json", "pipe:0"}, Stdin: bytes.NewReader(output.Bytes()), Stdout: &probe, Stderr: &logs})
			if rc != 0 || err != nil {
				t.Fatalf("probe: %d %v %s", rc, err, logs.String())
			}
			var result struct {
				Streams []struct {
					CodecType string `json:"codec_type"`
					CodecName string `json:"codec_name"`
				}
			}
			if err = json.Unmarshal(probe.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Streams) != 2 || result.Streams[0].CodecName != "h264" || result.Streams[1].CodecName != "aac" {
				t.Fatalf("unexpected streams %s", probe.String())
			}
		})
	}
	// A malformed input must produce an error, not successful empty output.
	s := NewStreamInput(context.Background(), "")
	defer s.Close()
	s.Write([]byte("invalid"))
	s.End(nil)
	if err := Run(context.Background(), s, s, io.Discard, "", nil); err == nil {
		t.Fatal("accepted invalid media")
	}
}
