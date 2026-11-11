//go:build windows && (amd64 || arm64)

package ffmpeg

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/tetratelabs/wazero/experimental"
	"golang.org/x/sys/windows"
)

// Keep large, short-lived WASM buffers out of the Go heap. Closing a module
// releases its committed pages immediately, without waiting for GC/scavenging.
func withLinearMemory(ctx context.Context) (context.Context, func()) {
	return withOwnedMemory(ctx, experimental.MemoryAllocatorFunc(
		func(_, max uint64) experimental.LinearMemory {
			return &windowsLinearMemory{max: max}
		}))
}

// Reserve the address range once, committing zero-filled pages on demand. This
// avoids both copying the previous buffer and temporarily retaining two large
// Go allocations when memory.grow runs. The backing address never changes.
type windowsLinearMemory struct {
	address uintptr
	max     uint64
	size    uint64
	freed   bool
}

func (m *windowsLinearMemory) Reallocate(size uint64) []byte {
	if m.freed || size > m.max || m.max > uint64(^uint(0)>>1) {
		return nil
	}
	if size == 0 {
		return []byte{}
	}
	if m.address == 0 {
		address, err := windows.VirtualAlloc(0, uintptr(m.max), windows.MEM_RESERVE, windows.PAGE_READWRITE)
		if err != nil {
			panic(fmt.Errorf("ffmpeg: reserve WASM memory: %w", err))
		}
		m.address = address
	}
	if size > m.size {
		_, err := windows.VirtualAlloc(m.address, uintptr(size), windows.MEM_COMMIT, windows.PAGE_READWRITE)
		if err != nil {
			if m.size == 0 {
				// Instantiation cannot proceed without the minimum memory. Wazero
				// has not taken ownership yet, so release the reservation here.
				m.Free()
				panic(fmt.Errorf("ffmpeg: commit WASM memory: %w", err))
			}
			return nil // memory.grow reports failure to the guest.
		}
		m.size = size
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(m.address)), int(size))
}

func (m *windowsLinearMemory) Free() {
	if m.address != 0 {
		_ = windows.VirtualFree(m.address, 0, windows.MEM_RELEASE)
		m.address = 0
	}
	m.size = 0
	m.freed = true
}
