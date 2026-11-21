package ffmpeg

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental"
)

// Verify the pinned runtime's allocator lifecycle on every development host:
// host-owned buffers must not be released before execution ends, or retained
// after failed instantiation, traps or cancellation.
func TestLinearMemoryModuleLifecycle(t *testing.T) {
	testLinearMemoryModuleLifecycle(t, func(_, _ uint64) experimental.LinearMemory {
		return &testLinearMemory{}
	})
}

type testLinearMemory struct{ buffer []byte }

func (m *testLinearMemory) Reallocate(size uint64) []byte {
	m.buffer = append(m.buffer, make([]byte, int(size)-len(m.buffer))...)
	return m.buffer
}
func (m *testLinearMemory) Free() { m.buffer = nil }

type trackedLinearMemory struct {
	experimental.LinearMemory
	freeCount int
}

func (m *trackedLinearMemory) Free() {
	m.freeCount++
	m.LinearMemory.Free()
}

func testLinearMemoryModuleLifecycle(t *testing.T, allocate experimental.MemoryAllocatorFunc) {
	t.Helper()
	for _, tt := range []struct {
		name string
		// A module with one memory (min=1/max=2 pages), optionally a start
		// function that traps or loops until its context is cancelled.
		wasm    string
		wantErr bool
	}{
		{"close", "0061736d01000000050401010102", false},
		{"trap", "0061736d01000000010401600000030201000504010101020801000a05010300000b", true},
		{"cancel", "0061736d01000000010401600000030201000504010101020801000a0901070003400c000b0b", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var memory *trackedLinearMemory
			ctx, release := withOwnedMemory(ctx, experimental.MemoryAllocatorFunc(
				func(_, max uint64) experimental.LinearMemory {
					memory = &trackedLinearMemory{LinearMemory: allocate(0, max)}
					return memory
				}))
			defer release()
			rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
			defer rt.Close(context.Background())
			binary, err := hex.DecodeString(tt.wasm)
			if err != nil {
				t.Fatal(err)
			}
			module, err := rt.Instantiate(ctx, binary)
			if (err != nil) != tt.wantErr {
				t.Fatalf("instantiate: %v", err)
			}
			if !tt.wantErr {
				if memory == nil || memory.freeCount != 0 {
					t.Fatal("module did not allocate native memory")
				}
				if err := module.Close(ctx); err != nil {
					t.Fatal(err)
				}
			}
			release()
			release()
			if memory == nil || memory.freeCount != 1 {
				t.Fatal("host did not release memory exactly once")
			}
		})
	}
}
