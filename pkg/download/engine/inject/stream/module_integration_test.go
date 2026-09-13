package stream_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/stream"
)

type createdObjectURL struct {
	opts *stream.ObjectURLOptions
	open stream.ObjectURLOpener
}

func TestResponseBlobObjectURLReadsResponseStream(t *testing.T) {
	const payload = "response blob payload"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, payload[:8])
		w.(http.Flusher).Flush() // force an unknown-length/chunked response
		_, _ = io.WriteString(w, payload[8:])
	}))
	defer server.Close()

	created := make(chan createdObjectURL, 1)
	engine, cleanup := newStreamTestEngine(t, created)
	defer func() {
		cleanup()
		engine.Close()
	}()

	value, err := engine.RunString(fmt.Sprintf(`
		(async () => {
			const response = await fetch(%q);
			const blob = await response.blob();
			return __gopeed_blob_create_object_url(blob);
		})()
	`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if value != "blob:test-1" {
		t.Fatalf("unexpected object URL: %#v", value)
	}

	object := <-created
	if object.opts.Size != int64(len(payload)) || object.opts.ContentType != "text/plain" {
		t.Fatalf("unexpected object URL options: %#v", object.opts)
	}
	if !object.opts.Range {
		t.Fatal("materialized response Blob should advertise range support")
	}
	for i := 0; i < 2; i++ {
		reader, err := object.open(context.Background(), stream.ObjectURLOpenRequest{Offset: 0, End: -1})
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if got := string(data); got != payload {
			t.Fatalf("unexpected response Blob data on open %d: %q", i+1, got)
		}
	}
	rangeReader, err := object.open(context.Background(), stream.ObjectURLOpenRequest{Offset: 9, End: 12})
	if err != nil {
		t.Fatal(err)
	}
	rangeData, readErr := io.ReadAll(rangeReader)
	_ = rangeReader.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got, want := string(rangeData), payload[9:13]; got != want {
		t.Fatalf("unexpected materialized Blob range: got %q want %q", got, want)
	}
}

func TestResponseBlobRejectsTruncatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "64")
		_, _ = io.WriteString(w, "partial")
	}))
	defer server.Close()

	created := make(chan createdObjectURL, 1)
	engine, cleanup := newStreamTestEngine(t, created)
	defer func() {
		cleanup()
		engine.Close()
	}()

	value, err := engine.RunString(fmt.Sprintf(`
		(async () => {
			const response = await fetch(%q);
			try {
				await response.blob();
				return "resolved";
			} catch (_) {
				return "rejected";
			}
		})()
	`, server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if value != "rejected" {
		t.Fatalf("truncated response unexpectedly materialized a Blob: %#v", value)
	}
}

func TestEmptyBlobObjectURL(t *testing.T) {
	created := make(chan createdObjectURL, 1)
	engine, cleanup := newStreamTestEngine(t, created)
	defer func() {
		cleanup()
		engine.Close()
	}()

	value, err := engine.RunString(`
		__gopeed_blob_create_object_url(new Blob([], { type: "application/empty" }))
	`)
	if err != nil {
		t.Fatal(err)
	}
	if value != "blob:test-1" {
		t.Fatalf("unexpected object URL: %#v", value)
	}

	object := <-created
	if object.opts.Size != 0 || object.opts.Range {
		t.Fatalf("unexpected empty Blob options: %#v", object.opts)
	}
	reader, err := object.open(context.Background(), stream.ObjectURLOpenRequest{Offset: 0, End: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty Blob, got %q", data)
	}
}

func TestBlobObjectURLReaderCloseCancelsPendingJSReader(t *testing.T) {
	created := make(chan createdObjectURL, 1)
	engine, cleanup := newStreamTestEngine(t, created)
	defer func() {
		cleanup()
		engine.Close()
	}()

	value, err := engine.RunString(`
		globalThis.__blobPipeCancelState = {
			readCalls: 0,
			cancelCalls: 0,
			releaseCalls: 0,
			cancelReason: "",
			resolveRead: null,
		};
		const pendingSource = {
			getReader() {
				return {
					read() {
						__blobPipeCancelState.readCalls++;
						return new Promise((resolve) => {
							__blobPipeCancelState.resolveRead = resolve;
						});
					},
					cancel(reason) {
						__blobPipeCancelState.cancelCalls++;
						__blobPipeCancelState.cancelReason = String(reason);
						if (__blobPipeCancelState.resolveRead) {
							__blobPipeCancelState.resolveRead({ done: true, value: undefined });
						}
						return Promise.resolve();
					},
					releaseLock() {
						__blobPipeCancelState.releaseCalls++;
					},
				};
			},
		};
		__gopeed_blob_create_object_url(async () => pendingSource);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if value != "blob:test-1" {
		t.Fatalf("unexpected object URL: %#v", value)
	}

	object := <-created
	reader, err := object.open(context.Background(), stream.ObjectURLOpenRequest{Offset: 0, End: -1})
	if err != nil {
		t.Fatal(err)
	}
	readDone := make(chan error, 1)
	go func() {
		_, readErr := reader.Read(make([]byte, 1))
		readDone <- readErr
	}()

	waitForJSValue(t, engine, `String(__blobPipeCancelState.readCalls)`, "1")
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case readErr := <-readDone:
		if !errors.Is(readErr, context.Canceled) {
			t.Fatalf("unexpected pending read result: %v", readErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending Go read was not released by Close")
	}
	waitForJSValue(t, engine, `JSON.stringify({
		cancelCalls: __blobPipeCancelState.cancelCalls,
		releaseCalls: __blobPipeCancelState.releaseCalls,
		cancelReason: __blobPipeCancelState.cancelReason,
	})`, `{"cancelCalls":1,"releaseCalls":1,"cancelReason":"blob request closed"}`)
}

func waitForJSValue(t *testing.T, engine *engine.Engine, expression, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var got any
	for time.Now().Before(deadline) {
		var err error
		got, err = engine.RunString(expression)
		if err != nil {
			t.Fatal(err)
		}
		if got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for JavaScript value %q, got %#v", want, got)
}

func newStreamTestEngine(t *testing.T, created chan<- createdObjectURL) (*engine.Engine, func()) {
	t.Helper()
	var next atomic.Int64
	e := engine.NewEngine(&engine.Config{
		StreamConfig: &stream.Config{
			CreateObjectURL: func(opts *stream.ObjectURLOptions, open stream.ObjectURLOpener) (string, error) {
				created <- createdObjectURL{opts: opts, open: open}
				return fmt.Sprintf("blob:test-%d", next.Add(1)), nil
			},
			RevokeObjectURL: func(string) error { return nil },
		},
	})
	return e, e.Close
}

// A stalled disk/network consumer must pause the producer without blocking
// the JS event loop, so cancellation and other tracks can still run.
func TestBlobBackpressureLeavesEventLoopResponsive(t *testing.T) {
	created := make(chan createdObjectURL, 1)
	e, cleanup := newStreamTestEngine(t, created)
	defer cleanup()
	_, err := e.RunString(`
 globalThis.produced = 0;
 globalThis.sourceCancelled = false;
 __gopeed_blob_create_object_url(() => new ReadableStream({
   pull(c) { produced++; c.enqueue(new Uint8Array(64*1024)); },
   cancel() { sourceCancelled = true; }
 }));
 `)
	if err != nil {
		t.Fatal(err)
	}
	object := <-created
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reader, err := object.open(ctx, stream.ObjectURLOpenRequest{End: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	time.Sleep(100 * time.Millisecond)
	value, err := e.RunString("produced")
	if err != nil {
		t.Fatal(err)
	}
	before := value.(int64)
	if before > 10 {
		t.Fatalf("unbounded production: %d", before)
	}
	time.Sleep(50 * time.Millisecond)
	after, err := e.RunString("produced")
	if err != nil || after != value {
		t.Fatalf("producer did not pause: %v -> %v (%v)", value, after, err)
	}
	if ctx.Err() != nil {
		t.Fatal("event loop only resumed after cancellation")
	}
	if _, err = reader.Read(make([]byte, 64*1024)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	after, err = e.RunString("produced")
	if err != nil || after.(int64) <= before {
		t.Fatalf("producer did not resume: %v (%v)", after, err)
	}
	reader.Close()
	waitForJSValue(t, e, "String(sourceCancelled)", "true")
}
