package engine

import (
	"github.com/dop251/goja"
	"testing"
	"time"
)

func TestCloseUnblocksPendingPromise(t *testing.T) {
	e := NewEngine(nil)
	entered := make(chan struct{})
	_, err := e.runOnLoop(func(r *goja.Runtime) (goja.Value, error) {
		return goja.Undefined(), r.Set("entered", func() { close(entered) })
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := e.RunString(`entered(); new Promise(()=>{});`); done <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("script did not enter")
	}
	e.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closed execution returned success")
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Engine.Close left the Promise waiter blocked")
	}
}
