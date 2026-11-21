//go:build darwin || linux

package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type resourceSnapshot struct {
	HeapAlloc   uint64 `json:"heap_alloc_bytes"`
	HeapInuse   uint64 `json:"heap_inuse_bytes"`
	HeapObjects uint64 `json:"heap_objects"`
	RSS         int64  `json:"rss_bytes"`
	PeakRSS     int64  `json:"process_peak_rss_bytes"`
	Goroutines  int    `json:"goroutines"`
}

type resourceResult struct {
	Engine         string           `json:"engine"`
	Extension      string           `json:"extension"`
	Mode           string           `json:"mode"`
	Count          int              `json:"count"`
	UserNS         int64            `json:"user_cpu_ns"`
	SystemNS       int64            `json:"system_cpu_ns"`
	WallNS         int64            `json:"wall_ns"`
	Allocated      uint64           `json:"allocated_bytes"`
	GCs            uint32           `json:"gc_cycles"`
	GCPause        uint64           `json:"gc_pause_ns"`
	OutputHash     string           `json:"output_sha256"`
	Baseline       resourceSnapshot `json:"baseline"`
	AfterWork      resourceSnapshot `json:"after_work"`
	AfterGC        resourceSnapshot `json:"after_gc"`
	AfterOSRelease resourceSnapshot `json:"after_os_release"`
	AfterClose     resourceSnapshot `json:"after_close_os_release"`
}

// moejs releases weak-keyed compiled metadata via asynchronous Go cleanups.
// Multiple collections with scheduling opportunities avoid counting objects
// pending those cleanups as permanently live. This is outside CPU timing.
func settledGC(t *testing.T) {
	t.Helper()
	var previous uint64
	for i := 0; i < 8; i++ {
		runtime.GC()
		time.Sleep(20 * time.Millisecond)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		delta := m.HeapAlloc
		if previous > m.HeapAlloc {
			delta = previous - m.HeapAlloc
		} else {
			delta = m.HeapAlloc - previous
		}
		if i >= 2 && delta < 64*1024 {
			return
		}
		previous = m.HeapAlloc
	}
	t.Fatal("live heap did not settle within eight GC rounds")
}

func selfUsage(t *testing.T) syscall.Rusage {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	return usage
}

func timevalNS(v syscall.Timeval) int64 { return int64(v.Sec)*1e9 + int64(v.Usec)*1e3 }

func snapshot(t *testing.T) resourceSnapshot {
	t.Helper()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	// ps runs only outside the timed CPU interval. RUSAGE_SELF excludes its CPU.
	b, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Fatal(err)
	}
	rss, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	peak := int64(selfUsage(t).Maxrss)
	if runtime.GOOS == "linux" {
		peak *= 1024
	}
	return resourceSnapshot{mem.HeapAlloc, mem.HeapInuse, mem.HeapObjects, rss * 1024, peak, runtime.NumGoroutine()}
}

func resourceRuntime(t *testing.T, engine, name, src, host string) Runtime {
	t.Helper()
	r, err := New(engine)
	if err != nil {
		t.Fatal(err)
	}
	// Clone the bundle for each runtime, like a fresh production file read.
	for _, s := range []string{fmt.Sprintf("var extensionName=%q;", name), host, strings.Clone(src)} {
		if _, err = r.Eval(s); err != nil {
			r.Close()
			t.Fatal(err)
		}
	}
	return r
}

func resolveResource(t *testing.T, r Runtime) string {
	t.Helper()
	value, err := r.Eval("resolveOnce()")
	if err != nil {
		t.Fatal(err)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("unexpected resolve result %T", value)
	}
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// Each invocation is a fresh OS process. The Python driver repeats both engines
// in alternating order; the ordinary test suite skips this measurement worker.
func TestResourceWorker(t *testing.T) {
	if os.Getenv("GOPEED_RESOURCE_WORKER") != "1" {
		t.Skip("process resource measurement worker")
	}
	engine, name, mode := os.Getenv("RESOURCE_ENGINE"), os.Getenv("RESOURCE_EXTENSION"), os.Getenv("RESOURCE_MODE")
	count, err := strconv.Atoi(os.Getenv("RESOURCE_COUNT"))
	if err != nil || count < 1 || count > 10000 {
		t.Fatal("invalid RESOURCE_COUNT")
	}
	src := source(t, name)
	hostBytes, err := os.ReadFile("replay.js")
	if err != nil {
		t.Fatal(err)
	}
	host := string(hostBytes)
	// Pay one-time global initialization only before timing repeat events.
	// Retained-memory runs start without a priming runtime so their memory
	// includes the selected engine's first-use caches and builtins.
	var hash string
	if mode != "retained" {
		prime := resourceRuntime(t, engine, name, src, host)
		hash = resolveResource(t, prime)
		prime.Close()
		prime = nil
	}
	debug.FreeOSMemory()
	result := resourceResult{Engine: engine, Extension: name, Mode: mode, Count: count, Baseline: snapshot(t)}
	var retained []Runtime
	if mode == "warm" {
		r := resourceRuntime(t, engine, name, src, host)
		if h := resolveResource(t, r); h != hash {
			t.Fatal("warm output differs")
		}
		retained = append(retained, r)
		debug.FreeOSMemory()
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cpuBefore := selfUsage(t)
	started := time.Now()
	for i := 0; i < count; i++ {
		var r Runtime
		switch mode {
		case "warm":
			r = retained[0]
		case "cold", "retained":
			r = resourceRuntime(t, engine, name, src, host)
		default:
			t.Fatal("unknown RESOURCE_MODE")
		}
		h := resolveResource(t, r)
		if hash == "" {
			hash = h
		} else if h != hash {
			t.Fatal("resolve output changed")
		}
		if mode == "retained" {
			retained = append(retained, r)
		} else if mode == "cold" {
			r.Close()
		}
	}
	result.WallNS = time.Since(started).Nanoseconds()
	cpuAfter := selfUsage(t)
	runtime.ReadMemStats(&after)
	result.UserNS = timevalNS(cpuAfter.Utime) - timevalNS(cpuBefore.Utime)
	result.SystemNS = timevalNS(cpuAfter.Stime) - timevalNS(cpuBefore.Stime)
	result.Allocated = after.TotalAlloc - before.TotalAlloc
	result.GCs = after.NumGC - before.NumGC
	result.GCPause = after.PauseTotalNs - before.PauseTotalNs
	result.AfterWork = snapshot(t)
	settledGC(t)
	result.AfterGC = snapshot(t)
	debug.FreeOSMemory()
	result.AfterOSRelease = snapshot(t)
	runtime.KeepAlive(retained)
	for _, r := range retained {
		r.Close()
	}
	retained = nil
	settledGC(t)
	debug.FreeOSMemory()
	result.AfterClose = snapshot(t)
	// Keep input strings alive across all snapshots, including after closing
	// runtimes, so memory deltas cannot accidentally subtract the input bundle.
	runtime.KeepAlive(src)
	runtime.KeepAlive(host)
	result.OutputHash = hash
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(os.Getenv("RESOURCE_OUTPUT"), append(output, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
