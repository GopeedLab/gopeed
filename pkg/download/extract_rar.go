package download

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mholt/archives"
)

// extractRarMultiPart extracts a multi-part RAR archive
func extractRarMultiPart(firstPartPath string, destDir string, password string, progressCallback ExtractProgressCallback) error {
	// For RAR archives, we need to use the Name field in archives.Rar
	// to let it automatically find subsequent volumes

	dir := filepath.Dir(firstPartPath)
	fileName := filepath.Base(firstPartPath)

	rar := archives.Rar{
		Password: password,
		Name:     fileName,
		FS:       os.DirFS(dir),
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Count files first for progress
	totalFiles := 0
	err := rar.Extract(context.Background(), nil, func(ctx context.Context, fileInfo archives.FileInfo) error {
		if !fileInfo.IsDir() {
			totalFiles++
		}
		return nil
	})

	if err != nil {
		// If counting fails, proceed without progress
		totalFiles = 0
	}

	// Reset and extract with progress tracking
	return rar.Extract(context.Background(), nil, createExtractionHandler(destDir, totalFiles, progressCallback))
}
