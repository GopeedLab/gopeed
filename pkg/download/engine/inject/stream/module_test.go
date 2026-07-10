package stream

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/file"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/formdata"
	"github.com/dop251/goja"
)

func TestExportFetchRequestSnapshotsArrayBuffer(t *testing.T) {
	runtime := goja.New()
	buffer := runtime.NewArrayBuffer([]byte("before"))
	request := runtime.NewObject()
	setFetchRequestDefaults(t, request)
	if err := request.Set("body", buffer); err != nil {
		t.Fatal(err)
	}

	meta, err := exportFetchRequest(runtime, request)
	if err != nil {
		t.Fatal(err)
	}
	buffer.Bytes()[0] = 'X'

	body, ok := meta.Body.([]byte)
	if !ok {
		t.Fatalf("expected byte snapshot, got %T", meta.Body)
	}
	if got := string(body); got != "before" {
		t.Fatalf("ArrayBuffer snapshot changed after export: %q", got)
	}
}

func TestExportFetchRequestSnapshotsFormData(t *testing.T) {
	runtime := goja.New()
	runtime.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	if err := formdata.Enable(runtime); err != nil {
		t.Fatal(err)
	}
	value, err := runtime.RunString(`
		globalThis.form = new FormData();
		form.append("field", "before");
		form;
	`)
	if err != nil {
		t.Fatal(err)
	}
	request := runtime.NewObject()
	setFetchRequestDefaults(t, request)
	if err := request.Set("body", value); err != nil {
		t.Fatal(err)
	}

	meta, err := exportFetchRequest(runtime, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.RunString(`form.set("field", "after")`); err != nil {
		t.Fatal(err)
	}

	body, ok := meta.Body.(*fetchFormDataSnapshot)
	if !ok {
		t.Fatalf("expected FormData snapshot, got %T", meta.Body)
	}
	if len(body.entries) != 1 {
		t.Fatalf("expected one FormData entry, got %d", len(body.entries))
	}
	if got := body.entries[0]; got.name != "field" || got.value != "before" {
		t.Fatalf("unexpected FormData snapshot: %#v", got)
	}
}

func setFetchRequestDefaults(t *testing.T, request *goja.Object) {
	t.Helper()
	for key, value := range map[string]any{
		"url":         "https://example.test",
		"method":      "POST",
		"redirect":    "follow",
		"credentials": "same-origin",
	} {
		if err := request.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSnapshotFetchBodyCopiesFileMetadata(t *testing.T) {
	source := &file.File{
		Reader: strings.NewReader("payload"),
		Name:   "before.txt",
		Size:   7,
	}

	value, err := snapshotFetchBody(source)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, ok := value.(*file.File)
	if !ok {
		t.Fatalf("expected file snapshot, got %T", value)
	}
	source.Name = "after.txt"
	source.Size = 99

	if snapshot.Name != "before.txt" || snapshot.Size != 7 {
		t.Fatalf("file metadata was not snapshotted: %#v", snapshot)
	}
}

func TestFetchRegistryCloseAll(t *testing.T) {
	registry := newFetchRegistry()
	streamCtx, cancel := context.WithCancel(context.Background())
	body := &trackingReadCloser{}
	registry.streams["active"] = &fetchStream{
		body:   body,
		cancel: cancel,
		ctx:    streamCtx,
		ch:     make(chan fetchChunk),
	}

	registry.CloseAll()
	registry.CloseAll()

	select {
	case <-streamCtx.Done():
	default:
		t.Fatal("active fetch context was not canceled")
	}
	if got := body.closeCalls.Load(); got != 1 {
		t.Fatalf("expected body to close once, got %d", got)
	}
	registry.mu.Lock()
	closed := registry.closed
	remaining := len(registry.streams)
	registry.mu.Unlock()
	if !closed || remaining != 0 {
		t.Fatalf("registry not fully closed: closed=%v remaining=%d", closed, remaining)
	}
	if _, err := registry.Open("", nil, &fetchRequest{URL: "https://example.test"}); err == nil {
		t.Fatal("expected a closed registry to reject new fetches")
	}
}

func TestFetchRegistryReadPreservesChunkOrder(t *testing.T) {
	registry := newFetchRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fetchStream{
		body:   io.NopCloser(strings.NewReader("")),
		cancel: cancel,
		ctx:    ctx,
		ch:     make(chan fetchChunk, 2),
	}
	stream.ch <- fetchChunk{data: []byte("abc")}
	stream.ch <- fetchChunk{data: []byte("def")}
	registry.streams["ordered"] = stream
	defer registry.CloseAll()

	var got strings.Builder
	for range 4 {
		chunk, done, err := registry.Read("ordered", 2)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			t.Fatal("stream ended before buffered chunks were read")
		}
		got.Write(chunk)
	}
	if got.String() != "abcdef" {
		t.Fatalf("fetch chunks were reordered: %q", got.String())
	}
}

type trackingReadCloser struct {
	closeCalls atomic.Int32
}

func (r *trackingReadCloser) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (r *trackingReadCloser) Close() error {
	r.closeCalls.Add(1)
	return nil
}
