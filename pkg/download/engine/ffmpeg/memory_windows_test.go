//go:build windows && (amd64 || arm64)

package ffmpeg

import (
	"testing"
	"unsafe"

	"github.com/tetratelabs/wazero/experimental"
	"golang.org/x/sys/windows"
)

func TestWindowsLinearMemoryGrowthAndRelease(t *testing.T) {
	const page = 65536
	m := &windowsLinearMemory{max: 4 * page}
	defer m.Free()
	first := m.Reallocate(page)
	if len(first) != page {
		t.Fatalf("initial allocation: %d", len(first))
	}
	first[0], first[page-1] = 17, 29
	address := m.address
	grown := m.Reallocate(3 * page)
	if unsafe.SliceData(first) != unsafe.SliceData(grown) || grown[0] != 17 || grown[page-1] != 29 {
		t.Fatal("growing memory moved or lost existing bytes")
	}
	for _, b := range grown[page:] {
		if b != 0 {
			t.Fatal("newly committed memory is not zero-filled")
		}
	}
	if m.Reallocate(5*page) != nil || m.address != address || m.size != 3*page {
		t.Fatal("failed growth changed the existing allocation")
	}
	m.Free()
	m.Free()
	if m.address != 0 || m.Reallocate(page) != nil {
		t.Fatal("freed memory can still be used")
	}
	var info windows.MemoryBasicInformation
	if err := windows.VirtualQuery(address, &info, unsafe.Sizeof(info)); err != nil {
		t.Fatal(err)
	}
	const memFree = 0x10000 // MEMORY_BASIC_INFORMATION.State: MEM_FREE.
	if info.State != memFree {
		t.Fatalf("pages remain reserved after Free: state=%x", info.State)
	}
}

func TestWindowsLinearMemoryModuleLifecycle(t *testing.T) {
	testLinearMemoryModuleLifecycle(t, func(_, max uint64) experimental.LinearMemory {
		return &windowsLinearMemory{max: max}
	})
}
