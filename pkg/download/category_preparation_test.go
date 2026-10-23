package download

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
)

type failNextTaskPutStorage struct {
	Storage
	fail          atomic.Bool
	failBucket    string
	beforeTaskPut func(*Task)
}

func (s *failNextTaskPutStorage) Put(bucket string, key string, value any) error {
	if bucket == bucketTask && s.beforeTaskPut != nil {
		s.beforeTaskPut(value.(*Task))
	}
	failBucket := s.failBucket
	if failBucket == "" {
		failBucket = bucketTask
	}
	if bucket == failBucket && (bucket != bucketTask || !value.(*Task).NeedsPreparation) && s.fail.CompareAndSwap(true, false) {
		return errors.New("task snapshot write failed")
	}
	return s.Storage.Put(bucket, key, value)
}

func TestDownloader_PreparationSaveFailureCanRetry(t *testing.T) {
	for _, failBucket := range []string{bucketSave, bucketTask} {
		for _, restart := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/restart=%t", failBucket, restart), func(t *testing.T) {
				url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
				downloadDir := t.TempDir()
				storageDir := t.TempDir()
				storage := &failNextTaskPutStorage{Storage: NewBoltStorage(storageDir), failBucket: failBucket}
				downloader := NewDownloader(&DownloaderConfig{Storage: storage, StorageDir: storageDir})
				if err := downloader.Setup(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { downloader.Close() })
				categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
				if err := os.MkdirAll(categoryDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(categoryDir, "setup.data"), []byte("existing"), 0644); err != nil {
					t.Fatal(err)
				}
				failed := watchTaskError(t, downloader)
				storage.fail.Store(true)
				id, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{Path: downloadDir})
				if err != nil {
					t.Fatal(err)
				}
				waitTaskError(t, failed)
				task := downloader.GetTask(id)
				if !task.NeedsPreparation || task.Meta.Opts.Path != downloadDir || task.Meta.Opts.Name != "" {
					t.Fatal("failed preparation changed the original directory, name or flag")
				}
				if restart {
					if err := downloader.Close(); err != nil {
						t.Fatal(err)
					}
					downloader = NewDownloader(&DownloaderConfig{Storage: NewBoltStorage(storageDir), StorageDir: storageDir})
					if err := downloader.Setup(); err != nil {
						t.Fatal(err)
					}
				}
				done := watchTaskDone(t, downloader)
				if err := downloader.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
					t.Fatal(err)
				}
				waitTaskDone(t, done)
				task = downloader.GetTask(id)
				if task.NeedsPreparation || task.Meta.Opts.Path != categoryDir || task.Meta.Opts.Name != "setup (1).data" {
					t.Fatalf("retry did not prepare the final directory and duplicate name: %+v", task.Meta.Opts)
				}
				data, err := os.ReadFile(filepath.Join(categoryDir, "setup (1).data"))
				if err != nil || string(data) != "0123456789" {
					t.Fatalf("downloaded content = %q, %v", data, err)
				}
			})
		}
	}
}

// watchTaskError subscribes to the error event, so a failing task can be
// awaited instead of polled.
func watchTaskError(t *testing.T, d *Downloader) <-chan struct{} {
	t.Helper()
	failed := make(chan struct{}, 1)
	d.Listener(func(event *Event) {
		if event.Key == EventKeyError {
			select {
			case failed <- struct{}{}:
			default:
			}
		}
	})
	return failed
}

func waitTaskError(t *testing.T, failed <-chan struct{}) {
	t.Helper()
	select {
	case <-failed:
	case <-time.After(30 * time.Second):
		t.Fatal("task did not fail")
	}
}

// waitTaskStatus waits until the task reaches one of the given statuses.
func waitTaskStatus(t *testing.T, d *Downloader, task *Task, statuses ...base.Status) base.Status {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status := d.taskStatus(task)
		for _, want := range statuses {
			if status == want {
				return status
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task status = %q, want one of %v", d.taskStatus(task), statuses)
	return ""
}

// setMaxRunning limits the number of tasks that run at the same time.
func setMaxRunning(t *testing.T, d *Downloader, max int) {
	t.Helper()
	cfg, err := d.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxRunning = max
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

// A task that waits in the queue is prepared when it starts, the first
// preparation must not be skipped by the wait status.
func TestDownloader_QueuedTaskIsPreparedWhenItStarts(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.Header().Set("Content-Disposition", `attachment; filename="slow.bin"`)
		w.Write([]byte("slow"))
	}))
	defer slow.Close()
	fast := startCategoryFileServer(t, "setup.data", []byte("0123456789"))

	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	setMaxRunning(t, downloader, 1)

	// The blocker takes the only running slot.
	blockerID, err := downloader.CreateDirect(&base.Request{URL: slow.URL}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	blocker := downloader.GetTask(blockerID)
	if blocker == nil {
		t.Fatal("blocker task not found")
	}
	waitTaskStatus(t, downloader, blocker, base.DownloadStatusRunning)

	queuedID, err := downloader.CreateDirect(&base.Request{URL: fast}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	queued := downloader.GetTask(queuedID)
	if queued == nil {
		t.Fatal("task not found")
	}
	if queued.Status != base.DownloadStatusWait {
		t.Fatalf("queued task status = %q, want wait", queued.Status)
	}

	close(release)
	waitTaskStatus(t, downloader, queued, base.DownloadStatusDone, base.DownloadStatusError)

	if queued.Meta.Opts.Path != categoryDir {
		t.Fatalf("queued task path = %q, want the category dir %q", queued.Meta.Opts.Path, categoryDir)
	}
	if _, err := os.Stat(filepath.Join(categoryDir, "setup.data")); err != nil {
		t.Fatalf("categorized file missing: %v", err)
	}
}

// A task that failed before it was prepared is prepared when it continues.
func TestDownloader_FailedTaskIsPreparedWhenItContinues(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	allowedDir := t.TempDir()

	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	// The white list rejects the base directory, so the first start fails.
	downloader.cfg.WhiteDownloadDirs = []string{allowedDir}

	failed := watchTaskError(t, downloader)
	taskID, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskError(t, failed)
	waitTaskStatus(t, downloader, task, base.DownloadStatusError)
	if !task.NeedsPreparation {
		t.Fatal("a failed preparation must keep the needs preparation flag")
	}

	// The white list accepts the base directory now, the retry prepares the
	// task again.
	downloader.cfg.WhiteDownloadDirs = []string{allowedDir, downloadDir, categoryDir}
	done := watchTaskDone(t, downloader)
	if err := downloader.Continue(&TaskFilter{IDs: []string{taskID}}); err != nil {
		t.Fatal(err)
	}
	waitTaskDone(t, done)

	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("retried task path = %q, want the category dir %q", task.Meta.Opts.Path, categoryDir)
	}
	if task.NeedsPreparation {
		t.Fatal("a prepared task must clear the needs preparation flag")
	}
}

// A failed preparation save leaves the live task untouched, so the next
// attempt starts from the original base directory and name.
func TestDownloader_PreparationSaveFailureKeepsOriginalData(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	storage := &failNextTaskPutStorage{Storage: NewMemStorage()}
	downloader := NewDownloader(&DownloaderConfig{Storage: storage})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	})
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	setMaxRunning(t, downloader, 0)

	result, err := downloader.Resolve(&base.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := downloader.CreateWithOptions(result.ID, &base.Options{
		Path: downloadDir,
		Name: "renamed?.data",
	})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	originalMeta := util.DeepClone(task.Meta)
	originalRequest := task.Meta.Req
	storage.beforeTaskPut = func(snapshot *Task) {
		if !task.NeedsPreparation || !reflect.DeepEqual(task.Meta, originalMeta) {
			t.Error("preparation was published before the snapshot was saved")
		}
		if snapshot.NeedsPreparation || snapshot.Meta.Opts.Path != categoryDir {
			t.Error("the snapshot must contain the prepared directory and cleared flag")
		}
	}

	storage.fail.Store(true)
	if err := downloader.prepareTask(task); err == nil {
		t.Fatal("prepareTask() succeeded, want a storage error")
	}
	if !task.NeedsPreparation {
		t.Fatal("a failed save must keep the needs preparation flag")
	}
	if !reflect.DeepEqual(task.Meta, originalMeta) {
		t.Fatalf("task metadata changed after a failed save:\n got: %#v\nwant: %#v", task.Meta, originalMeta)
	}
	if !reflect.DeepEqual(task.fetcher.Meta(), originalMeta) {
		t.Fatalf("fetcher metadata changed after a failed save:\n got: %#v\nwant: %#v", task.fetcher.Meta(), originalMeta)
	}

	if err := downloader.prepareTask(task); err != nil {
		t.Fatal(err)
	}
	if task.NeedsPreparation {
		t.Fatal("a successful retry must clear the needs preparation flag")
	}
	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("retried task path = %q, want %q", task.Meta.Opts.Path, categoryDir)
	}
	if task.Meta != task.fetcher.Meta() || task.Meta.Req != originalRequest {
		t.Fatal("preparation must preserve the shared metadata and resolved request")
	}
	storage.beforeTaskPut = nil
	task.Meta.Res.Size++
	var saved Task
	if _, err := storage.Get(bucketTask, taskID, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Meta.Res.Size != originalMeta.Res.Size {
		t.Fatal("live metadata changes must not alter the saved task snapshot")
	}
}

// The white list is checked on every start, a task that was rejected once
// keeps failing instead of writing into the rejected directory.
func TestDownloader_WhiteListIsCheckedOnEveryStart(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	allowedDir := t.TempDir()

	downloader := newSetupCategoryDownloader(t)
	putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	downloader.cfg.WhiteDownloadDirs = []string{allowedDir}

	failed := watchTaskError(t, downloader)
	taskID, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskError(t, failed)
	waitTaskStatus(t, downloader, task, base.DownloadStatusError)

	// The task is continued, the white list still rejects the directory.
	failed = watchTaskError(t, downloader)
	if err := downloader.Continue(&TaskFilter{IDs: []string{taskID}}); err != nil {
		t.Fatal(err)
	}
	waitTaskError(t, failed)
	waitTaskStatus(t, downloader, task, base.DownloadStatusError)

	if _, err := os.Stat(filepath.Join(downloadDir, "setup.data")); err == nil {
		t.Fatalf("the file was written into the rejected dir %q", downloadDir)
	}
}

// The white list is checked on every start, even for a task that was prepared
// before the white list changed.
func TestDownloader_WhiteListIsCheckedForAPreparedTask(t *testing.T) {
	// The first request is truncated, so the download fails after the task was
	// prepared and the retry can show that the directory is checked again.
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="setup.data"`)
		w.Header().Set("Content-Length", "8")
		w.WriteHeader(http.StatusOK)
		if atomic.AddInt32(&requests, 1) == 1 {
			w.Write([]byte("0123"))
			return
		}
		w.Write([]byte("01234567"))
	}))
	defer server.Close()

	downloadDir := t.TempDir()
	allowedDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	downloader.cfg.WhiteDownloadDirs = []string{allowedDir, downloadDir, categoryDir}

	failed := watchTaskError(t, downloader)
	taskID, err := downloader.CreateDirect(&base.Request{URL: server.URL}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskError(t, failed)
	waitTaskStatus(t, downloader, task, base.DownloadStatusError)
	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("task path = %q, want the category dir %q", task.Meta.Opts.Path, categoryDir)
	}

	// The white list no longer accepts the prepared directory, the task must
	// not download into it.
	downloader.cfg.WhiteDownloadDirs = []string{allowedDir}
	requestsBeforeRetry := atomic.LoadInt32(&requests)
	failed = watchTaskError(t, downloader)
	if err := downloader.Continue(&TaskFilter{IDs: []string{taskID}}); err != nil {
		t.Fatal(err)
	}
	waitTaskError(t, failed)
	waitTaskStatus(t, downloader, task, base.DownloadStatusError)

	if atomic.LoadInt32(&requests) != requestsBeforeRetry {
		t.Fatalf("the task downloaded into the rejected dir %q", categoryDir)
	}
}

// A prepared task keeps its directory, a later start never classifies it
// again.
func TestDownloader_PreparedTaskKeepsItsDirectory(t *testing.T) {
	// The first request is truncated, so the download fails after the task was
	// prepared and the retry can show that the directory is kept.
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="setup.data"`)
		w.Header().Set("Content-Length", "8")
		w.WriteHeader(http.StatusOK)
		if atomic.AddInt32(&requests, 1) == 1 {
			w.Write([]byte("0123"))
			return
		}
		w.Write([]byte("01234567"))
	}))
	defer server.Close()

	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})

	failed := watchTaskError(t, downloader)
	taskID, err := downloader.CreateDirect(&base.Request{URL: server.URL}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskError(t, failed)
	waitTaskStatus(t, downloader, task, base.DownloadStatusError)
	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("task path = %q, want the category dir %q", task.Meta.Opts.Path, categoryDir)
	}
	if task.NeedsPreparation {
		t.Fatal("a prepared task must clear the needs preparation flag")
	}

	// The category no longer matches the file, the prepared task keeps the
	// directory it was prepared with.
	putCategoryConfig(t, downloader, downloadDir, true, []string{"iso"})
	done := watchTaskDone(t, downloader)
	if err := downloader.Continue(&TaskFilter{IDs: []string{taskID}}); err != nil {
		t.Fatal(err)
	}
	waitTaskDone(t, done)

	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("task path = %q, want the prepared dir %q", task.Meta.Opts.Path, categoryDir)
	}
	if _, err := os.Stat(filepath.Join(categoryDir, "setup.data")); err != nil {
		t.Fatalf("the file was not written into the prepared dir: %v", err)
	}
}

// The global default download directory is only saved when the user named the
// directory explicitly, an auto categorized task never becomes the default.
func TestDownloader_DefaultPathNeedsTheSkipFlag(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	chosenDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	done := watchTaskDone(t, downloader)

	// The skip flag is missing, the task keeps the auto categorize decision
	// and the global default stays untouched.
	if _, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{
		Path:          chosenDir,
		AsDefaultPath: true,
	}); err != nil {
		t.Fatal(err)
	}
	waitTaskDone(t, done)
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DownloadDir == chosenDir {
		t.Fatalf("default download dir = %q, want it unchanged", cfg.DownloadDir)
	}

	// The user named the directory, so it becomes the global default.
	if _, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{
		Path:               chosenDir,
		AsDefaultPath:      true,
		SkipAutoCategorize: true,
	}); err != nil {
		t.Fatal(err)
	}
	waitTaskDone(t, done)
	cfg, err = downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DownloadDir != chosenDir {
		t.Fatalf("default download dir = %q, want %q", cfg.DownloadDir, chosenDir)
	}
}

// Every task of a batch decides its own directory, the shared options of the
// batch do not leak into the single tasks.
func TestDownloader_BatchTasksDecideTheirOwnDirectory(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	chosenDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	done := watchTaskDone(t, downloader)

	taskIDs, err := downloader.CreateDirectBatch(&base.CreateTaskBatch{
		Reqs: []*base.CreateTaskBatchItem{
			{
				Req:  &base.Request{URL: url},
				Opts: &base.Options{Path: chosenDir, SkipAutoCategorize: true},
			},
			{
				Req:  &base.Request{URL: url},
				Opts: &base.Options{Path: downloadDir},
			},
		},
		Opts: &base.Options{Path: downloadDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(taskIDs) != 2 {
		t.Fatalf("task count = %d, want 2", len(taskIDs))
	}
	waitTaskDone(t, done)
	waitTaskDone(t, done)

	skipped := downloader.GetTask(taskIDs[0])
	if skipped == nil {
		t.Fatal("task not found")
	}
	if skipped.Meta.Opts.Path != chosenDir {
		t.Fatalf("skipped task path = %q, want %q", skipped.Meta.Opts.Path, chosenDir)
	}
	categorized := downloader.GetTask(taskIDs[1])
	if categorized == nil {
		t.Fatal("task not found")
	}
	if categorized.Meta.Opts.Path != categoryDir {
		t.Fatalf("categorized task path = %q, want %q", categorized.Meta.Opts.Path, categoryDir)
	}
}
