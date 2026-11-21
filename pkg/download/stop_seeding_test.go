package download

import (
	"errors"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

type seedingTestFetcher struct {
	fetcher.Fetcher
	closed     int
	closeErr   error
	uploadDone chan struct{}
}

func (f *seedingTestFetcher) Close() error {
	f.closed++
	if f.closeErr != nil {
		return f.closeErr
	}
	if f.closed == 1 {
		close(f.uploadDone)
	}
	return nil
}
func (f *seedingTestFetcher) Progress() fetcher.Progress { return fetcher.Progress{1024} }
func (f *seedingTestFetcher) Stats() *fetcher.Stats      { return &fetcher.Stats{} }
func (f *seedingTestFetcher) Meta() *fetcher.FetcherMeta { return nil }
func (f *seedingTestFetcher) Upload() error              { return nil }
func (f *seedingTestFetcher) UploadedBytes() int64       { return 0 }
func (f *seedingTestFetcher) WaitUpload() error {
	<-f.uploadDone
	return nil
}

type seedingTestManager struct{ fetcher.FetcherManager }

func (*seedingTestManager) Store(fetcher.Fetcher) (any, error) { return nil, nil }

func newSeedingTestDownloader(t *testing.T) (*Downloader, *Task, *seedingTestFetcher) {
	t.Helper()
	d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Clear() })
	f := &seedingTestFetcher{uploadDone: make(chan struct{})}
	task := NewTask()
	task.ID = "seeding"
	task.Protocol = "bt"
	task.Status = base.DownloadStatusDone
	task.Uploading = true
	task.Meta = &fetcher.FetcherMeta{Req: &base.Request{URL: "magnet:?xt=urn:btih:test"}, Opts: &base.Options{Name: "a.bin"}}
	task.Progress = &Progress{Downloaded: 1024}
	initTask(task)
	task.fetcher = f
	task.fetcherManager = &seedingTestManager{}
	d.lock.Lock()
	d.tasks = append(d.tasks, task)
	d.lock.Unlock()
	return d, task, f
}

func TestDownloader_PauseStopsSeedingOfDoneTask(t *testing.T) {
	d, task, f := newSeedingTestDownloader(t)
	if err := d.Pause(&TaskFilter{IDs: []string{task.ID}}); err != nil {
		t.Fatal(err)
	}
	if f.closed != 1 || task.Uploading || d.taskStatus(task) != base.DownloadStatusDone {
		t.Fatalf("seeding not stopped: closed=%d uploading=%v status=%s", f.closed, task.Uploading, task.Status)
	}
	var saved Task
	ok, err := d.storage.Get(bucketTask, task.ID, &saved)
	if err != nil || !ok || saved.Uploading || saved.Status != base.DownloadStatusDone || saved.Progress.Downloaded != 1024 {
		t.Fatalf("stop not persisted: ok=%v err=%v saved=%+v", ok, err, saved)
	}
	if err := d.Pause(&TaskFilter{IDs: []string{task.ID}}); !errors.Is(err, ErrTaskNotFound) || f.closed != 1 {
		t.Fatalf("second pause: err=%v closed=%d", err, f.closed)
	}
}

func TestDownloader_PauseSeedingCloseFailureKeepsUploading(t *testing.T) {
	d, task, f := newSeedingTestDownloader(t)
	f.closeErr = errors.New("close failed")
	if err := d.Pause(&TaskFilter{IDs: []string{task.ID}}); err == nil || !task.Uploading {
		t.Fatalf("failed stop reported as success: err=%v uploading=%v", err, task.Uploading)
	}
}

func TestDownloader_PauseAllKeepsSeeding(t *testing.T) {
	d, task, f := newSeedingTestDownloader(t)
	_ = d.Pause(nil)
	time.Sleep(10 * time.Millisecond)
	if f.closed != 0 || !task.Uploading {
		t.Fatalf("pause all changed seeding: closed=%d uploading=%v", f.closed, task.Uploading)
	}
}
