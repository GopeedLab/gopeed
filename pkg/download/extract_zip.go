package download

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
)

// extractZipMultiPart extracts a multi-part ZIP archive (.zip.001, .zip.002, etc.)
// These are created by simply splitting a ZIP file into chunks, so we concatenate them
func extractZipMultiPart(firstPartPath string, destDir string, password string, progressCallback ExtractProgressCallback) error {
	// Find all parts
	parts, err := findZipMultiParts(firstPartPath)
	if err != nil {
		return err
	}

	// Create a multi-part reader that reads across all parts
	multiReader := newMultiPartFileReader(parts)
	defer multiReader.Close()

	// Get total size
	totalSize := multiReader.Size()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// First pass: count files for progress
	totalFiles := 0
	zip := archives.Zip{}
	err = zip.Extract(context.Background(), io.NewSectionReader(multiReader, 0, totalSize), func(ctx context.Context, fileInfo archives.FileInfo) error {
		if !fileInfo.IsDir() {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		// If counting fails, proceed without progress
		totalFiles = 0
	}

	// Reset reader for actual extraction
	multiReader.Close()
	multiReader = newMultiPartFileReader(parts)
	defer multiReader.Close()

	// Second pass: extract with progress tracking
	return zip.Extract(context.Background(), io.NewSectionReader(multiReader, 0, totalSize), createExtractionHandler(destDir, totalFiles, progressCallback))
}

// findZipMultiParts finds all parts of a multi-part ZIP archive in order
func findZipMultiParts(firstPartPath string) ([]string, error) {
	// Extract base name (e.g., "Archive.zip" from "Archive.zip.001")
	dir := filepath.Dir(firstPartPath)
	baseName := filepath.Base(firstPartPath)

	// Remove the .001 suffix to get the base
	if idx := strings.LastIndex(baseName, "."); idx > 0 {
		baseName = baseName[:idx] // "Archive.zip"
	}

	var parts []string
	partNum := 1

	for {
		partPath := filepath.Join(dir, baseName+fmt.Sprintf(".%03d", partNum))
		if _, err := os.Stat(partPath); os.IsNotExist(err) {
			break
		}
		parts = append(parts, partPath)
		partNum++
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("no parts found for %s", firstPartPath)
	}

	return parts, nil
}

// multiPartFileReader provides io.ReaderAt over multiple files concatenated
type multiPartFileReader struct {
	parts     []string
	files     []*os.File
	sizes     []int64
	offsets   []int64 // cumulative offsets for each file
	totalSize int64
}

func newMultiPartFileReader(parts []string) *multiPartFileReader {
	return &multiPartFileReader{parts: parts}
}

func (m *multiPartFileReader) init() error {
	if m.files != nil {
		return nil
	}

	m.files = make([]*os.File, len(m.parts))
	m.sizes = make([]int64, len(m.parts))
	m.offsets = make([]int64, len(m.parts))

	var offset int64
	for i, part := range m.parts {
		f, err := os.Open(part)
		if err != nil {
			m.Close()
			return err
		}
		stat, err := f.Stat()
		if err != nil {
			f.Close()
			m.Close()
			return err
		}
		m.files[i] = f
		m.sizes[i] = stat.Size()
		m.offsets[i] = offset
		offset += stat.Size()
	}
	m.totalSize = offset

	return nil
}

func (m *multiPartFileReader) Size() int64 {
	if err := m.init(); err != nil {
		return 0
	}
	return m.totalSize
}

func (m *multiPartFileReader) ReadAt(p []byte, off int64) (n int, err error) {
	if err := m.init(); err != nil {
		return 0, err
	}

	if off >= m.totalSize {
		return 0, io.EOF
	}

	// Find which file(s) to read from
	for i, fileOffset := range m.offsets {
		fileEnd := fileOffset + m.sizes[i]

		if off >= fileEnd {
			continue
		}

		// Read from this file
		localOffset := off - fileOffset
		toRead := len(p) - n
		if int64(toRead) > fileEnd-off {
			toRead = int(fileEnd - off)
		}

		read, err := m.files[i].ReadAt(p[n:n+toRead], localOffset)
		n += read
		off += int64(read)

		if err != nil && err != io.EOF {
			return n, err
		}

		if n >= len(p) {
			return n, nil
		}
	}

	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (m *multiPartFileReader) Close() error {
	for _, f := range m.files {
		if f != nil {
			f.Close()
		}
	}
	m.files = nil
	return nil
}
