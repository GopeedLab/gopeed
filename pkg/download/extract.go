package download

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v4"
)

// supportedArchiveExtensions contains file extensions that can be extracted
var supportedArchiveExtensions = []string{
	".zip",
	".tar",
	".tar.gz", ".tgz",
	".tar.bz2", ".tbz2",
	".tar.xz", ".txz",
	".tar.lz4", ".tlz4",
	".tar.sz", ".tsz",
	".rar",
	".7z",
	".gz",
	".bz2",
	".xz",
	".lz4",
	".sz",
}

// isArchiveFile checks if a file is a supported archive format
func isArchiveFile(filename string) bool {
	lowerName := strings.ToLower(filename)
	for _, ext := range supportedArchiveExtensions {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}
	return false
}

// extractArchive extracts an archive file to a destination directory
func extractArchive(archivePath string, destDir string, password string) error {
	// Open the archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info for format identification
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Identify the archive format
	format, input, err := archiver.Identify(context.Background(), archivePath, file)
	if err != nil {
		return err
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Handle password-protected archives
	if password != "" {
		// Try to set password for formats that support it
		if rar, ok := format.(archiver.Rar); ok {
			rar.Password = password
			format = rar
		}
		if sz, ok := format.(archiver.SevenZip); ok {
			sz.Password = password
			format = sz
		}
		// Note: Zip format in archiver v4 doesn't have a Password field
		// Password-protected zips are not fully supported in this version
	}

	// Handle extraction based on format type
	switch f := format.(type) {
	case archiver.Extractor:
		// For archive formats (zip, rar, 7z, tar, etc.)
		return f.Extract(context.Background(), input, func(ctx context.Context, af archiver.FileInfo) error {
			return extractFile(ctx, af, destDir)
		})
	case archiver.Decompressor:
		// For single-file compression formats (gz, bz2, xz, etc.)
		// Decompress to a file without the compression extension
		baseName := filepath.Base(archivePath)
		for _, ext := range supportedArchiveExtensions {
			if strings.HasSuffix(strings.ToLower(baseName), ext) {
				baseName = baseName[:len(baseName)-len(ext)]
				break
			}
		}
		destPath := filepath.Join(destDir, baseName)

		reader, err := f.OpenReader(input)
		if err != nil {
			return err
		}
		defer reader.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, reader)
		return err
	case archiver.Archiver:
		// This format is an archiver, try to extract using the extractor interface
		if ext, ok := format.(archiver.Extractor); ok {
			// Reset file position
			if seeker, ok := input.(io.Seeker); ok {
				seeker.Seek(0, io.SeekStart)
			}
			return ext.Extract(context.Background(), io.NewSectionReader(file, 0, stat.Size()), func(ctx context.Context, af archiver.FileInfo) error {
				return extractFile(ctx, af, destDir)
			})
		}
	}

	return nil
}

// extractFile handles extracting a single file from an archive
func extractFile(ctx context.Context, af archiver.FileInfo, destDir string) error {
	// Skip directories, they will be created when extracting files
	if af.IsDir() {
		destPath := filepath.Join(destDir, af.NameInArchive)
		return os.MkdirAll(destPath, af.Mode())
	}

	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(af.NameInArchive)
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
	reader, err := af.Open()
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
	return os.Chmod(destPath, af.Mode())
}
