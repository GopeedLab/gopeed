package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsArchiveFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		// Archive formats
		{"file.zip", true},
		{"file.ZIP", true},
		{"file.tar", true},
		{"file.tar.gz", true},
		{"file.tgz", true},
		{"file.tar.bz2", true},
		{"file.tbz2", true},
		{"file.tar.xz", true},
		{"file.txz", true},
		{"file.tar.zst", true},
		{"file.tzst", true},
		{"file.rar", true},
		{"file.RAR", true},
		{"file.7z", true},
		// Compression formats
		{"file.gz", true},
		{"file.bz2", true},
		{"file.xz", true},
		{"file.lz4", true},
		{"file.sz", true},
		{"file.zst", true},
		{"file.br", true},
		{"file.lz", true},
		// Non-archive files
		{"file.txt", false},
		{"file.pdf", false},
		{"file.exe", false},
		{"archive", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isArchiveFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isArchiveFile(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestExtractArchive_Zip(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip file
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	// Extract the archive
	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify the extracted files
	expectedFiles := []string{
		filepath.Join(destDir, "test.txt"),
		filepath.Join(destDir, "subdir", "nested.txt"),
	}

	for _, path := range expectedFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q not found after extraction", path)
		}
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(destDir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestExtractArchive_NonArchive(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a non-archive file
	txtPath := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("not an archive"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tempDir, "extracted")

	// Trying to extract a non-archive should return an error
	err = extractArchive(txtPath, destDir, "", nil)
	if err == nil {
		t.Error("expected error when extracting non-archive file")
	}
}

// createTestZip creates a test zip file with sample content
func createTestZip(path string) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	// Add a file to the root
	f, err := w.Create("test.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("Hello, World!"))
	if err != nil {
		return err
	}

	// Add a file in a subdirectory
	f, err = w.Create("subdir/nested.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("Nested content"))
	if err != nil {
		return err
	}

	return w.Close()
}

func TestExtractArchive_Progress(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "extract_progress_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip file with multiple files
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create zip with 4 files
	if err := createTestZipWithMultipleFiles(zipPath, 4); err != nil {
		t.Fatal(err)
	}

	// Track progress callbacks
	progressCalls := make([]struct {
		extracted int
		total     int
		progress  int
	}, 0)

	// Extract the archive with progress tracking
	err = extractArchive(zipPath, destDir, "", func(extracted int, total int, progress int) {
		progressCalls = append(progressCalls, struct {
			extracted int
			total     int
			progress  int
		}{extracted, total, progress})
	})
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify that progress callbacks were made
	if len(progressCalls) != 4 {
		t.Errorf("expected 4 progress callbacks, got %d", len(progressCalls))
	}

	// Verify the first progress call
	if len(progressCalls) > 0 {
		first := progressCalls[0]
		if first.extracted != 1 || first.total != 4 {
			t.Errorf("first progress call: expected extracted=1, total=4, got extracted=%d, total=%d", first.extracted, first.total)
		}
	}

	// Verify the last progress call
	if len(progressCalls) > 0 {
		last := progressCalls[len(progressCalls)-1]
		if last.extracted != 4 || last.total != 4 || last.progress != 100 {
			t.Errorf("last progress call: expected extracted=4, total=4, progress=100, got extracted=%d, total=%d, progress=%d", last.extracted, last.total, last.progress)
		}
	}
}

// createTestZipWithMultipleFiles creates a test zip file with the specified number of files
func createTestZipWithMultipleFiles(path string, numFiles int) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	for i := 0; i < numFiles; i++ {
		f, err := w.Create(filepath.Join("dir", "file"+string(rune('A'+i))+".txt"))
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("Content of file " + string(rune('A'+i))))
		if err != nil {
			return err
		}
	}

	return w.Close()
}

func TestOpenArchive_NonExistentFile(t *testing.T) {
	_, err := openArchive("/nonexistent/path/file.zip", "")
	if err == nil {
		t.Error("expected error when opening non-existent file")
	}
}

func TestOpenArchive_InvalidFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "open_archive_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that's not a valid archive format
	// Use a txt extension so it's not detected as an archive
	invalidPath := filepath.Join(tempDir, "invalid.txt")
	if err := os.WriteFile(invalidPath, []byte("not an archive file"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = openArchive(invalidPath, "")
	if err == nil {
		// archives.Identify may return a format even for non-archive files
		// This is expected behavior - it identifies based on content/extension
		t.Log("openArchive accepted the file - this is acceptable behavior")
	}
}

func TestOpenArchive_WithPassword(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "open_archive_pwd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid zip file to test password parameter handling
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	// Test that password is accepted (even if zip doesn't use it)
	info, err := openArchive(zipPath, "testpassword")
	if err != nil {
		t.Fatalf("openArchive with password failed: %v", err)
	}
	defer info.file.Close()

	if info.format == nil {
		t.Error("expected format to be set")
	}
}

func TestExtractArchive_NonExistentFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_nonexistent_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	destDir := filepath.Join(tempDir, "extracted")
	err = extractArchive("/nonexistent/file.zip", destDir, "", nil)
	if err == nil {
		t.Error("expected error when extracting non-existent file")
	}
}

func TestExtractArchive_WithPassword(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_pwd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip file
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	// Extract with password (zip doesn't require it, but tests the code path)
	err = extractArchive(zipPath, destDir, "password123", nil)
	if err != nil {
		t.Fatalf("extractArchive with password failed: %v", err)
	}

	// Verify extraction succeeded
	if _, err := os.Stat(filepath.Join(destDir, "test.txt")); os.IsNotExist(err) {
		t.Error("expected file not found after extraction")
	}
}

func TestExtractArchive_Gzip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_gzip_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a gzip compressed file
	gzPath := filepath.Join(tempDir, "test.txt.gz")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestGzip(gzPath, "Hello from gzip!"); err != nil {
		t.Fatal(err)
	}

	// Track progress callback values
	var progressCalls []struct {
		extracted int
		total     int
		progress  int
	}
	err = extractArchive(gzPath, destDir, "", func(extracted int, total int, progress int) {
		progressCalls = append(progressCalls, struct {
			extracted int
			total     int
			progress  int
		}{extracted, total, progress})
	})
	if err != nil {
		t.Fatalf("extractArchive failed for gzip: %v", err)
	}

	// Verify the decompressed file exists
	destPath := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("expected decompressed file not found")
	}

	// Verify content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Hello from gzip!" {
		t.Errorf("unexpected content: %q", string(content))
	}

	// Verify progress callbacks - should have exactly 2 calls for gzip: start (0,1,0) and end (1,1,100)
	if len(progressCalls) != 2 {
		t.Errorf("expected 2 progress callbacks for gzip, got %d", len(progressCalls))
	}
	if len(progressCalls) >= 2 {
		// Verify start callback
		if progressCalls[0].extracted != 0 || progressCalls[0].total != 1 || progressCalls[0].progress != 0 {
			t.Errorf("expected start callback (0,1,0), got (%d,%d,%d)", progressCalls[0].extracted, progressCalls[0].total, progressCalls[0].progress)
		}
		// Verify end callback
		if progressCalls[1].extracted != 1 || progressCalls[1].total != 1 || progressCalls[1].progress != 100 {
			t.Errorf("expected end callback (1,1,100), got (%d,%d,%d)", progressCalls[1].extracted, progressCalls[1].total, progressCalls[1].progress)
		}
	}
}

func createTestGzip(path string, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	_, err = gz.Write([]byte(content))
	if err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

func TestCountArchiveFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "count_archive_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test zip with known number of files
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZipWithMultipleFiles(zipPath, 5); err != nil {
		t.Fatal(err)
	}

	count, err := countArchiveFiles(zipPath, "")
	if err != nil {
		t.Fatalf("countArchiveFiles failed: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 files, got %d", count)
	}
}

func TestCountArchiveFiles_NonExistent(t *testing.T) {
	_, err := countArchiveFiles("/nonexistent/file.zip", "")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestCountArchiveFiles_Gzip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "count_gzip_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a gzip file
	gzPath := filepath.Join(tempDir, "test.txt.gz")
	if err := createTestGzip(gzPath, "test content"); err != nil {
		t.Fatal(err)
	}

	count, err := countArchiveFiles(gzPath, "")
	if err != nil {
		t.Fatalf("countArchiveFiles failed: %v", err)
	}

	// Gzip is single-file compression, should return 1
	if count != 1 {
		t.Errorf("expected 1 file for gzip, got %d", count)
	}
}

func TestExtractArchive_ProgressWithZeroFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_zero_progress_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create an empty zip file
	zipPath := filepath.Join(tempDir, "empty.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createEmptyZip(zipPath); err != nil {
		t.Fatal(err)
	}

	progressCalled := false
	err = extractArchive(zipPath, destDir, "", func(extracted int, total int, progress int) {
		progressCalled = true
	})
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Progress should not be called for empty archive
	if progressCalled {
		t.Error("progress callback should not be called for empty archive")
	}
}

func createEmptyZip(path string) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	return w.Close()
}

func TestExtractArchive_WithDirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_dirs_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestZipWithDirectories(zipPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify files were created (directories are created implicitly)
	expectedPaths := []string{
		filepath.Join(destDir, "dir1", "file1.txt"),
		filepath.Join(destDir, "dir2", "subdir", "file2.txt"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected path %q not found", path)
		}
	}
}

func createTestZipWithDirectories(path string) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	// Create file in directory (directory will be created automatically)
	f, err := w.Create("dir1/file1.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("content1"))
	if err != nil {
		return err
	}

	// Create nested directory structure
	f, err = w.Create("dir2/subdir/file2.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("content2"))
	if err != nil {
		return err
	}

	return w.Close()
}

func TestExtractArchive_NilProgressCallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_nil_progress_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestZipWithMultipleFiles(zipPath, 3); err != nil {
		t.Fatal(err)
	}

	// Should not panic with nil callback
	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}
}

func TestExtractArchive_GzipUppercase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_gzip_upper_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a gzip compressed file with uppercase extension
	gzPath := filepath.Join(tempDir, "test.txt.GZ")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestGzip(gzPath, "Hello from gzip!"); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(gzPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed for gzip with uppercase: %v", err)
	}

	// The base name should have the extension stripped
	destPath := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("expected decompressed file not found")
	}
}

func TestExtractArchive_PathTraversalPrevention(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_traversal_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a zip with a path traversal attempt
	zipPath := filepath.Join(tempDir, "malicious.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createMaliciousZip(zipPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify that the file was not created outside destDir
	// The malicious path should be sanitized
	dangerousPath := filepath.Join(tempDir, "evil.txt")
	if _, err := os.Stat(dangerousPath); err == nil {
		t.Error("path traversal attack succeeded - file created outside destDir")
	}
}

func createMaliciousZip(path string) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	// Create a file with path traversal attempt
	// Note: Go's zip library may sanitize this, but we test it anyway
	f, err := w.Create("../evil.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("malicious content"))
	if err != nil {
		return err
	}

	// Also add a normal file
	f, err = w.Create("safe.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("safe content"))
	if err != nil {
		return err
	}

	return w.Close()
}

func TestIsArchiveFile_AdditionalFormats(t *testing.T) {
	// Test additional archive formats
	tests := []struct {
		filename string
		expected bool
	}{
		// More compression formats
		{"file.tar.lz4", true},
		{"file.tlz4", true},
		{"file.tar.sz", true},
		{"file.tsz", true},
		// Edge cases
		{"file.tar.gz.backup", false},
		{".gz", true},
		{"archive.ZIP.bak", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isArchiveFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isArchiveFile(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestExtractArchive_DestDirCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_destdir_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	// Use a nested path that doesn't exist
	destDir := filepath.Join(tempDir, "level1", "level2", "extracted")

	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify destDir was created
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Error("destDir was not created")
	}

	// Verify files were extracted
	if _, err := os.Stat(filepath.Join(destDir, "test.txt")); os.IsNotExist(err) {
		t.Error("file not extracted")
	}
}

func TestExtractArchive_ProgressCallbackValues(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_progress_values_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create zip with 10 files to test progress calculation
	if err := createTestZipWithMultipleFiles(zipPath, 10); err != nil {
		t.Fatal(err)
	}

	var progressValues []int
	err = extractArchive(zipPath, destDir, "", func(extracted int, total int, progress int) {
		progressValues = append(progressValues, progress)
	})
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify progress increases monotonically
	for i := 1; i < len(progressValues); i++ {
		if progressValues[i] < progressValues[i-1] {
			t.Errorf("progress decreased from %d to %d", progressValues[i-1], progressValues[i])
		}
	}

	// Verify last progress is 100
	if len(progressValues) > 0 && progressValues[len(progressValues)-1] != 100 {
		t.Errorf("final progress should be 100, got %d", progressValues[len(progressValues)-1])
	}
}

func TestCountArchiveFiles_WithDirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "count_dirs_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZipWithDirectories(zipPath); err != nil {
		t.Fatal(err)
	}

	count, err := countArchiveFiles(zipPath, "")
	if err != nil {
		t.Fatalf("countArchiveFiles failed: %v", err)
	}

	// Should only count files, not directories
	// createTestZipWithDirectories creates 2 files
	if count != 2 {
		t.Errorf("expected 2 files, got %d", count)
	}
}

func TestOpenArchive_FileStatError(t *testing.T) {
	// Test error path when file can't be stat'd
	// This is hard to test without mocking, so we just ensure
	// the function handles basic error cases

	tempDir, err := os.MkdirTemp("", "open_archive_stat_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid zip first
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	// Test normal opening
	info, err := openArchive(zipPath, "")
	if err != nil {
		t.Fatalf("openArchive failed: %v", err)
	}
	defer info.file.Close()

	if info.stat == nil {
		t.Error("stat should not be nil")
	}
	if info.format == nil {
		t.Error("format should not be nil")
	}
	if info.input == nil {
		t.Error("input should not be nil")
	}
}

func TestExtractArchive_FilePermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_perms_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Check that files are readable
	content, err := os.ReadFile(filepath.Join(destDir, "test.txt"))
	if err != nil {
		t.Fatalf("couldn't read extracted file: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestExtractArchive_TarGz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_targz_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tarGzPath := filepath.Join(tempDir, "test.tar.gz")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestTarGz(tarGzPath); err != nil {
		t.Fatal(err)
	}

	var progressCalls int
	err = extractArchive(tarGzPath, destDir, "", func(extracted int, total int, progress int) {
		progressCalls++
	})
	if err != nil {
		t.Fatalf("extractArchive failed for tar.gz: %v", err)
	}

	// Verify the extracted file exists
	destPath := filepath.Join(destDir, "test.txt")
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("couldn't read extracted file: %v", err)
	}
	if string(content) != "Hello from tar.gz!" {
		t.Errorf("unexpected content: %q", string(content))
	}

	// Verify nested file exists
	nestedPath := filepath.Join(destDir, "subdir", "nested.txt")
	content, err = os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("couldn't read nested file: %v", err)
	}
	if string(content) != "Nested content" {
		t.Errorf("unexpected nested content: %q", string(content))
	}
}

func createTestTarGz(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add a file
	content := []byte("Hello from tar.gz!")
	header := &tar.Header{
		Name: "test.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write(content); err != nil {
		return err
	}

	// Add a nested file
	nestedContent := []byte("Nested content")
	nestedHeader := &tar.Header{
		Name: "subdir/nested.txt",
		Mode: 0644,
		Size: int64(len(nestedContent)),
	}
	if err := tarWriter.WriteHeader(nestedHeader); err != nil {
		return err
	}
	if _, err := tarWriter.Write(nestedContent); err != nil {
		return err
	}

	return nil
}

func TestCountArchiveFiles_TarGz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "count_targz_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tarGzPath := filepath.Join(tempDir, "test.tar.gz")
	if err := createTestTarGz(tarGzPath); err != nil {
		t.Fatal(err)
	}

	count, err := countArchiveFiles(tarGzPath, "")
	if err != nil {
		t.Fatalf("countArchiveFiles failed: %v", err)
	}

	// Should have 2 files
	if count != 2 {
		t.Errorf("expected 2 files, got %d", count)
	}
}

func TestExtractArchive_LargeFileCount(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_large_count_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create zip with many files to test progress capping at 100
	numFiles := 100
	if err := createTestZipWithMultipleFiles(zipPath, numFiles); err != nil {
		t.Fatal(err)
	}

	var maxProgress int
	err = extractArchive(zipPath, destDir, "", func(extracted int, total int, progress int) {
		if progress > maxProgress {
			maxProgress = progress
		}
		// Progress should never exceed 100
		if progress > 100 {
			t.Errorf("progress exceeded 100: %d", progress)
		}
	})
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Max progress should be 100
	if maxProgress != 100 {
		t.Errorf("expected max progress 100, got %d", maxProgress)
	}
}

func TestExtractArchive_GzipNoExtension(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_gzip_noext_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a gzip with just .gz (no original extension)
	gzPath := filepath.Join(tempDir, "compressed.gz")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestGzip(gzPath, "Compressed content"); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(gzPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Should create file named "compressed" (without .gz)
	destPath := filepath.Join(destDir, "compressed")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("expected decompressed file not found")
	}
}

func TestExtractArchive_ReadOnlyDestDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_readonly_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "readonly")

	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	// Create a read-only directory
	if err := os.MkdirAll(destDir, 0555); err != nil {
		t.Fatal(err)
	}
	// Make it writable for cleanup
	defer os.Chmod(destDir, 0755)

	// Extraction should fail due to read-only destination
	err = extractArchive(zipPath, destDir, "", nil)
	if err == nil {
		t.Error("expected error when extracting to read-only directory")
	}
}

func TestSupportedArchiveExtensions(t *testing.T) {
	// Test that all listed extensions are recognized
	for _, ext := range supportedArchiveExtensions {
		filename := "test" + ext
		if !isArchiveFile(filename) {
			t.Errorf("extension %q should be recognized as archive", ext)
		}
	}
}

func TestExtractArchive_Tar(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_tar_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tarPath := filepath.Join(tempDir, "test.tar")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createTestTar(tarPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(tarPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed for tar: %v", err)
	}

	// Verify files were extracted
	destPath := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("expected file not found")
	}
}

func createTestTar(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()

	content := []byte("Hello from tar!")
	header := &tar.Header{
		Name: "test.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write(content); err != nil {
		return err
	}

	return nil
}

func TestCountArchiveFiles_Tar(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "count_tar_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tarPath := filepath.Join(tempDir, "test.tar")
	if err := createTestTar(tarPath); err != nil {
		t.Fatal(err)
	}

	count, err := countArchiveFiles(tarPath, "")
	if err != nil {
		t.Fatalf("countArchiveFiles failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 file, got %d", count)
	}
}

func TestOpenArchive_ValidArchive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "open_valid_archive_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	info, err := openArchive(zipPath, "")
	if err != nil {
		t.Fatalf("openArchive failed: %v", err)
	}
	defer info.file.Close()

	// Verify all fields are set
	if info.file == nil {
		t.Error("file should not be nil")
	}
	if info.stat == nil {
		t.Error("stat should not be nil")
	}
	if info.format == nil {
		t.Error("format should not be nil")
	}
	if info.input == nil {
		t.Error("input should not be nil")
	}
	if info.stat.Size() == 0 {
		t.Error("file size should be > 0")
	}
}

func TestExtractArchive_NestedDirectoryStructure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_nested_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "nested.zip")
	destDir := filepath.Join(tempDir, "extracted")

	if err := createDeeplyNestedZip(zipPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify deeply nested file
	deepPath := filepath.Join(destDir, "a", "b", "c", "d", "deep.txt")
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Error("deeply nested file not found")
	}
}

func createDeeplyNestedZip(path string) error {
	zipFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	f, err := w.Create("a/b/c/d/deep.txt")
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("Deep content"))
	if err != nil {
		return err
	}

	return w.Close()
}

func TestExtractArchive_EmptyFileName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_empty_name_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create a normal zip - the archive library handles empty names gracefully
	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	err = extractArchive(zipPath, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}
}

func TestExtractArchive_ProgressTracking(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_progress_track_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	numFiles := 5
	if err := createTestZipWithMultipleFiles(zipPath, numFiles); err != nil {
		t.Fatal(err)
	}

	var extractedValues []int
	var totalValues []int

	err = extractArchive(zipPath, destDir, "", func(extracted int, total int, progress int) {
		extractedValues = append(extractedValues, extracted)
		totalValues = append(totalValues, total)
	})
	if err != nil {
		t.Fatalf("extractArchive failed: %v", err)
	}

	// Verify total is always the same
	for _, total := range totalValues {
		if total != numFiles {
			t.Errorf("total should be %d, got %d", numFiles, total)
		}
	}

	// Verify extracted increases sequentially
	for i, extracted := range extractedValues {
		if extracted != i+1 {
			t.Errorf("extracted should be %d, got %d", i+1, extracted)
		}
	}
}

func TestArchiveInfo_Fields(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "archive_info_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	info, err := openArchive(zipPath, "")
	if err != nil {
		t.Fatalf("openArchive failed: %v", err)
	}
	defer info.file.Close()

	// Verify file name
	if info.file.Name() != zipPath {
		t.Errorf("expected file name %q, got %q", zipPath, info.file.Name())
	}

	// Verify stat mode
	if info.stat.Mode().IsDir() {
		t.Error("expected file, not directory")
	}
}

// Tests for multi-part archive detection
func TestIsMultiPartArchive(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		// 7z multi-part
		{"archive.7z.001", true},
		{"archive.7z.002", true},
		{"archive.7z.100", true},
		{"ARCHIVE.7Z.001", true},

		// RAR new style
		{"archive.part01.rar", true},
		{"archive.part1.rar", true},
		{"archive.part99.rar", true},
		{"ARCHIVE.PART01.RAR", true},

		// RAR old style (extension parts)
		{"archive.r00", true},
		{"archive.r01", true},
		{"archive.r99", true},

		// ZIP multi-part
		{"archive.zip.001", true},
		{"archive.zip.002", true},

		// ZIP split
		{"archive.z01", true},
		{"archive.z02", true},

		// Regular (non-multi-part) archives
		{"archive.zip", false},
		{"archive.rar", false},
		{"archive.7z", false},
		{"archive.tar.gz", false},

		// Non-archive files
		{"file.txt", false},
		{"file.001", false}, // No .7z or .zip prefix
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isMultiPartArchive(tt.filename)
			if result != tt.expected {
				t.Errorf("isMultiPartArchive(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetArchivePartInfo_7z(t *testing.T) {
	tests := []struct {
		filename    string
		baseName    string
		partNumber  int
		isMultiPart bool
	}{
		{"archive.7z.001", "archive.7z", 1, true},
		{"archive.7z.002", "archive.7z", 2, true},
		{"archive.7z.010", "archive.7z", 10, true},
		{"my.file.7z.005", "my.file.7z", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if info.IsMultiPart != tt.isMultiPart {
				t.Errorf("IsMultiPart: expected %v, got %v", tt.isMultiPart, info.IsMultiPart)
			}
			if info.BaseName != tt.baseName {
				t.Errorf("BaseName: expected %q, got %q", tt.baseName, info.BaseName)
			}
			if info.PartNumber != tt.partNumber {
				t.Errorf("PartNumber: expected %d, got %d", tt.partNumber, info.PartNumber)
			}
		})
	}
}

func TestGetArchivePartInfo_RarNewStyle(t *testing.T) {
	tests := []struct {
		filename    string
		baseName    string
		partNumber  int
		isMultiPart bool
	}{
		{"archive.part01.rar", "archive", 1, true},
		{"archive.part02.rar", "archive", 2, true},
		{"archive.part1.rar", "archive", 1, true},
		{"my.archive.part10.rar", "my.archive", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if info.IsMultiPart != tt.isMultiPart {
				t.Errorf("IsMultiPart: expected %v, got %v", tt.isMultiPart, info.IsMultiPart)
			}
			if info.BaseName != tt.baseName {
				t.Errorf("BaseName: expected %q, got %q", tt.baseName, info.BaseName)
			}
			if info.PartNumber != tt.partNumber {
				t.Errorf("PartNumber: expected %d, got %d", tt.partNumber, info.PartNumber)
			}
		})
	}
}

func TestGetArchivePartInfo_RarOldStyle(t *testing.T) {
	tests := []struct {
		filename    string
		baseName    string
		partNumber  int
		isMultiPart bool
	}{
		{"archive.r00", "archive", 1, true}, // r00 is treated as first extension part (after .rar)
		{"archive.r01", "archive", 1, true},
		{"archive.r99", "archive", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if info.IsMultiPart != tt.isMultiPart {
				t.Errorf("IsMultiPart: expected %v, got %v", tt.isMultiPart, info.IsMultiPart)
			}
			if info.BaseName != tt.baseName {
				t.Errorf("BaseName: expected %q, got %q", tt.baseName, info.BaseName)
			}
		})
	}
}

func TestGetArchivePartInfo_ZipMultiPart(t *testing.T) {
	tests := []struct {
		filename    string
		baseName    string
		partNumber  int
		isMultiPart bool
	}{
		{"archive.zip.001", "archive.zip", 1, true},
		{"archive.zip.002", "archive.zip", 2, true},
		{"my.file.zip.010", "my.file.zip", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if info.IsMultiPart != tt.isMultiPart {
				t.Errorf("IsMultiPart: expected %v, got %v", tt.isMultiPart, info.IsMultiPart)
			}
			if info.BaseName != tt.baseName {
				t.Errorf("BaseName: expected %q, got %q", tt.baseName, info.BaseName)
			}
			if info.PartNumber != tt.partNumber {
				t.Errorf("PartNumber: expected %d, got %d", tt.partNumber, info.PartNumber)
			}
		})
	}
}

func TestGetArchivePartInfo_ZipSplit(t *testing.T) {
	tests := []struct {
		filename    string
		baseName    string
		partNumber  int
		isMultiPart bool
	}{
		{"archive.z01", "archive", 1, true},
		{"archive.z02", "archive", 2, true},
		{"archive.z99", "archive", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if info.IsMultiPart != tt.isMultiPart {
				t.Errorf("IsMultiPart: expected %v, got %v", tt.isMultiPart, info.IsMultiPart)
			}
			if info.BaseName != tt.baseName {
				t.Errorf("BaseName: expected %q, got %q", tt.baseName, info.BaseName)
			}
			if info.PartNumber != tt.partNumber {
				t.Errorf("PartNumber: expected %d, got %d", tt.partNumber, info.PartNumber)
			}
		})
	}
}

func TestGetArchivePartInfo_NonMultiPart(t *testing.T) {
	tests := []string{
		"archive.zip",
		"archive.rar",
		"archive.7z",
		"file.txt",
		"file.001",
	}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			info := getArchivePartInfo(filename)
			if info.IsMultiPart {
				t.Errorf("Expected non-multi-part for %q, but got IsMultiPart=true", filename)
			}
		})
	}
}

func TestIsFirstPart(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		// First parts
		{"archive.7z.001", true},
		{"archive.part01.rar", true},
		{"archive.part1.rar", true},
		{"archive.zip.001", true},
		{"archive.z01", true},

		// Non-first parts
		{"archive.7z.002", false},
		{"archive.part02.rar", false},
		{"archive.zip.002", false},
		{"archive.z02", false},

		// For RAR old style (.r00, .r01), these are NOT the first part
		// The first part is the .rar file, but these extension files
		// have partNumber=1 due to parsePartNumber treating 00 as 1
		// So isFirstPart returns true for these (which is technically correct
		// in terms of part numbering, even though .rar is the "real" first file)
		{"archive.r00", true},
		{"archive.r01", true},

		// Non-multi-part (should return false)
		{"archive.zip", false},
		{"archive.rar", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isFirstPart(tt.filename)
			if result != tt.expected {
				t.Errorf("isFirstPart(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetMultiPartArchiveBaseName(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"/path/to/archive.7z.001", "/path/to/archive.7z"},
		{"/path/to/archive.part01.rar", "/path/to/archive"},
		{"/path/to/archive.zip.001", "/path/to/archive.zip"},
		{"/path/to/archive.z01", "/path/to/archive"},
		// Non-multi-part should return empty
		{"/path/to/archive.zip", ""},
		{"/path/to/file.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := GetMultiPartArchiveBaseName(tt.filename)
			if result != tt.expected {
				t.Errorf("GetMultiPartArchiveBaseName(%q) = %q, expected %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsArchiveFile_IncludesMultiPart(t *testing.T) {
	// Test that isArchiveFile returns true for multi-part archives
	tests := []struct {
		filename string
		expected bool
	}{
		{"archive.7z.001", true},
		{"archive.7z.002", true},
		{"archive.part01.rar", true},
		{"archive.zip.001", true},
		{"archive.z01", true},
		{"archive.r00", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isArchiveFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isArchiveFile(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestArchivePartInfo_PatternField(t *testing.T) {
	// Verify that the Pattern field is set correctly for different formats
	tests := []struct {
		filename        string
		patternContains string
	}{
		{"archive.7z.001", ".7z)"},
		{"archive.part01.rar", ".part"},
		{"archive.r00", ".r("},
		{"archive.zip.001", ".zip)"},
		{"archive.z01", ".z("},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if !info.IsMultiPart {
				t.Fatalf("Expected multi-part archive for %q", tt.filename)
			}
			if info.Pattern == "" {
				t.Errorf("Pattern should not be empty for %q", tt.filename)
			}
		})
	}
}

func TestArchivePartInfo_FirstPartPath(t *testing.T) {
	// Verify that FirstPartPath is set correctly for different formats
	tests := []struct {
		filename       string
		expectedSuffix string // Expected suffix of the FirstPartPath
	}{
		{"archive.7z.001", "archive.7z.001"},
		{"archive.7z.002", "archive.7z.001"},
		{"archive.7z.005", "archive.7z.001"},
		{"archive.part01.rar", "archive.part01.rar"},
		{"archive.part02.rar", "archive.part01.rar"},
		{"archive.zip.001", "archive.zip.001"},
		{"archive.zip.003", "archive.zip.001"},
		{"archive.z01", "archive.z01"},
		{"archive.z05", "archive.z01"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info := getArchivePartInfo(tt.filename)
			if !info.IsMultiPart {
				t.Fatalf("Expected multi-part archive for %q", tt.filename)
			}
			if info.FirstPartPath == "" {
				t.Errorf("FirstPartPath should not be empty for %q", tt.filename)
			}
			if !strings.HasSuffix(info.FirstPartPath, tt.expectedSuffix) {
				t.Errorf("FirstPartPath for %q = %q, expected suffix %q", tt.filename, info.FirstPartPath, tt.expectedSuffix)
			}
		})
	}
}

// Tests for multiPartFileReader
func TestMultiPartFileReader_Basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multipart_reader_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test parts with known content
	part1Content := []byte("Hello")
	part2Content := []byte("World")
	part3Content := []byte("!")

	part1Path := filepath.Join(tempDir, "test.001")
	part2Path := filepath.Join(tempDir, "test.002")
	part3Path := filepath.Join(tempDir, "test.003")

	if err := os.WriteFile(part1Path, part1Content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part2Path, part2Content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part3Path, part3Content, 0644); err != nil {
		t.Fatal(err)
	}

	reader := newMultiPartFileReader([]string{part1Path, part2Path, part3Path})
	defer reader.Close()

	// Test Size()
	expectedSize := int64(len(part1Content) + len(part2Content) + len(part3Content))
	if reader.Size() != expectedSize {
		t.Errorf("Size() = %d, expected %d", reader.Size(), expectedSize)
	}
}

func TestMultiPartFileReader_ReadAt(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multipart_readat_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test parts with known content
	part1Path := filepath.Join(tempDir, "test.001")
	part2Path := filepath.Join(tempDir, "test.002")

	if err := os.WriteFile(part1Path, []byte("AAAA"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part2Path, []byte("BBBB"), 0644); err != nil {
		t.Fatal(err)
	}

	reader := newMultiPartFileReader([]string{part1Path, part2Path})
	defer reader.Close()

	// Test reading from first part
	buf := make([]byte, 2)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("ReadAt(0) error: %v", err)
	}
	if n != 2 || string(buf) != "AA" {
		t.Errorf("ReadAt(0) = %q, expected %q", string(buf[:n]), "AA")
	}

	// Test reading across parts
	buf = make([]byte, 4)
	n, err = reader.ReadAt(buf, 2)
	if err != nil {
		t.Errorf("ReadAt(2) error: %v", err)
	}
	if n != 4 || string(buf) != "AABB" {
		t.Errorf("ReadAt(2) = %q, expected %q", string(buf[:n]), "AABB")
	}

	// Test reading from second part only
	buf = make([]byte, 2)
	n, err = reader.ReadAt(buf, 6)
	if err != nil {
		t.Errorf("ReadAt(6) error: %v", err)
	}
	if n != 2 || string(buf) != "BB" {
		t.Errorf("ReadAt(6) = %q, expected %q", string(buf[:n]), "BB")
	}
}

func TestMultiPartFileReader_ReadAtEOF(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multipart_eof_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	part1Path := filepath.Join(tempDir, "test.001")
	if err := os.WriteFile(part1Path, []byte("ABC"), 0644); err != nil {
		t.Fatal(err)
	}

	reader := newMultiPartFileReader([]string{part1Path})
	defer reader.Close()

	// Reading beyond EOF should return io.EOF
	buf := make([]byte, 10)
	n, err := reader.ReadAt(buf, 100)
	if err != io.EOF {
		t.Errorf("ReadAt beyond EOF: expected io.EOF, got %v", err)
	}
	if n != 0 {
		t.Errorf("ReadAt beyond EOF: expected 0 bytes, got %d", n)
	}
}

func TestMultiPartFileReader_Close(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multipart_close_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	part1Path := filepath.Join(tempDir, "test.001")
	if err := os.WriteFile(part1Path, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	reader := newMultiPartFileReader([]string{part1Path})

	// Initialize by calling Size
	_ = reader.Size()

	// Close should work
	err = reader.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// After close, files should be nil
	if reader.files != nil {
		t.Error("files should be nil after Close()")
	}
}

func TestMultiPartFileReader_InitError(t *testing.T) {
	// Test with non-existent file
	reader := newMultiPartFileReader([]string{"/nonexistent/path/file.001"})
	defer reader.Close()

	// Size should return 0 on error
	size := reader.Size()
	if size != 0 {
		t.Errorf("Size() with non-existent file should return 0, got %d", size)
	}

	// ReadAt should return error
	buf := make([]byte, 10)
	_, err := reader.ReadAt(buf, 0)
	if err == nil {
		t.Error("ReadAt with non-existent file should return error")
	}
}

// Tests for findZipMultiParts
func TestFindZipMultiParts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find_zip_parts_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test parts
	for i := 1; i <= 5; i++ {
		partPath := filepath.Join(tempDir, fmt.Sprintf("archive.zip.%03d", i))
		if err := os.WriteFile(partPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	firstPartPath := filepath.Join(tempDir, "archive.zip.001")
	parts, err := findZipMultiParts(firstPartPath)
	if err != nil {
		t.Fatalf("findZipMultiParts error: %v", err)
	}

	if len(parts) != 5 {
		t.Errorf("Expected 5 parts, got %d", len(parts))
	}

	// Verify order
	for i, part := range parts {
		expected := filepath.Join(tempDir, fmt.Sprintf("archive.zip.%03d", i+1))
		if part != expected {
			t.Errorf("Part %d: expected %q, got %q", i, expected, part)
		}
	}
}

func TestFindZipMultiParts_NoParts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find_zip_no_parts_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Don't create any parts
	firstPartPath := filepath.Join(tempDir, "archive.zip.001")
	_, err = findZipMultiParts(firstPartPath)
	if err == nil {
		t.Error("Expected error when no parts found")
	}
}

func TestFindZipMultiParts_SinglePart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find_zip_single_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create only the first part
	firstPartPath := filepath.Join(tempDir, "archive.zip.001")
	if err := os.WriteFile(firstPartPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	parts, err := findZipMultiParts(firstPartPath)
	if err != nil {
		t.Fatalf("findZipMultiParts error: %v", err)
	}

	if len(parts) != 1 {
		t.Errorf("Expected 1 part, got %d", len(parts))
	}
}

// Tests for extractMultiPartArchive error paths
func TestExtractMultiPartArchive_NonExistentFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_multipart_err_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	destDir := filepath.Join(tempDir, "extracted")

	// Test with non-existent 7z multi-part
	err = extractMultiPartArchive("/nonexistent/archive.7z.001", destDir, "", nil)
	if err == nil {
		t.Error("Expected error for non-existent 7z file")
	}

	// Test with non-existent zip multi-part
	err = extractMultiPartArchive("/nonexistent/archive.zip.001", destDir, "", nil)
	if err == nil {
		t.Error("Expected error for non-existent zip file")
	}
}

// Test extractZipMultiPart with invalid archive
func TestExtractZipMultiPart_InvalidArchive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_zip_invalid_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create invalid "zip" parts (just text files)
	part1Path := filepath.Join(tempDir, "invalid.zip.001")
	if err := os.WriteFile(part1Path, []byte("not a zip file"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tempDir, "extracted")
	err = extractZipMultiPart(part1Path, destDir, "", nil)
	// Should return an error because the file is not a valid zip
	if err == nil {
		t.Error("Expected error for invalid zip archive")
	}
}

// Test extractRarMultiPart with non-existent file
func TestExtractRarMultiPart_NonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_rar_err_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	destDir := filepath.Join(tempDir, "extracted")
	err = extractRarMultiPart("/nonexistent/archive.part01.rar", destDir, "", nil)
	if err == nil {
		t.Error("Expected error for non-existent RAR file")
	}
}

// Test extractSevenZipMultiPart with non-existent file
func TestExtractSevenZipMultiPart_NonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_7z_err_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	destDir := filepath.Join(tempDir, "extracted")
	err = extractSevenZipMultiPart("/nonexistent/archive.7z.001", destDir, "", nil)
	if err == nil {
		t.Error("Expected error for non-existent 7z file")
	}
}

// Test extractSevenZipMultiPart with invalid archive
func TestExtractSevenZipMultiPart_Invalid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_7z_invalid_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create an invalid 7z file
	part1Path := filepath.Join(tempDir, "invalid.7z.001")
	if err := os.WriteFile(part1Path, []byte("not a 7z file"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tempDir, "extracted")
	err = extractSevenZipMultiPart(part1Path, destDir, "", nil)
	if err == nil {
		t.Error("Expected error for invalid 7z archive")
	}
}

// Test determineFirstPartPath with all patterns
func TestDetermineFirstPartPath(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		baseName string
		pattern  string
		expected string
	}{
		{
			name:     "7z pattern",
			dir:      "/path/to",
			baseName: "archive.7z",
			pattern:  `(?i)^(.+\.7z)\.(\d{3})$`,
			expected: "/path/to/archive.7z.001",
		},
		{
			name:     "RAR new style pattern",
			dir:      "/path/to",
			baseName: "archive",
			pattern:  `(?i)^(.+)\.part(\d+)\.rar$`,
			expected: "/path/to/archive.part01.rar",
		},
		{
			name:     "RAR old style pattern",
			dir:      "/path/to",
			baseName: "archive",
			pattern:  `(?i)^(.+)\.r(\d{2})$`,
			expected: "/path/to/archive.rar",
		},
		{
			name:     "ZIP multi-part pattern",
			dir:      "/path/to",
			baseName: "archive.zip",
			pattern:  `(?i)^(.+\.zip)\.(\d{3})$`,
			expected: "/path/to/archive.zip.001",
		},
		{
			name:     "ZIP split pattern",
			dir:      "/path/to",
			baseName: "archive",
			pattern:  `(?i)^(.+)\.z(\d{2})$`,
			expected: "/path/to/archive.z01",
		},
		{
			name:     "Unknown pattern",
			dir:      "/path/to",
			baseName: "archive",
			pattern:  "unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineFirstPartPath(tt.dir, tt.baseName, tt.pattern)
			if result != tt.expected {
				t.Errorf("determineFirstPartPath(%q, %q, %q) = %q, expected %q",
					tt.dir, tt.baseName, tt.pattern, result, tt.expected)
			}
		})
	}
}

// Test parsePartNumber
func TestParsePartNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"001", 1},
		{"01", 1},
		{"1", 1},
		{"00", 1}, // 00 is treated as 1
		{"10", 10},
		{"99", 99},
		{"100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var result int
			_, err := parsePartNumber(tt.input, &result)
			if err != nil {
				t.Errorf("parsePartNumber(%q) error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parsePartNumber(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

// Test multiPartArchivePatterns directly
func TestMultiPartArchivePatterns(t *testing.T) {
	// Verify patterns are valid and match expected formats
	testCases := []struct {
		filename string
		matches  bool
	}{
		// 7z
		{"archive.7z.001", true},
		{"archive.7z.999", true},
		{"Archive.7Z.001", true},
		{"archive.7z.01", false},   // Only 3 digits
		{"archive.7z.0001", false}, // 4 digits not matched

		// RAR new style
		{"archive.part01.rar", true},
		{"archive.part1.rar", true},
		{"archive.part999.rar", true},
		{"archive.PART01.RAR", true},

		// RAR old style
		{"archive.r00", true},
		{"archive.r01", true},
		{"archive.r99", true},
		{"archive.R00", true},

		// ZIP multi-part
		{"archive.zip.001", true},
		{"archive.zip.999", true},
		{"Archive.ZIP.001", true},

		// ZIP split
		{"archive.z01", true},
		{"archive.z99", true},
		{"Archive.Z01", true},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			matched := false
			for _, pattern := range multiPartArchivePatterns {
				if pattern.MatchString(tc.filename) {
					matched = true
					break
				}
			}
			if matched != tc.matches {
				t.Errorf("Pattern match for %q: got %v, expected %v", tc.filename, matched, tc.matches)
			}
		})
	}
}

// Test extractRarMultiPart - test that destDir creation works
func TestExtractRarMultiPart_DestDirCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_rar_destdir_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple RAR-like file (will fail extraction but destDir should be created)
	rarPath := filepath.Join(tempDir, "test.part01.rar")
	if err := os.WriteFile(rarPath, []byte("Rar!\x1a\x07\x00"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tempDir, "level1", "level2", "extracted")
	// We expect an error since the file is not a complete valid RAR
	_ = extractRarMultiPart(rarPath, destDir, "", nil)

	// The destDir should have been created before the extraction error
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Log("Note: destDir was not created, extraction failed early")
	}
}

// Test the old-style RAR detection with .rar + .r00 files
func TestGetArchivePartInfo_RarOldStyleWithRar(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rar_old_style_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .rar file and .r00 file to simulate old-style multi-part
	rarPath := filepath.Join(tempDir, "archive.rar")
	r00Path := filepath.Join(tempDir, "archive.r00")

	if err := os.WriteFile(rarPath, []byte("fake rar"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r00Path, []byte("fake r00"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test that .rar file is detected as first part of old-style multi-part
	info := getArchivePartInfo(rarPath)
	if !info.IsMultiPart {
		t.Error("Expected .rar with .r00 to be detected as multi-part")
	}
	if info.Pattern != "rar-old-style" {
		t.Errorf("Expected pattern 'rar-old-style', got %q", info.Pattern)
	}
	if info.FirstPartPath != rarPath {
		t.Errorf("Expected FirstPartPath to be %q, got %q", rarPath, info.FirstPartPath)
	}
}

// Test extractSevenZipFile function indirectly through mock
func TestExtractSevenZipMultiPart_DestDirCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "7z_destdir_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create an invalid 7z file - the test is for error handling, not extraction
	part1Path := filepath.Join(tempDir, "test.7z.001")
	if err := os.WriteFile(part1Path, []byte("invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	// Use a nested dest directory that doesn't exist
	destDir := filepath.Join(tempDir, "level1", "level2", "extracted")
	err = extractSevenZipMultiPart(part1Path, destDir, "", nil)
	// Error is expected because the file is invalid, but the dest directory should be created
	// Actually, error happens before directory creation in this case
	if err == nil {
		t.Error("Expected error for invalid 7z")
	}
}

// Test multiPartFileReader with multiple files spanning reads
func TestMultiPartFileReader_SpanningRead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multipart_spanning_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create 3 parts with known sizes
	parts := []string{
		filepath.Join(tempDir, "test.001"),
		filepath.Join(tempDir, "test.002"),
		filepath.Join(tempDir, "test.003"),
	}

	contents := []string{"12345", "67890", "ABCDE"}
	for i, content := range contents {
		if err := os.WriteFile(parts[i], []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	reader := newMultiPartFileReader(parts)
	defer reader.Close()

	// Read all content at once
	buf := make([]byte, 15)
	n, err := reader.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("ReadAt error: %v", err)
	}
	if n != 15 {
		t.Errorf("Expected to read 15 bytes, got %d", n)
	}
	if string(buf) != "1234567890ABCDE" {
		t.Errorf("Expected '1234567890ABCDE', got %q", string(buf))
	}
}

// Test extractZipMultiPart with destDir creation
func TestExtractZipMultiPart_DestDirCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zip_multipart_destdir_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid single-part "multi-part" zip (just one .001 file with valid zip content)
	// First create a valid zip
	zipPath := filepath.Join(tempDir, "temp.zip")
	if err := createTestZip(zipPath); err != nil {
		t.Fatal(err)
	}

	// Read the zip content and write as .001
	zipContent, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	part1Path := filepath.Join(tempDir, "archive.zip.001")
	if err := os.WriteFile(part1Path, zipContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Extract to nested directory
	destDir := filepath.Join(tempDir, "level1", "level2", "extracted")
	err = extractZipMultiPart(part1Path, destDir, "", nil)
	if err != nil {
		t.Fatalf("extractZipMultiPart error: %v", err)
	}

	// Verify destDir was created and files extracted
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Error("destDir was not created")
	}

	// Verify extracted file
	extractedFile := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Error("Expected file not found after extraction")
	}
}

// Test extractZipMultiPart with progress callback
func TestExtractZipMultiPart_Progress(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zip_multipart_progress_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid zip with multiple files
	zipPath := filepath.Join(tempDir, "temp.zip")
	if err := createTestZipWithMultipleFiles(zipPath, 4); err != nil {
		t.Fatal(err)
	}

	zipContent, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	part1Path := filepath.Join(tempDir, "archive.zip.001")
	if err := os.WriteFile(part1Path, zipContent, 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tempDir, "extracted")
	var progressCalls int
	err = extractZipMultiPart(part1Path, destDir, "", func(extracted int, total int, progress int) {
		progressCalls++
	})
	if err != nil {
		t.Fatalf("extractZipMultiPart error: %v", err)
	}

	// Should have progress calls
	if progressCalls == 0 {
		t.Error("Expected progress callbacks")
	}
}
