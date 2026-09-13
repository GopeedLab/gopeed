package engine

import (
	"bytes"
	"context"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/stream"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFFmpegJSStreams(t *testing.T) {
	video, err := os.ReadFile("testdata/ffmpeg/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	audio, err := os.ReadFile("testdata/ffmpeg/audio.mp4")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"streams", "http", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b := video
				if r.URL.Path == "/audio" {
					b = audio
				}
				http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(b))
			}))
			defer srv.Close()
			e := NewEngine(nil)
			defer e.Close()
			// Input data is deliberately interleaved through one JS producer.
			// Native push/read operations must never block the event loop.
			e.Runtime.Set("videoBytes", e.Runtime.NewArrayBuffer(video))
			e.Runtime.Set("audioBytes", e.Runtime.NewArrayBuffer(audio))
			e.Runtime.Set("sourceURL", srv.URL)
			e.Runtime.Set("inputMode", mode)
			result, err := e.RunString(`(async()=>{
		  let vc, ac, offset=0;
		  const v=new Uint8Array(videoBytes), a=new Uint8Array(audioBytes);
		  const video=new ReadableStream({start(c){vc=c;}});
		  const audio=new ReadableStream({start(c){ac=c;}});
		  const timer=setInterval(()=>{
		    if(offset<v.length)vc.enqueue(v.slice(offset,offset+97));
		    if(offset<a.length)ac.enqueue(a.slice(offset,offset+97));
		    offset+=97;
		    if(offset>=Math.max(v.length,a.length)){clearInterval(timer);vc.close();ac.close();}
		  },1);
		  const stream=__gopeed_ffmpeg.merge({
		    video:inputMode==='http'?{url:sourceURL+'/video'}:video,
		    audio:inputMode==='streams'?audio:{url:sourceURL+'/audio'},
		    args:['-metadata','title=gopeed-test']
		  });
		  const bytes=await new Response(stream).arrayBuffer();
		  return new Uint8Array(bytes);
		})()`)
			if err != nil {
				t.Fatal(err)
			}
			out, ok := result.([]byte)
			if !ok {
				t.Fatalf("output %T", result)
			}
			if !bytes.Contains(out, []byte("moof")) || !bytes.Contains(out, []byte("gopeed-test")) {
				t.Fatalf("invalid output (%d bytes)", len(out))
			}
		})
	}
}

func TestFFmpegJSErrorAndCancel(t *testing.T) {
	for _, scenario := range []string{"input-error", "abort", "output-cancel", "invalid-args"} {
		t.Run(scenario, func(t *testing.T) {
			e := NewEngine(nil)
			defer e.Close()
			e.Runtime.Set("scenario", scenario)
			result, err := e.RunString(`(async()=>{
		 let cancelled=0;
		 const source=()=>new ReadableStream({
		   pull(c){if(scenario==='input-error')c.error(new Error('upstream failed'));else return new Promise(()=>{});},
		   cancel(){cancelled++;}
		 });
		 const controller=new AbortController();
		 const output=__gopeed_ffmpeg.merge({video:source(),audio:source(),signal:controller.signal,
		   args:scenario==='invalid-args'?['-i','/outside']:[]});
		 const reader=output.getReader();
		 const pending=reader.read().then(()=>'',e=>e.message);
		 if(scenario==='abort')setTimeout(()=>controller.abort(),20);
		 if(scenario==='output-cancel')setTimeout(()=>reader.cancel(),20);
		 const message=await pending;
		 await new Promise(r=>setTimeout(r,10));
		 return {message,cancelled};
		})()`)
			if err != nil {
				t.Fatal(err)
			}
			m := result.(map[string]any)
			if scenario != "output-cancel" && m["message"] == "" {
				t.Fatalf("missing error: %v", m)
			}
			if scenario == "abort" || scenario == "output-cancel" {
				if fmt.Sprint(m["cancelled"]) != "2" {
					t.Fatalf("inputs not cancelled: %v", m)
				}
			}
		})
	}
}

func TestFFmpegMalformedMediaFails(t *testing.T) {
	e := NewEngine(nil)
	defer e.Close()
	_, err := e.RunString(`(async()=>{
	  const source=()=>new ReadableStream({start(c){c.enqueue(new Uint8Array([1,2,3]));c.close();}});
	  return await new Response(__gopeed_ffmpeg.merge({video:source(),audio:source()})).arrayBuffer();
	})()`)
	if err == nil || !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("expected FFmpeg error: %v", err)
	}
}

func TestFFmpegBlobOpener(t *testing.T) {
	var open stream.ObjectURLOpener
	e := NewEngine(&Config{StreamConfig: &stream.Config{CreateObjectURL: func(opts *stream.ObjectURLOptions, opener stream.ObjectURLOpener) (string, error) {
		if opts.Range || opts.Size != 0 {
			t.Fatal("merged output must not advertise range or a guessed size")
		}
		open = opener
		return "blob:ffmpeg-test", nil
	}}})
	defer e.Close()
	video, _ := os.ReadFile("testdata/ffmpeg/video.mp4")
	audio, _ := os.ReadFile("testdata/ffmpeg/audio.mp4")
	e.Runtime.Set("videoBytes", e.Runtime.NewArrayBuffer(video))
	e.Runtime.Set("audioBytes", e.Runtime.NewArrayBuffer(audio))
	_, err := e.RunString(`__gopeed_blob_create_object_url(()=>{
	  const source=b=>new ReadableStream({start(c){c.enqueue(new Uint8Array(b));c.close();}});
	  return __gopeed_ffmpeg.merge({video:source(videoBytes),audio:source(audioBytes)});
	},{contentType:'video/mp4'})`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		r, err := open(ctx, stream.ObjectURLOpenRequest{End: -1})
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		cancel()
		if err != nil || !bytes.Contains(b, []byte("moof")) {
			t.Fatalf("Blob read: %v (%d bytes)", err, len(b))
		}
	}
}

func TestFFmpegEngineCloseCancelsRequest(t *testing.T) {
	started, closed := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(closed)
	}))
	defer srv.Close()
	e := NewEngine(nil)
	defer e.Close()
	_, err := e.RunString(fmt.Sprintf(`
	globalThis.cancelCount=0;
	const audio=new ReadableStream({cancel(){cancelCount++;}});
	__gopeed_ffmpeg.merge({video:{url:%q},audio}).getReader().read().catch(()=>{});
	"started";`, srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP did not start")
	}
	e.Close()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP leaked after engine shutdown")
	}
	if e.Runtime.Get("cancelCount").ToInteger() != 1 {
		t.Fatal("JS input not cancelled on shutdown")
	}
}
