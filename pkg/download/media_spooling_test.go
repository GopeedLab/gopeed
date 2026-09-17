package download

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	media "github.com/GopeedLab/gopeed/pkg/download/engine/ffmpeg"
)

func TestQueuedMergeReportsInputProgress(t *testing.T) {
	for _, mode := range []string{"streams", "http"} {
		t.Run(mode, func(t *testing.T) {
			release1, err := media.Acquire(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer release1()
			release2, err := media.Acquire(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer release2()
			video, err := os.ReadFile("engine/testdata/ffmpeg/video.mp4")
			if err != nil {
				t.Fatal(err)
			}
			audio, err := os.ReadFile("engine/testdata/ffmpeg/audio.mp4")
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b := video
				if r.URL.Path == "/audio" {
					b = audio
				}
				http.ServeContent(w, r, "media", time.Time{}, bytes.NewReader(b))
			}))
			defer server.Close()
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
			e.Runtime.Set("sourceURL", server.URL)
			result, err := e.RunString(`__gopeed_blob_create_object_url(()=>{
   const source=b=>new ReadableStream({start(c){c.enqueue(new Uint8Array(b));c.close();}});
   return __gopeed_ffmpeg.merge(mode==='http'?{video:{url:sourceURL+'/video'},audio:{url:sourceURL+'/audio'}}:
    {inputs:()=>({video:source(videoBytes),audio:source(audioBytes)})});
  },{contentType:'video/mp4'})`)
			if err != nil {
				t.Fatal(err)
			}
			output := t.TempDir()
			id, err := d.CreateDirect(&base.Request{URL: fmt.Sprint(result)}, &base.Options{Path: output, Name: "merged.mp4"})
			if err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(5 * time.Second)
			for {
				task := d.GetTask(id)
				task.lock.Lock()
				task.statusLock.Lock()
				status, downloaded := task.Status, task.Progress.Downloaded
				total := int64(0)
				if task.Meta.Res != nil {
					total = task.Meta.Res.Size
				}
				task.statusLock.Unlock()
				task.lock.Unlock()
				if downloaded == int64(len(video)+len(audio)) {
					if status != base.DownloadStatusRunning || total != 0 {
						t.Fatalf("queued task: status=%v total=%d", status, total)
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("input progress not reported: status=%v bytes=%d", status, downloaded)
				}
				time.Sleep(20 * time.Millisecond)
			}
			task := d.GetTask(id)
			if mode == "streams" {
				time.Sleep(16 * time.Second)
			} else {
				time.Sleep(200 * time.Millisecond)
			}
			task.lock.Lock()
			task.statusLock.Lock()
			speed := task.Progress.Speed
			task.statusLock.Unlock()
			task.lock.Unlock()
			// Once the upstream is drained, waiting for FFmpeg is still a running task.
			if speed < 0 || (mode == "streams" && speed != 0) {
				t.Fatal("negative queued speed", speed)
			}
			release1()
			deadline = time.Now().Add(100 * time.Second)
			for d.taskStatus(task) != base.DownloadStatusDone {
				if time.Now().After(deadline) {
					t.Fatal("merge did not complete")
				}
				time.Sleep(20 * time.Millisecond)
			}
			// Done is published before extension hooks finish. Let the watcher
			// release its Blob/session before tearing down the downloader.
			deadline = time.Now().Add(5 * time.Second)
			for {
				if _, active := d.watchedTasks.Load(id); !active {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("completion hooks did not finish")
				}
				time.Sleep(10 * time.Millisecond)
			}
			body, err := os.ReadFile(filepath.Join(output, "merged.mp4"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(body, []byte("moof")) {
				t.Fatal("invalid merged output")
			}
			task.lock.Lock()
			task.statusLock.Lock()
			downloaded, total := task.Progress.Downloaded, task.Meta.Res.Size
			task.statusLock.Unlock()
			task.lock.Unlock()
			if downloaded != int64(len(body)) || total != int64(len(body)) {
				t.Fatal("completion did not use output size", downloaded, total, len(body))
			}
			files, err := os.ReadDir(d.cfg.TempDir)
			if err != nil || len(files) != 0 {
				t.Fatal("temporary media leaked", files, err)
			}
		})
	}
}
