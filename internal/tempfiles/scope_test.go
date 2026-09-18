package tempfiles

import (
	"errors"
	"io"
	"os"
	"sync"
	"testing"
)

func TestSharedDirectoryOwnersAndLateCreation(t *testing.T) {
	root := t.TempDir()
	a, b := &Scope{}, &Scope{}
	own := func(scope *Scope) string {
		t.Helper()
		dir, err := os.MkdirTemp(root, "gopeed-media-")
		if err != nil {
			t.Fatal(err)
		}
		if err := scope.Track(dir); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	first, second := own(a), own(b)
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatal("removed other owner's path", err)
	}
	late, err := os.MkdirTemp(root, "gopeed-media-")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Track(late); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if _, err := os.Stat(late); !os.IsNotExist(err) {
		t.Fatal("late creation leaked", err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentCreationAndShutdown(t *testing.T) {
	root := t.TempDir()
	scope := &Scope{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dir, err := os.MkdirTemp(root, "gopeed-media-")
			if err != nil {
				t.Error(err)
				return
			}
			if err := scope.Track(dir); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				t.Error(err)
			}
		}()
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("shutdown leaked paths", entries, err)
	}
}
