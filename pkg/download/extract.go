package download

import (
	"context"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/mholt/archives"
)

// supportedArchiveExtensions contains file extensions supported by mholt/archives library
var supportedArchiveExtensions = []string{
	// Archive formats
	".zip",
	".tar",
	".tar.gz", ".tgz",
	".tar.bz2", ".tbz2",
	".tar.xz", ".txz",
	".tar.lz4", ".tlz4",
	".tar.sz", ".tsz",
	".tar.zst", ".tzst",
	".rar",
	".7z",
	// Compression formats
	".gz",
	".bz2",
	".xz",
	".lz4",
	".sz",
	".zst",
	".br",
	".lz",
}

// ExtractProgressCallback is called to report extraction progress
type ExtractProgressCallback func(extractedFiles int, totalFiles int, progress int)

// isArchiveFile checks if a file is a supported archive format
func isArchiveFile(filename string) bool {
	lowerName := strings.ToLower(filename)
	return slices.ContainsFunc(supportedArchiveExtensions, func(ext string) bool {
		return strings.HasSuffix(lowerName, ext)
	})
}

// archiveInfo holds information about an opened archive
type archiveInfo struct {
	file   *os.File
	stat   os.FileInfo
	format archives.Format
	input  io.Reader
}

// openArchive opens an archive file and identifies its format
func openArchive(archivePath string, password string) (*archiveInfo, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	format, input, err := archives.Identify(context.Background(), archivePath, file)
	if err != nil {
		file.Close()
		return nil, err
	}

	// Handle password-protected archives
	if password != "" {
		if rar, ok := format.(archives.Rar); ok {
			rar.Password = password
			format = rar
		}
		if sz, ok := format.(archives.SevenZip); ok {
			sz.Password = password
			format = sz
		}
	}

	return &archiveInfo{
		file:   file,
		stat:   stat,
		format: format,
		input:  input,
	}, nil
}

// createExtractionHandler creates a handler function for extracting files with progress tracking
func createExtractionHandler(destDir string, totalFiles int, extractedFiles *atomic.Int32, progressCallback ExtractProgressCallback) func(ctx context.Context, fileInfo archives.FileInfo) error {
	return func(ctx context.Context, fileInfo archives.FileInfo) error {
		err := extractFile(ctx, fileInfo, destDir)
		if err == nil && !fileInfo.IsDir() {
			extracted := int(extractedFiles.Add(1))
			if progressCallback != nil && totalFiles > 0 {
				progress := int(math.Min(float64((extracted*100)/totalFiles), 100))
				progressCallback(extracted, totalFiles, progress)
			}
		}
		return err
	}
}

// extractArchive extracts an archive file to a destination directory
func extractArchive(archivePath string, destDir string, password string, progressCallback ExtractProgressCallback) error {
	// Open the archive file
	info, err := openArchive(archivePath, password)
	if err != nil {
		return err
	}
	defer info.file.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Handle extraction based on format type
	switch f := info.format.(type) {
	case archives.Extractor:
		// For archive formats (zip, rar, 7z, tar, etc.)
		// First, count total files for progress tracking
		totalFiles, err := countArchiveFiles(archivePath, password)
		if err != nil {
			// If counting fails, proceed without progress reporting
			totalFiles = 0
		}
		var extractedFiles atomic.Int32
		return f.Extract(context.Background(), info.input, createExtractionHandler(destDir, totalFiles, &extractedFiles, progressCallback))
	case archives.Decompressor:
		// For single-file compression formats (gz, bz2, xz, etc.)
		// Decompress to a file without the compression extension
		baseName := filepath.Base(archivePath)
		lowerBaseName := strings.ToLower(baseName)
		for _, ext := range supportedArchiveExtensions {
			if strings.HasSuffix(lowerBaseName, ext) {
				// Get the actual suffix from the original filename (preserving case)
				actualSuffix := baseName[len(baseName)-len(ext):]
				baseName = strings.TrimSuffix(baseName, actualSuffix)
				break
			}
		}
		destPath := filepath.Join(destDir, baseName)

		reader, err := f.OpenReader(info.input)
		if err != nil {
			return err
		}
		defer reader.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		// Report progress at start and end for decompression
		if progressCallback != nil {
			progressCallback(0, 1, 0)
		}
		_, err = io.Copy(destFile, reader)
		if err == nil && progressCallback != nil {
			progressCallback(1, 1, 100)
		}
		return err
	case archives.Archiver:
		// This format is an archiver, try to extract using the extractor interface
		if ext, ok := info.format.(archives.Extractor); ok {
			// Reset file position
			if seeker, ok := info.input.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
			}
			// Count total files for progress tracking
			totalFiles, err := countArchiveFiles(archivePath, password)
			if err != nil {
				totalFiles = 0
			}
			var extractedFiles atomic.Int32
			return ext.Extract(context.Background(), io.NewSectionReader(info.file, 0, info.stat.Size()), createExtractionHandler(destDir, totalFiles, &extractedFiles, progressCallback))
		}
	}

	return nil
}

// createCountingHandler creates a handler function for counting files in an archive
func createCountingHandler(count *int) func(ctx context.Context, fileInfo archives.FileInfo) error {
	return func(ctx context.Context, fileInfo archives.FileInfo) error {
		if !fileInfo.IsDir() {
			*count++
		}
		return nil
	}
}

// countArchiveFiles counts the number of files in an archive for progress calculation
func countArchiveFiles(archivePath string, password string) (int, error) {
	info, err := openArchive(archivePath, password)
	if err != nil {
		return 0, err
	}
	defer info.file.Close()

	count := 0
	switch f := info.format.(type) {
	case archives.Extractor:
		err = f.Extract(context.Background(), info.input, createCountingHandler(&count))
	case archives.Archiver:
		if ext, ok := info.format.(archives.Extractor); ok {
			if seeker, ok := info.input.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
			}
			err = ext.Extract(context.Background(), io.NewSectionReader(info.file, 0, info.stat.Size()), createCountingHandler(&count))
		}
	case archives.Decompressor:
		// Single file compression, count as 1
		return 1, nil
	}

	return count, err
}

// extractFile handles extracting a single file from an archive
func extractFile(ctx context.Context, fileInfo archives.FileInfo, destDir string) error {
	// Skip directories, they will be created when extracting files
	if fileInfo.IsDir() {
		destPath := filepath.Join(destDir, fileInfo.NameInArchive)
		return os.MkdirAll(destPath, fileInfo.Mode())
	}

	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(fileInfo.NameInArchive)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		// Skip files with suspicious paths
		return nil
	}

	destPath := filepath.Join(destDir, cleanPath)

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Open the file from the archive
	reader, err := fileInfo.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents
	_, err = io.Copy(destFile, reader)
	if err != nil {
		return err
	}

	// Set file permissions
	return os.Chmod(destPath, fileInfo.Mode())
}
