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
	var cleanup func()
	e := engine.NewEngine(&engine.Config{
		StreamConfig: &stream.Config{
			CreateObjectURL: func(opts *stream.ObjectURLOptions, open stream.ObjectURLOpener) (string, error) {
				created <- createdObjectURL{opts: opts, open: open}
				return fmt.Sprintf("blob:test-%d", next.Add(1)), nil
			},
			RevokeObjectURL: func(string) error { return nil },
			RegisterCleanup: func(fn func()) {
				cleanup = fn
			},
		},
	})
	if cleanup == nil {
		t.Fatal("stream cleanup was not registered")
	}
	return e, cleanup
}
