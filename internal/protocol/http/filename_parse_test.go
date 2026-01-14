package http

import (
	"testing"
)

// TestParseFilenameWithAmpersand tests the fix for filenames containing & character
// Issue: filenames with & are HTML-encoded as &amp; and then truncated at the semicolon
func TestParseFilenameWithAmpersand(t *testing.T) {
	tests := []struct {
		name        string
		disposition string
		want        string
	}{
		{
			name:        "quoted filename with &amp;",
			disposition: `attachment; filename="查询处理&amp;优化.pptx"`,
			want:        "查询处理&优化.pptx",
		},
		{
			name:        "unquoted filename with &amp;",
			disposition: `attachment; filename=test&amp;file.txt`,
			want:        "test&file.txt",
		},
		{
			name:        "quoted filename with &amp; and extra params",
			disposition: `attachment; filename="test&amp;file.txt"; charset=utf-8`,
			want:        "test&file.txt",
		},
		{
			name:        "filename with multiple HTML entities",
			disposition: `attachment; filename="test&amp;&lt;&gt;.txt"`,
			want:        "test&<>.txt",
		},
		{
			name:        "normal filename without entities",
			disposition: `attachment; filename="normal.txt"`,
			want:        "normal.txt",
		},
		{
			name:        "filename with actual ampersand (no encoding)",
			disposition: `attachment; filename="test&file.txt"`,
			want:        "test&file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFilename(tt.disposition)
			if got != tt.want {
				t.Errorf("parseFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFindParamValueEnd tests the helper function for finding parameter value boundaries
func TestFindParamValueEnd(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{
			name:  "quoted value with semicolon inside",
			value: `"test&amp;file.txt"; charset=utf-8`,
			want:  19, // Position of semicolon after closing quote
		},
		{
			name:  "quoted value without semicolon after",
			value: `"test.txt"`,
			want:  -1,
		},
		{
			name:  "unquoted value with semicolon",
			value: `test.txt; charset=utf-8`,
			want:  8,
		},
		{
			name:  "unquoted value without semicolon",
			value: `test.txt`,
			want:  -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findParamValueEnd(tt.value)
			if got != tt.want {
				t.Errorf("findParamValueEnd() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestUnescapeHTMLEntities tests HTML entity unescaping
func TestUnescapeHTMLEntities(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "ampersand",
			in:   "test&amp;file.txt",
			want: "test&file.txt",
		},
		{
			name: "less than and greater than",
			in:   "&lt;test&gt;.txt",
			want: "<test>.txt",
		},
		{
			name: "quote",
			in:   "test&quot;file.txt",
			want: "test\"file.txt",
		},
		{
			name: "multiple entities",
			in:   "a&amp;b&lt;c&gt;d&quot;e",
			want: "a&b<c>d\"e",
		},
		{
			name: "no entities",
			in:   "normal.txt",
			want: "normal.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeHTMLEntities(tt.in)
			if got != tt.want {
				t.Errorf("unescapeHTMLEntities() = %q, want %q", got, tt.want)
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
			got := truncateFilename(tt.filename, tt.maxLength)
			if got != tt.want {
				t.Errorf("truncateFilename() = %q (len=%d), want %q (len=%d)",
					got, len(got), tt.want, len(tt.want))
			}
			// Verify result doesn't exceed maxLength
			if len(got) > tt.maxLength {
				t.Errorf("truncateFilename() result length %d exceeds maxLength %d",
					len(got), tt.maxLength)
			}
		})
	}
}
