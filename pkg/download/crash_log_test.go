package download

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func TestLogPanicPreservesPreviousCrash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crash.log")
	const previous = "previous crash traceback\n"
	if err := os.WriteFile(path, []byte(previous), 0600); err != nil {
		t.Fatal(err)
	}
	logPanic(dir)
	t.Cleanup(func() { _ = debug.SetCrashOutput(nil, debug.CrashOptions{}) })
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != previous {
		t.Fatalf("previous crash was overwritten: %q", data)
	}
}
