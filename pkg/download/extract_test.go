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
		{"file.zip", true},
		{"file.ZIP", true},
		{"file.tar", true},
		{"file.tar.gz", true},
		{"file.tgz", true},
		{"file.tar.bz2", true},
		{"file.tbz2", true},
		{"file.tar.xz", true},
		{"file.txz", true},
		{"file.rar", true},
		{"file.RAR", true},
		{"file.7z", true},
		{"file.gz", true},
		{"file.bz2", true},
		{"file.xz", true},
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
	err = extractArchive(zipPath, destDir, "")
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
	err = extractArchive(txtPath, destDir, "")
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
