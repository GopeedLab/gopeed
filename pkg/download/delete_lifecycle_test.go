package download

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

type deleteResolveManager struct {
	*generationTestManager
	returned chan struct{}
}

func (m *deleteResolveManager) Build() fetcher.Fetcher {
	return &deleteResolveFetcher{generationTestFetcher: m.generationTestManager.Build().(*generationTestFetcher), returned: m.returned}
}

type deleteResolveFetcher struct {
	*generationTestFetcher
	returned chan struct{}
}

func (f *deleteResolveFetcher) Resolve(req *base.Request) error {
	defer close(f.returned)
	return f.generationTestFetcher.Resolve(req)
}
func waitDeleteTest(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("operation did not finish")
	}
}
func TestDeleteDuringResolveCannotRestartOrPersist(t *testing.T) {
	for _, force := range []bool{false, true} {
		for _, resolveErr := range []error{nil, errors.New("late resolve error")} {
			t.Run(fmt.Sprintf("force=%v/error=%v", force, resolveErr != nil), func(t *testing.T) {
				release := make(chan struct{})
				var once sync.Once
				defer once.Do(func() { close(release) })
				m := &deleteResolveManager{generationTestManager: &generationTestManager{resolveStarted: make(chan struct{}), resolveRelease: release, resolveErr: resolveErr}, returned: make(chan struct{})}
				d := NewDownloader(&DownloaderConfig{Storage: NewMemStorage(), StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{m}})
				if err := d.Setup(); err != nil {
					t.Fatal(err)
				}
				defer d.Close()
				id, err := d.CreateDirect(&base.Request{URL: "generation://delete"}, &base.Options{Path: t.TempDir(), Name: "delete.bin"})
				if err != nil {
					t.Fatal(err)
				}
				waitDeleteTest(t, m.resolveStarted)
				task := d.GetTask(id)
				if err := d.Delete(&TaskFilter{IDs: []string{id}}, force); err != nil {
					t.Fatal(err)
				}
				once.Do(func() { close(release) })
				waitDeleteTest(t, m.returned)
				task.lock.Lock()
				task.lock.Unlock()
				// A stale error callback can run immediately after the startup handler exits.
				d.doOnError(task, errors.New("late callback"))
				if got := m.starts.Load(); got != 0 {
					t.Errorf("deleted task started %d time(s)", got)
				}
				if d.GetTask(id) != nil {
					t.Error("deleted task is still listed")
				}
				for _, bucket := range []string{bucketTask, bucketSave, bucketTaskStats} {
					var value any
					present, err := d.storage.Get(bucket, id, &value)
					if err != nil {
						t.Fatal(err)
					}
					if present {
						t.Errorf("deleted task resurrected in %s", bucket)
					}
				}
			})
		}
	}
}

type blockedTaskStorage struct {
	Storage
	entered chan struct{}
	release chan struct{}
	blockID string
	once    sync.Once
}

func (s *blockedTaskStorage) Put(bucket, key string, value any) error {
	if bucket == bucketTask && key == s.blockID {
		s.once.Do(func() { close(s.entered); <-s.release })
	}
	return s.Storage.Put(bucket, key, value)
}
func TestDeleteSerializesInFlightTaskCheckpoint(t *testing.T) {
	s := &blockedTaskStorage{Storage: NewMemStorage(), entered: make(chan struct{}), release: make(chan struct{})}
	d := NewDownloader(&DownloaderConfig{Storage: s, StorageDir: t.TempDir(), TempDir: t.TempDir(), FetchManagers: []fetcher.FetcherManager{&generationTestManager{}}})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.cfg.MaxRunning = 0
	id, err := d.CreateDirect(&base.Request{URL: "generation://checkpoint"}, &base.Options{Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	task := d.GetTask(id)
	s.blockID = id
	saved := make(chan struct{})
	go func() { defer close(saved); _ = d.saveTask(task) }()
	waitDeleteTest(t, s.entered)
	deleted := make(chan struct{})
	go func() { defer close(deleted); _ = d.Delete(&TaskFilter{IDs: []string{id}}, false) }()
	deadline := time.Now().Add(3 * time.Second)
	for d.GetTask(id) != nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(s.release)
	waitDeleteTest(t, saved)
	waitDeleteTest(t, deleted)
	var value any
	present, err := s.Get(bucketTask, id, &value)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("in-flight checkpoint restored a deleted task")
	}
}
