//go:build windows

package http

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Windows cannot unlink an ordinary open file. Put it in a private directory
// and ask the kernel to delete the data file whenever its last handle closes,
// including process termination. A killed process may leave only an empty,
// negligible directory; normal cleanup removes that directory too.
func openPrefetchTempFile() (*os.File, string, string, error) {
	dirPath, err := os.MkdirTemp("", "gopeed-prefetch-*")
	if err != nil {
		return nil, "", "", err
	}

	filePath := filepath.Join(dirPath, "data")
	file, err := os.OpenFile(
		filePath,
		os.O_RDWR|os.O_CREATE|os.O_EXCL|windows.O_FILE_FLAG_DELETE_ON_CLOSE,
		0o600,
	)
	if err != nil {
		_ = os.Remove(dirPath)
		return nil, "", "", err
	}
	return file, filePath, dirPath, nil
}

func cleanupPrefetchTempArtifacts(filePath, dirPath string) {
	// Close should already have triggered delete-on-close. Explicit removal is
	// harmless and covers filesystems that defer the deletion briefly.
	if filePath != "" {
		_ = os.Remove(filePath)
	}
	if dirPath != "" {
		_ = os.Remove(dirPath)
	}
}
