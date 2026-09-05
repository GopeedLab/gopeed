//go:build !windows

package http

import (
	"fmt"
	"os"
)

// openPrefetchTempFile creates an anonymous-on-crash prefetch file. POSIX
// keeps the open file description valid after unlink while removing the only
// directory entry, so the kernel reclaims the data even if the process dies.
func openPrefetchTempFile() (*os.File, string, string, error) {
	file, err := os.CreateTemp("", "gopeed-prefetch-*")
	if err != nil {
		return nil, "", "", err
	}

	filePath := file.Name()
	if err := os.Remove(filePath); err != nil {
		_ = file.Close()
		_ = os.Remove(filePath)
		return nil, "", "", fmt.Errorf("unlink prefetch temporary file: %w", err)
	}
	return file, filePath, "", nil
}

func cleanupPrefetchTempArtifacts(filePath, _ string) {
	// Normally the path was unlinked at creation. Retrying removal also covers
	// unusual filesystems where the first unlink raced with external activity.
	if filePath != "" {
		_ = os.Remove(filePath)
	}
}
