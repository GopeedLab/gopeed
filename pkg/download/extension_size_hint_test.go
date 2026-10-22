package download

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestExtensionFileSizeIsAdvisory(t *testing.T) {
	video, err := os.ReadFile("engine/testdata/ffmpeg/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	audio, err := os.ReadFile("engine/testdata/ffmpeg/audio.mp4")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"plain", "merge", "wrapped-merge"} {
		var expected []byte
		for _, size := range []int64{0, 3, 1 << 20} {
			t.Run(fmt.Sprintf("%s/size=%d", mode, size), func(t *testing.T) {
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir(), RefreshInterval: 50})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Clear()
				e, _ := d.newExtensionEngine()
				defer e.Close()
				e.Runtime.Set("videoBytes", e.Runtime.NewArrayBuffer(video))
				e.Runtime.Set("audioBytes", e.Runtime.NewArrayBuffer(audio))
				e.Runtime.Set("mode", mode)
				result, err := e.RunString(`__gopeed_blob_create_object_url(()=>{
  const source=b=>new ReadableStream({start(c){c.enqueue(new Uint8Array(b));c.close();}});
  if (mode==='plain') return source([0,1,2,3,4,5,6,7,8,9]);
  const output=__gopeed_ffmpeg.merge({inputs:()=>({video:source(videoBytes),audio:source(audioBytes)})});
  if (mode==='merge') return output;
  const reader=output.getReader();
  return new ReadableStream({async pull(c){const {done,value}=await reader.read();if(done)c.close();else c.enqueue(value);},cancel(r){return reader.cancel(r);}});
},{contentType:'video/mp4',range:false})`)
				if err != nil {
					t.Fatal(err)
				}
				output := t.TempDir()
				id, err := d.CreateDirect(&base.Request{URL: fmt.Sprint(result)}, &base.Options{Size: size, Path: output, Name: "output.bin"})
				if err != nil {
					t.Fatal(err)
				}
				waitForTaskTerminal(t, d, id, 30*time.Second)
				body, err := os.ReadFile(filepath.Join(output, "output.bin"))
				if err != nil {
					t.Fatal(err)
				}
				if size == 0 {
					expected = body
					if mode == "plain" && !bytes.Equal(body, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}) {
						t.Fatalf("unexpected stream output: %v", body)
					}
					if mode != "plain" && !bytes.Contains(body, []byte("moof")) {
						t.Fatal("invalid merged output")
					}
				} else if !bytes.Equal(body, expected) {
					t.Fatalf("size hint changed output: got %d bytes, want %d", len(body), len(expected))
				}
				task := d.GetTask(id)
				task.lock.Lock()
				defer task.lock.Unlock()
				if task.Meta.Res.Size != int64(len(body)) || task.Meta.Res.Files[0].Size != int64(len(body)) {
					t.Fatalf("final metadata must use actual size: %#v", task.Meta.Res)
				}
			})
		}
	}
}

func TestExtensionFileSizeYieldsToBlobContentLength(t *testing.T) {
	for _, materialized := range []bool{false, true} {
		for _, size := range []int64{3, 1 << 20} {
			t.Run(fmt.Sprintf("materialized=%v/estimate=%d", materialized, size), func(t *testing.T) {
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir()})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Clear()
				e, _ := d.newExtensionEngine()
				defer e.Close()
				e.Runtime.Set("materialized", materialized)
				result, err := e.RunString(`(()=>{
  const payload=new Uint8Array([0,1,2,3,4,5,6,7,8,9]);
  const source=materialized?new Blob([payload]):()=>new ReadableStream({start(c){c.enqueue(payload);c.close();}});
  return __gopeed_blob_create_object_url(source,{size:10});
})()`)
				if err != nil {
					t.Fatal(err)
				}
				output := t.TempDir()
				id, err := d.CreateDirect(&base.Request{URL: fmt.Sprint(result)}, &base.Options{Size: size, Path: output, Name: "output.bin"})
				if err != nil {
					t.Fatal(err)
				}
				waitForTaskTerminal(t, d, id, 5*time.Second)
				body, err := os.ReadFile(filepath.Join(output, "output.bin"))
				if err != nil || !bytes.Equal(body, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}) {
					t.Fatalf("unexpected body=%v err=%v", body, err)
				}
				task := d.GetTask(id)
				if task.Meta.Res.Size != 10 || task.Meta.Res.Files[0].Size != 10 {
					t.Fatalf("Blob Content-Length did not override estimate: %+v", task.Meta.Res)
				}
			})
		}
	}
}

func TestExtensionFileSizeHintWaitsForEOF(t *testing.T) {
	for _, fail := range []bool{false, true} {
		for _, size := range []int64{3, 1 << 20} {
			t.Run(fmt.Sprintf("fail=%v/size=%d", fail, size), func(t *testing.T) {
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir(), RefreshInterval: 50})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Clear()
				e, _ := d.newExtensionEngine()
				defer e.Close()
				e.Runtime.Set("fail", fail)
				result, err := e.RunString(`__gopeed_blob_create_object_url(()=>new ReadableStream({
  start(c){c.enqueue(new Uint8Array([0,1,2,3,4,5,6,7,8,9]));globalThis.finish=()=>{if(fail)c.error(new Error('upstream failed'));else c.close();};}
}),{})`)
				if err != nil {
					t.Fatal(err)
				}
				output := t.TempDir()
				id, err := d.CreateDirect(&base.Request{URL: fmt.Sprint(result)}, &base.Options{Size: size, Path: output, Name: "output.bin"})
				if err != nil {
					t.Fatal(err)
				}
				waitForTaskDownloadedBetween(t, d, id, 10, 10, 5*time.Second)
				task := d.GetTask(id)
				task.lock.Lock()
				if task.Meta.Res.Size != size || d.taskStatus(task) != base.DownloadStatusRunning {
					t.Errorf("hint must remain visible until EOF: size=%d status=%s", task.Meta.Res.Size, d.taskStatus(task))
				}
				task.lock.Unlock()
				stat, err := os.Stat(filepath.Join(output, "output.bin"))
				if err != nil || stat.Size() != 10 {
					t.Fatalf("hint must not preallocate or truncate output: stat=%v err=%v", stat, err)
				}
				if _, err := e.RunString(`finish()`); err != nil {
					t.Fatal(err)
				}
				if fail {
					waitForTaskStatus(t, d, id, base.DownloadStatusError, 5*time.Second)
				} else {
					waitForTaskTerminal(t, d, id, 5*time.Second)
				}
			})
		}
	}
}

func TestExtensionFileSizeHintPauseContinue(t *testing.T) {
	for _, size := range []int64{3, 1 << 20} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir(), RefreshInterval: 50})
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			defer d.Clear()
			e, _ := d.newExtensionEngine()
			defer e.Close()
			result, err := e.RunString(`(()=>{
  let opens=0;
  return __gopeed_blob_create_object_url(()=>new ReadableStream({start(c){
    if (++opens===1) c.enqueue(new Uint8Array([0,1,2,3,4,5,6,7,8,9]));
    else {c.enqueue(new Uint8Array([5,6]));c.close();}
  }}),{});
})()`)
			if err != nil {
				t.Fatal(err)
			}
			output := t.TempDir()
			id, err := d.CreateDirect(&base.Request{URL: fmt.Sprint(result)}, &base.Options{Size: size, Path: output, Name: "output.bin"})
			if err != nil {
				t.Fatal(err)
			}
			waitForTaskDownloadedBetween(t, d, id, 10, 10, 5*time.Second)
			filter := &TaskFilter{IDs: []string{id}}
			if err := d.Pause(filter); err != nil {
				t.Fatal(err)
			}
			if err := d.Continue(filter); err != nil {
				t.Fatal(err)
			}
			waitForTaskTerminal(t, d, id, 5*time.Second)
			body, err := os.ReadFile(filepath.Join(output, "output.bin"))
			if err != nil {
				t.Fatal(err)
			}
			task := d.GetTask(id)
			task.lock.Lock()
			defer task.lock.Unlock()
			if !bytes.Equal(body, []byte{5, 6}) || task.Meta.Res.Size != 2 {
				t.Fatalf("resumed stream kept an estimate or stale bytes: body=%v size=%d", body, task.Meta.Res.Size)
			}
		})
	}
}
