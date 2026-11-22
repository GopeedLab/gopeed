package download

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestFailedDeleteDuringResolveCanContinue(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	m := &deleteResolveManager{generationTestManager: &generationTestManager{
		resolveStarted: make(chan struct{}), resolveRelease: release, holdOpen: true,
	}, returned: make(chan struct{})}
	s := &singleFailureTaskDeleteStorage{Storage: NewMemStorage()}
	d := NewDownloader(&DownloaderConfig{Storage: s, StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{m}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, err := d.CreateDirect(&base.Request{URL: "generation://failed-delete-resolve"}, &base.Options{Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	waitDeleteTest(t, m.resolveStarted)
	task := d.GetTask(id)
	s.fail.Store(true)
	if err := d.Delete(&TaskFilter{IDs: []string{id}}, false); err == nil {
		t.Fatal("expected storage deletion failure")
	}
	once.Do(func() { close(release) })
	waitDeleteTest(t, m.returned)
	task.lock.Lock()
	task.lock.Unlock()
	if m.starts.Load() != 0 {
		t.Fatal("invalidated startup must not reach Start")
	}
	if err := d.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
		t.Fatalf("failed deletion left task in %s, cannot continue: %v", d.taskStatus(task), err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for m.starts.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if m.starts.Load() != 1 {
		t.Fatal("continued task did not start exactly once")
	}
	task.lock.Lock()
	task.lock.Unlock()
}

type deleteRecoveryStorage struct {
	Storage
	bucket string
	fail   atomic.Bool
}

func (s *deleteRecoveryStorage) Delete(bucket, key string) error {
	if bucket == s.bucket && s.fail.Swap(false) {
		return errors.New("injected deletion failure")
	}
	return s.Storage.Delete(bucket, key)
}

// Unlike the generation fixture, this fetcher stops its watcher on Pause and
// cannot be started again after Close, as with a dropped BT torrent.
type deleteRecoveryManager struct {
	generationTestManager
	failClose atomic.Bool
	restored  atomic.Int64
}

func (m *deleteRecoveryManager) Build() fetcher.Fetcher {
	return m.wrap(m.generationTestManager.Build().(*generationTestFetcher))
}

func (m *deleteRecoveryManager) wrap(f *generationTestFetcher) *deleteRecoveryFetcher {
	return &deleteRecoveryFetcher{generationTestFetcher: f, manager: m, waiting: make(chan struct{})}
}

func (m *deleteRecoveryManager) Store(fetcher.Fetcher) (any, error) {
	return &recoveryTestData{SeedBytes: 42}, nil
}

func (m *deleteRecoveryManager) Restore() (any, func(*fetcher.FetcherMeta, any) fetcher.Fetcher) {
	return &recoveryTestData{}, func(meta *fetcher.FetcherMeta, value any) fetcher.Fetcher {
		m.restored.Store(value.(*recoveryTestData).SeedBytes)
		return m.wrap(&generationTestFetcher{manager: &m.generationTestManager, meta: meta})
	}
}

type deleteRecoveryFetcher struct {
	*generationTestFetcher
	manager     *deleteRecoveryManager
	closed      atomic.Bool
	running     atomic.Bool
	waiting     chan struct{}
	once        sync.Once
	waitRelease <-chan struct{}
	waitSuccess bool
}

func (f *deleteRecoveryFetcher) Start() error {
	if f.closed.Load() {
		return errors.New("cannot restart a closed fetcher")
	}
	f.running.Store(true)
	return f.generationTestFetcher.Start()
}

func (f *deleteRecoveryFetcher) Pause() error {
	if f.running.Swap(false) {
		f.done <- errors.New("paused")
	}
	return nil
}

func (f *deleteRecoveryFetcher) Close() error {
	f.closed.Store(true)
	_ = f.Pause()
	if f.manager.failClose.Swap(false) {
		return errors.New("close failed after stopping the fetcher")
	}
	return nil
}

func (f *deleteRecoveryFetcher) Wait() error {
	f.once.Do(func() { close(f.waiting) })
	err := f.generationTestFetcher.Wait()
	if f.waitRelease != nil {
		<-f.waitRelease
		if f.waitSuccess {
			return nil
		}
	}
	return err
}

func TestFailedDeleteRunningTaskCanResumeAndComplete(t *testing.T) {
	for _, stage := range []string{bucketTask, bucketTaskStats, bucketSave, "close", "files", "late-close", "late-close-error"} {
		t.Run(stage, func(t *testing.T) {
			s := &deleteRecoveryStorage{Storage: NewMemStorage(), bucket: stage}
			lateClose := stage == "late-close" || stage == "late-close-error"
			if lateClose {
				s.bucket = bucketSave
			}
			m := &deleteRecoveryManager{generationTestManager: generationTestManager{holdOpen: true}}
			d := NewDownloader(&DownloaderConfig{Storage: s, StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{m}})
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			id, err := d.CreateDirect(&base.Request{URL: "generation://recover-running"}, &base.Options{Path: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			task := d.GetTask(id)
			task.lock.Lock()
			f := task.fetcher.(*deleteRecoveryFetcher)
			task.lock.Unlock()
			waitDeleteTest(t, f.waiting)
			watcher, _ := d.watchedTasks.Load(id)
			var release chan struct{}
			var releaseOnce sync.Once
			if lateClose {
				release = make(chan struct{})
				defer releaseOnce.Do(func() { close(release) })
				f.waitRelease = release
				f.waitSuccess = stage == "late-close"
			}
			var blockedPath string
			if stage == "files" {
				blockedPath = task.Meta.SingleFilepath()
				if err := os.MkdirAll(blockedPath, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(blockedPath, "keep"), []byte("nonempty"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			s.fail.Store(true)
			m.failClose.Store(stage == "close")
			if err := d.Delete(&TaskFilter{IDs: []string{id}}, stage == "files"); err == nil {
				t.Fatal("expected deletion failure")
			}
			if f.running.Load() || d.taskStatus(task) != base.DownloadStatusPause {
				t.Fatal("failed deletion must restore a paused task and stop transmission")
			}
			if !lateClose {
				waitDeleteTest(t, watcher.(*taskWatcher).done)
			}
			var saved Task
			if exists, err := s.Get(bucketTask, id, &saved); err != nil || !exists || saved.Status != base.DownloadStatusPause {
				t.Fatalf("failed deletion did not persist paused recovery: exists=%v err=%v status=%s", exists, err, saved.Status)
			}
			var checkpoint recoveryTestData
			exists, err := s.Get(bucketSave, id, &checkpoint)
			if stage == "files" {
				if err != nil || exists {
					t.Fatalf("file cleanup must not leave a checkpoint for removed bytes: exists=%v err=%v", exists, err)
				}
			} else if err != nil || !exists || checkpoint.SeedBytes != 42 {
				t.Fatalf("failed deletion lost checkpoint: exists=%v err=%v checkpoint=%+v", exists, err, checkpoint)
			}
			if blockedPath != "" {
				if err := os.RemoveAll(blockedPath); err != nil {
					t.Fatal(err)
				}
			}
			if err := d.Continue(&TaskFilter{IDs: []string{id}}); err != nil {
				t.Fatal(err)
			}
			task.lock.Lock()
			resumed := task.fetcher.(*deleteRecoveryFetcher)
			task.lock.Unlock()
			waitDeleteTest(t, resumed.waiting)
			if stage == bucketSave || stage == "close" || stage == "files" || lateClose {
				wantSeed := int64(42)
				if stage == "files" {
					wantSeed = 0
				}
				if resumed == f || m.restored.Load() != wantSeed {
					t.Fatal("closed fetcher was not rebuilt from the retained checkpoint")
				}
			}
			if lateClose {
				releaseOnce.Do(func() { close(release) })
				waitDeleteTest(t, watcher.(*taskWatcher).done)
				if d.taskStatus(task) != base.DownloadStatusRunning {
					t.Fatal("old completion changed the replacement run's status")
				}
				current, exists := d.watchedTasks.Load(id)
				if !exists || current.(*taskWatcher).fetcher != resumed {
					t.Fatal("old completion removed the replacement watcher")
				}
			}
			resumed.running.Store(false)
			resumed.done <- nil
			waitForTaskStatus(t, d, id, base.DownloadStatusDone, 3*time.Second)
			if m.starts.Load() != 2 {
				t.Fatal("resuming should start one new transfer")
			}
		})
	}
}
