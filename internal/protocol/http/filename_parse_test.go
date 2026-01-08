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
