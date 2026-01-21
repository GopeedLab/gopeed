package util

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDir(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty path",
			args: args{
				path: ".",
			},
			want: "",
		},
		{
			name: "normal path case 1",
			args: args{
				path: "./a/b/c/1.txt",
			},
			want: "a/b/c",
		},
		{
			name: "normal path case 2",
			args: args{
				path: "a/b/c/1.txt",
			},
			want: "a/b/c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Dir(tt.args.path); got != tt.want {
				t.Errorf("Dir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilepath(t *testing.T) {
	type args struct {
		path       string
		originName string
		customName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "origin name",
			args: args{
				path:       "/Downloads",
				originName: "1.txt",
				customName: "",
			},
			want: "/Downloads/1.txt",
		},
		{
			name: "origin name",
			args: args{
				path:       "/Downloads",
				originName: "1.txt",
				customName: "2.txt",
			},
			want: "/Downloads/2.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filepath(tt.args.path, tt.args.originName, tt.args.customName); got != tt.want {
				t.Errorf("SingleFilepath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeRemove(t *testing.T) {
	name := "test_safe_remove.data"
	file, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := SafeRemove(name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); err == nil {
		t.Fatal(err)
	}

	if err := SafeRemove("test_safe_remove_not_exist.data"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckDuplicateAndRename(t *testing.T) {
	// Test with extension
	doCheckDuplicateAndRename(t, []string{}, "a.txt", "a.txt")
	doCheckDuplicateAndRename(t, []string{"a.txt"}, "a.txt", "a (1).txt")
	doCheckDuplicateAndRename(t, []string{"a.txt", "a (1).txt"}, "a.txt", "a (2).txt")

	// Test without extension
	doCheckDuplicateAndRename(t, []string{}, "a", "a")
	doCheckDuplicateAndRename(t, []string{"a"}, "a", "a (1)")
	doCheckDuplicateAndRename(t, []string{"a", "a (1)"}, "a", "a (2)")

	// Test hidden files (starting with dot)
	doCheckDuplicateAndRename(t, []string{}, ".gitignore", ".gitignore")
	doCheckDuplicateAndRename(t, []string{".gitignore"}, ".gitignore", ".gitignore (1)")
	doCheckDuplicateAndRename(t, []string{".gitignore", ".gitignore (1)"}, ".gitignore", ".gitignore (2)")

	// Test hidden files with extension
	doCheckDuplicateAndRename(t, []string{}, ".config.json", ".config.json")
	doCheckDuplicateAndRename(t, []string{".config.json"}, ".config.json", ".config (1).json")

	// Test multiple dots
	doCheckDuplicateAndRename(t, []string{}, "test.tar.gz", "test.tar.gz")
	doCheckDuplicateAndRename(t, []string{"test.tar.gz"}, "test.tar.gz", "test.tar (1).gz")
}

func doCheckDuplicateAndRename(t *testing.T, exitsPaths []string, path string, except string) {
	for _, path := range exitsPaths {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}

	defer func() {
		for _, path := range exitsPaths {
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
		}
	}()

	got, err := CheckDuplicateAndRename(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != except {
		t.Errorf("CheckDuplicateAndRename() = %v, want %v", got, except)
	}
}

func TestIsExistsFile(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "exist",
			args: args{
				path: "./path.go",
			},
			want: true,
		},
		{
			name: "not exist",
			args: args{
				path: "./path_not_exist.go",
			},
			want: false,
		},
		{
			name: "is dir",
			args: args{
				path: "../util",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExistsFile(tt.args.path); got != tt.want {
				t.Errorf("IsExistsFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplaceInvalidFilename(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "blank",
			args: args{
				path: "",
			},
			want: "",
		},
		{
			name: "normal",
			args: args{
				path: "test.txt",
			},
			want: "test.txt",
		},
		{
			name: "case1",
			args: args{
				path: "te/st.txt",
			},
			want: "te_st.txt",
		},
		{
			name: "case2",
			args: args{
				path: "te/st:.txt",
			},
			want: "te_st_.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplaceInvalidFilename(tt.args.path); got != tt.want {
				t.Errorf("ReplaceInvalidFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSafeFilename tests the combined filename sanitization functionality
func TestSafeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "short filename - no changes needed",
			filename: "test.txt",
			want:     "test.txt",
		},
		{
			name:     "empty filename",
			filename: "",
			want:     "",
		},
		{
			name:     "invalid chars only",
			filename: "te/st:file.txt",
			want:     "te_st_file.txt",
		},
		{
			name:     "long filename only",
			filename: "this_is_a_very_long_filename_that_exceeds_the_maximum_allowed_length_and_should_be_truncated_properly.txt",
			want:     "this_is_a_very_long_filename_that_exceeds_the_maximum_allowed_length_and_should_be_truncated_pro.txt",
		},
		{
			name:     "both invalid chars and too long",
			filename: "path/to/very:long*filename?that<exceeds>filesystem|limits_and_has_invalid_characters_everywhere.pdf",
			want:     "path_to_very_long*filename?that<exceeds>filesystem|limits_and_has_invalid_characters_everywhere.pdf",
		},
		{
			name:     "unicode with invalid chars and truncation",
			filename: "测试/文件名:非常长的中文文件名_需要被截断_这是一个测试用的超长文件名.pdf",
			want:     "测试_文件名_非常长的中文文件名_需要被截断_这是一个测试用的超长文.pdf",
		},
		{
			name:     "hidden file with truncation",
			filename: ".gitignore_with_very_long_name_that_needs_truncation_and_more_characters_to_exceed_the_maximum_length",
			want:     ".gitignore_with_very_long_name_that_needs_truncation_and_more_characters_to_exceed_the_maximum_lengt",
		},
		{
			name:     "multiple dots and invalid chars",
			filename: "archive/tar.gz.backup:old.txt",
			want:     "archive_tar.gz.backup_old.txt",
		},
		{
			name:     "extension longer than reasonable",
			filename: "test.verylongextensionthatshouldnotbetreatedasextension_with_more_characters_to_exceed_maximum_length",
			want:     "test.verylongextensionthatshouldnotbetreatedasextension_with_more_characters_to_exceed_maximum_lengt",
		},
		{
			name:     "filename with spaces and invalid chars",
			filename: "my document/with:spaces and a very long name that needs to be truncated because it exceeds the maximum length.docx",
			want:     "my document_with_spaces and a very long name that needs to be truncated because it exceeds the .docx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeFilename(tt.filename)
			if got != tt.want {
				t.Errorf("SafeFilename() = %q (len=%d), want %q (len=%d)",
					got, len(got), tt.want, len(tt.want))
			}
			// Verify result doesn't exceed MaxFilenameLength
			if len(got) > MaxFilenameLength {
				t.Errorf("SafeFilename() result length %d exceeds MaxFilenameLength %d",
					len(got), MaxFilenameLength)
			}
		})
	}
}

func TestReplacePathPlaceholders(t *testing.T) {
	now := time.Now()
	year := fmt.Sprintf("%d", now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())
	date := fmt.Sprintf("%s-%s-%s", year, month, day)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "no placeholders",
			path: "/home/user/Downloads",
			want: "/home/user/Downloads",
		},
		{
			name: "year placeholder",
			path: "/Downloads/%year%",
			want: "/Downloads/" + year,
		},
		{
			name: "month placeholder",
			path: "/Downloads/%month%",
			want: "/Downloads/" + month,
		},
		{
			name: "day placeholder",
			path: "/Downloads/%day%",
			want: "/Downloads/" + day,
		},
		{
			name: "date placeholder",
			path: "/Downloads/%date%",
			want: "/Downloads/" + date,
		},
		{
			name: "multiple placeholders",
			path: "/Downloads/%year%-%month%",
			want: "/Downloads/" + year + "-" + month,
		},
		{
			name: "mixed path with placeholders",
			path: "/home/user/Downloads/%year%/%month%/%day%",
			want: "/home/user/Downloads/" + year + "/" + month + "/" + day,
		},
		{
			name: "windows style path",
			path: "D:\\Downloads\\%year%-%month%",
			want: "D:\\Downloads\\" + year + "-" + month,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplacePathPlaceholders(tt.path)
			if !strings.Contains(got, year) && tt.path != "" && strings.Contains(tt.path, "%year%") {
				t.Errorf("ReplacePathPlaceholders() = %v, want containing year %v", got, year)
			}
			if got != tt.want {
				t.Errorf("ReplacePathPlaceholders() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTruncateFilename tests the filename truncation functionality
func TestTruncateFilename(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		maxLength int
		want      string
	}{
		{
			name:      "short filename - no truncation",
			filename:  "test.txt",
			maxLength: 100,
			want:      "test.txt",
		},
		{
			name:      "filename at exact limit",
			filename:  "abcdefghij.txt", // 14 chars
			maxLength: 14,
			want:      "abcdefghij.txt",
		},
		{
			name:      "long filename with extension - truncate base",
			filename:  "this_is_a_very_long_filename_that_exceeds_the_maximum_allowed_length_and_should_be_truncated_properly.txt",
			maxLength: 100,
			want:      "this_is_a_very_long_filename_that_exceeds_the_maximum_allowed_length_and_should_be_truncated_pro.txt",
		},
		{
			name:      "long filename without extension",
			filename:  "this_is_a_very_long_filename_without_extension_that_exceeds_maximum_length_and_needs_truncation",
			maxLength: 100,
			want:      "this_is_a_very_long_filename_without_extension_that_exceeds_maximum_length_and_needs_truncation",
		},
		{
			name:      "filename with multiple dots",
			filename:  "archive.tar.gz.backup.old.file.with.many.dots.txt",
			maxLength: 30,
			want:      "archive.tar.gz.backup.old..txt", // Preserves last extension
		},
		{
			name:      "hidden file (starts with dot)",
			filename:  ".gitignore_with_very_long_name_that_needs_truncation",
			maxLength: 30,
			want:      ".gitignore_with_very_long_name",
		},
		{
			name:      "only extension (no base name)",
			filename:  ".txt",
			maxLength: 100,
			want:      ".txt",
		},
		{
			name:      "unicode characters in filename",
			filename:  "测试文件名_非常长的中文文件名_需要被截断_这是一个测试用的超长文件名.pdf",
			maxLength: 50,
			want:      "测试文件名_非常长的中文文件名_.pdf", // Truncated at byte boundary, preserving UTF-8
		},
		{
			name:      "extension longer than reasonable (>20 chars)",
			filename:  "test.verylongextensionthatshouldnotbetreatedasextension",
			maxLength: 30,
			want:      "test.verylongextensionthatshou",
		},
		{
			name:      "very short max length with extension",
			filename:  "document.pdf",
			maxLength: 10,
			want:      "docume.pdf",
		},
		{
			name:      "maxLength smaller than extension",
			filename:  "test.pdf",
			maxLength: 3,
			want:      "tes",
		},
		{
			name:      "empty filename",
			filename:  "",
			maxLength: 100,
			want:      "",
		},
		{
			name:      "filename with spaces",
			filename:  "my document with spaces and a very long name that needs to be truncated.docx",
			maxLength: 50,
			want:      "my document with spaces and a very long name .docx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateFilename(tt.filename, tt.maxLength)
			if got != tt.want {
				t.Errorf("TruncateFilename() = %q (len=%d), want %q (len=%d)",
					got, len(got), tt.want, len(tt.want))
			}
			// Verify result doesn't exceed maxLength
			if len(got) > tt.maxLength {
				t.Errorf("TruncateFilename() result length %d exceeds maxLength %d",
					len(got), tt.maxLength)
			}
		})
	}
}
