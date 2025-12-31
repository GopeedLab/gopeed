package download

import (
	"context"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
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

// multiPartArchivePatterns contains regex patterns for multi-part archive detection
// Each pattern should have a capture group for the base name and part number
var multiPartArchivePatterns = []*regexp.Regexp{
	// 7z multi-part: file.7z.001, file.7z.002, etc.
	regexp.MustCompile(`(?i)^(.+\.7z)\.(\d{3})$`),
	// RAR multi-part (new style): file.part01.rar, file.part02.rar, etc.
	regexp.MustCompile(`(?i)^(.+)\.part(\d+)\.rar$`),
	// RAR multi-part (old style): file.rar, file.r00, file.r01, etc. (first file is .rar)
	regexp.MustCompile(`(?i)^(.+)\.r(\d{2})$`),
	// ZIP multi-part: file.zip.001, file.zip.002, etc.
	regexp.MustCompile(`(?i)^(.+\.zip)\.(\d{3})$`),
	// ZIP split: file.z01, file.z02, ... file.zip (last file is .zip)
	regexp.MustCompile(`(?i)^(.+)\.z(\d{2})$`),
}

// ArchivePartInfo contains information about a multi-part archive
type ArchivePartInfo struct {
	// IsMultiPart indicates if this file is part of a multi-part archive
	IsMultiPart bool
	// BaseName is the common base name for all parts (without part number extension)
	BaseName string
	// PartNumber is the part number (1-indexed)
	PartNumber int
	// FirstPartPath is the path to the first part of the archive
	FirstPartPath string
	// Pattern indicates which pattern matched (for determining extraction method)
	Pattern string
}

// ExtractProgressCallback is called to report extraction progress
type ExtractProgressCallback func(extractedFiles int, totalFiles int, progress int)

// isArchiveFile checks if a file is a supported archive format
func isArchiveFile(filename string) bool {
	lowerName := strings.ToLower(filename)
	// Check for multi-part archive first
	if isMultiPartArchive(filename) {
		return true
	}
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
func createExtractionHandler(destDir string, totalFiles int, progressCallback ExtractProgressCallback) func(ctx context.Context, fileInfo archives.FileInfo) error {
	var extractedFiles atomic.Int32
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
		return f.Extract(context.Background(), info.input, createExtractionHandler(destDir, totalFiles, progressCallback))
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
			return ext.Extract(context.Background(), io.NewSectionReader(info.file, 0, info.stat.Size()), createExtractionHandler(destDir, totalFiles, progressCallback))
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

// isMultiPartArchive checks if a file is part of a multi-part archive
func isMultiPartArchive(filename string) bool {
	info := getArchivePartInfo(filename)
	return info.IsMultiPart
}

// getArchivePartInfo returns detailed information about a multi-part archive file
func getArchivePartInfo(filename string) ArchivePartInfo {
	baseName := filepath.Base(filename)

	for _, pattern := range multiPartArchivePatterns {
		matches := pattern.FindStringSubmatch(baseName)
		if len(matches) >= 3 {
			partNum := 0
			_, err := parsePartNumber(matches[2], &partNum)
			if err != nil {
				continue
			}

			info := ArchivePartInfo{
				IsMultiPart: true,
				BaseName:    matches[1],
				PartNumber:  partNum,
				Pattern:     pattern.String(),
			}

			// Determine the first part path based on pattern
			dir := filepath.Dir(filename)
			info.FirstPartPath = determineFirstPartPath(dir, info.BaseName, pattern.String())

			return info
		}
	}

	// Check for .rar files that might be the first part of old-style multi-part RAR
	if strings.HasSuffix(strings.ToLower(baseName), ".rar") && !strings.Contains(strings.ToLower(baseName), ".part") {
		// Check if there are .r00, .r01 files in the same directory
		dir := filepath.Dir(filename)
		nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		r00Path := filepath.Join(dir, nameWithoutExt+".r00")
		if _, err := os.Stat(r00Path); err == nil {
			return ArchivePartInfo{
				IsMultiPart:   true,
				BaseName:      nameWithoutExt,
				PartNumber:    1, // .rar is the first part in old-style
				FirstPartPath: filename,
				Pattern:       "rar-old-style",
			}
		}
	}

	return ArchivePartInfo{IsMultiPart: false}
}

// parsePartNumber parses a part number string and stores it in the provided pointer
// Returns the parsed number and nil error on success
func parsePartNumber(s string, partNum *int) (int, error) {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	// For .001, .002 style, part number is the value itself
	// For .part01 style, part number is the value itself
	// For .r00, .r01 style: r00=2 (since .rar is part 1), r01=3, etc.
	// However, for consistency, we return the raw number and handle the offset elsewhere
	*partNum = n
	// Treat 00 as part 0, let callers handle the semantics
	if n == 0 {
		*partNum = 1 // For .001 format, 001 should be 1
	}
	return *partNum, nil
}

// determineFirstPartPath determines the path to the first part of a multi-part archive
func determineFirstPartPath(dir, baseName, pattern string) string {
	switch {
	case strings.Contains(pattern, `.7z)`):
		// 7z multi-part: first part is .7z.001
		return filepath.Join(dir, baseName+".001")
	case strings.Contains(pattern, `.part`):
		// RAR new style: first part is .part01.rar or .part1.rar
		// Try both single and double digit formats
		if _, err := os.Stat(filepath.Join(dir, baseName+".part1.rar")); err == nil {
			return filepath.Join(dir, baseName+".part1.rar")
		}
		return filepath.Join(dir, baseName+".part01.rar")
	case strings.Contains(pattern, `.r(`):
		// RAR old style: first part is .rar (not .r00)
		return filepath.Join(dir, baseName+".rar")
	case strings.Contains(pattern, `.zip)`):
		// ZIP multi-part: first part is .zip.001
		return filepath.Join(dir, baseName+".001")
	case strings.Contains(pattern, `.z(`):
		// ZIP split: last part is .zip, but extraction should start from .z01
		return filepath.Join(dir, baseName+".z01")
	default:
		return ""
	}
}

// isFirstPart checks if the given file is the first part of a multi-part archive
func isFirstPart(filename string) bool {
	info := getArchivePartInfo(filename)
	if !info.IsMultiPart {
		return false
	}

	// For most formats, part 1 is the first part
	// For old-style RAR, check if this is the .rar file
	if info.Pattern == "rar-old-style" {
		return strings.HasSuffix(strings.ToLower(filename), ".rar")
	}

	return info.PartNumber == 1
}

// GetMultiPartArchiveBaseName returns the base name for a multi-part archive
// This is used to group related parts together
func GetMultiPartArchiveBaseName(filename string) string {
	info := getArchivePartInfo(filename)
	if !info.IsMultiPart {
		return ""
	}
	return filepath.Join(filepath.Dir(filename), info.BaseName)
}

// extractMultiPartArchive extracts a multi-part archive starting from the first part
func extractMultiPartArchive(firstPartPath string, destDir string, password string, progressCallback ExtractProgressCallback) error {
	info := getArchivePartInfo(firstPartPath)

	// For 7z multi-part archives, the bodgit/sevenzip library handles multi-volume automatically
	// when using OpenReader with a .001 file
	if strings.Contains(info.Pattern, `\.7z\)`) || strings.HasSuffix(strings.ToLower(firstPartPath), ".7z.001") {
		return extractSevenZipMultiPart(firstPartPath, destDir, password, progressCallback)
	}

	// For RAR multi-part archives, use the archives library with Name and FS fields
	if strings.Contains(info.Pattern, "rar") || info.Pattern == "rar-old-style" {
		return extractRarMultiPart(firstPartPath, destDir, password, progressCallback)
	}

	// For ZIP multi-part archives (.zip.001, .zip.002, etc.), concatenate parts and extract
	if strings.Contains(info.Pattern, `\.zip\)`) || strings.HasSuffix(strings.ToLower(firstPartPath), ".zip.001") {
		return extractZipMultiPart(firstPartPath, destDir, password, progressCallback)
	}

	// For other formats, try standard extraction (may not work for all multi-part formats)
	return extractArchive(firstPartPath, destDir, password, progressCallback)
}
