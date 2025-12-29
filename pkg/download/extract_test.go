package download

import (
	"archive/zip"
	"os"
	"path/filepath"
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
