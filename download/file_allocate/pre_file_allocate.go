package file_allocate

import "os"

type PreFileAllocator struct {
}

func (p *PreFileAllocator) Allocate(name string, size int64) (file *os.File, err error) {
	err = os.Truncate(name, size)
	file, err = os.OpenFile(name, os.O_RDWR, 0644)
	return
}
