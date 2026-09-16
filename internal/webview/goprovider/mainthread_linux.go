//go:build linux && cgo

package goprovider

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
static void pumpGTK(void) {
  // Bound each batch so Go creation/removal requests cannot be starved.
  for (int i = 0; i < 64 && g_main_context_pending(NULL); ++i)
    g_main_context_iteration(NULL, FALSE);
}
*/
import "C"

import (
	"runtime"
	"sync"
	"time"
)

var gtkThreadOnce sync.Once
var gtkThreadTasks = make(chan func())

func RunMainThreadLoop(run func() int) int { return run() }

func postMainThreadTask(task func()) bool {
	gtkThreadOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			// WebKit must always use its original UI thread, including after
			// the last page closes. A page cannot stop this shared event loop.
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case task := <-gtkThreadTasks:
					task()
				case <-ticker.C:
				}
				C.pumpGTK()
			}
		}()
	})
	gtkThreadTasks <- task
	return true
}
