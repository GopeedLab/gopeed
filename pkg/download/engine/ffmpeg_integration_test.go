package engine

import (
	"bytes"
	"context"
	"fmt"
	media "github.com/GopeedLab/gopeed/pkg/download/engine/ffmpeg"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/stream"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
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
		    inputs: () => ({ video:inputMode==='http'?{url:sourceURL+'/video'}:video,
		    audio:inputMode==='streams'?audio:{url:sourceURL+'/audio'} }),
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
		 const output=__gopeed_ffmpeg.merge({inputs:()=>({video:source(),audio:source()}),signal:controller.signal,
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
	  return await new Response(__gopeed_ffmpeg.merge({inputs:()=>({video:source(),audio:source()})})).arrayBuffer();
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
	  return __gopeed_ffmpeg.merge({inputs:()=>({video:source(videoBytes),audio:source(audioBytes)})});
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
	__gopeed_ffmpeg.merge({inputs:()=>({video:{url:%q},audio})}).getReader().read().catch(()=>{});
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

func TestFFmpegInputFactoriesWaitForCapacity(t *testing.T) {
	for _, mode := range []string{"factory", "http", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			release1, _ := media.Acquire(context.Background())
			defer release1()
			release2, _ := media.Acquire(context.Background())
			defer release2()
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); http.Error(w, "test", 500) }))
			defer srv.Close()
			e := NewEngine(nil)
			defer e.Close()
			e.Runtime.Set("mode", mode)
			e.Runtime.Set("sourceURL", srv.URL)
			result, err := e.RunString(`(async()=>{
     globalThis.factoryCalls=0;
     globalThis.abortQueue=new AbortController();
     const options=mode==='http'?{video:{url:sourceURL},audio:{url:sourceURL}}:
       {inputs:()=>{factoryCalls++;throw new Error('factory failed');}};
     options.signal=abortQueue.signal;
     globalThis.work=__gopeed_ffmpeg.merge(options).getReader().read().then(()=>'',e=>e.message);
     await new Promise(r=>setTimeout(r,50));
     if(mode==='cancel')abortQueue.abort();
     return factoryCalls;
   })()`)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(result) != "0" || requests.Load() != 0 {
				t.Fatalf("queued inputs started: %v, requests=%d", result, requests.Load())
			}
			if mode == "cancel" {
				if _, err = e.RunString(`work`); err != nil {
					t.Fatal(err)
				}
			}
			release1()
			if _, err = e.RunString(`work`); err != nil {
				t.Fatal(err)
			}
			calls, err := e.RunString(`factoryCalls`)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "factory" && fmt.Sprint(calls) != "1" {
				t.Fatal(calls)
			}
			if mode == "cancel" && fmt.Sprint(calls) != "0" {
				t.Fatal("cancelled factory ran")
			}
			if mode == "http" && requests.Load() != 1 {
				t.Fatal("HTTP did not start after admission")
			}
			// Failure/cancellation must return the acquired slot even with the other held.
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			release, err := media.Acquire(ctx)
			if err != nil {
				t.Fatalf("slot leaked: %v", err)
			}
			release()
		})
	}
}

func TestFFmpegRequiresFactoryForStreams(t *testing.T) {
	e := NewEngine(nil)
	defer e.Close()
	result, err := e.RunString(`(()=>{
  const source=()=>new ReadableStream({});
  let rejected=0;
  for(const opts of [
   {video:source(),audio:source()},
   {video:{url:'https://example.test'},audio:source()},
   {inputs:()=>({}),video:{url:'https://example.test'}},
   {inputs:{video:source(),audio:source()}}
  ]) {try{__gopeed_ffmpeg.merge(opts);}catch(e){if(e instanceof TypeError)rejected++;}}
  return rejected;
 })()`)
	if err != nil || fmt.Sprint(result) != "4" {
		t.Fatalf("validation: %v %v", result, err)
	}
}

func TestFFmpegCancelPendingFactoryCleansLateInputs(t *testing.T) {
	e := NewEngine(nil)
	defer e.Close()
	result, err := e.RunString(`(async()=>{
   let ready,finish,cancelled=0,signal;
   const entered=new Promise(r=>ready=r);
   const abort=new AbortController();
   const output=__gopeed_ffmpeg.merge({signal:abort.signal,inputs:async opts=>{
     signal=opts.signal;ready();await new Promise(r=>finish=r);
     const source=()=>new ReadableStream({cancel(){cancelled++;}});
     return {video:source(),audio:source()};
   }});
   const work=output.getReader().read().catch(e=>e.name);
   await entered;abort.abort();await work;finish();
   await new Promise(r=>setTimeout(r,30));
   return {cancelled,aborted:signal.aborted};
 })()`)
	if err != nil {
		t.Fatal(err)
	}
	state := result.(map[string]any)
	if fmt.Sprint(state["cancelled"]) != "2" || state["aborted"] != true {
		t.Fatal(state)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	first, err := media.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := media.Acquire(ctx)
	if err != nil {
		t.Fatal("factory cancellation leaked capacity")
	}
	second()
}
