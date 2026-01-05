package download

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/sevenzip"
)

// extractSevenZipMultiPart extracts a multi-part 7z archive using bodgit/sevenzip directly
// The mholt/archives wrapper doesn't properly handle multi-part 7z files because it uses
// io.SectionReader which can only see the first part. The bodgit/sevenzip library's
// OpenReaderWithPassword function handles multi-part files automatically when given the .001 file path.
func extractSevenZipMultiPart(firstPartPath string, destDir string, password string, progressCallback ExtractProgressCallback) error {
	// Use bodgit/sevenzip directly - it automatically handles .001, .002, etc. files
	var reader *sevenzip.ReadCloser
	var err error

	if password != "" {
		reader, err = sevenzip.OpenReaderWithPassword(firstPartPath, password)
	} else {
		reader, err = sevenzip.OpenReader(firstPartPath)
	}
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Count total files for progress
	totalFiles := 0
	for _, f := range reader.File {
		if !f.FileInfo().IsDir() {
			totalFiles++
		}
	}

	// Extract files with progress tracking
	extractedFiles := 0
	for _, f := range reader.File {
		destPath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Extract file - open, copy, close immediately (as recommended by bodgit/sevenzip)
		if err := extractSevenZipFile(f, destPath); err != nil {
			return err
		}

		extractedFiles++
		if progressCallback != nil && totalFiles > 0 {
			progress := int(float64(extractedFiles) / float64(totalFiles) * 100)
			progressCallback(extractedFiles, totalFiles, progress)
		}
	}

	return nil
}

// extractSevenZipFile extracts a single file from a 7z archive
// This follows the bodgit/sevenzip recommended pattern of closing rc before processing the next file
func extractSevenZipFile(f *sevenzip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
