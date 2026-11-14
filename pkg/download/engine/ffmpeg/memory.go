package ffmpeg

import (
	"context"
	"sync"

	"github.com/tetratelabs/wazero/experimental"
)

// The runtime does not always invoke Free when a module's start function traps.
// Keep host ownership until execution returns, including failed instantiation.
func withOwnedMemory(ctx context.Context, allocator experimental.MemoryAllocator) (context.Context, func()) {
	var memories []*ownedLinearMemory
	ctx = experimental.WithMemoryAllocator(ctx, experimental.MemoryAllocatorFunc(
		func(cap, max uint64) experimental.LinearMemory {
			memory := &ownedLinearMemory{LinearMemory: allocator.Allocate(cap, max)}
			memories = append(memories, memory)
			return memory
		}))
	return ctx, func() {
		for _, memory := range memories {
			memory.Free()
		}
	}
}

type ownedLinearMemory struct {
	experimental.LinearMemory
	once sync.Once
}

func (m *ownedLinearMemory) Free() { m.once.Do(m.LinearMemory.Free) }
