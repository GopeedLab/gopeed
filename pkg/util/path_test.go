package util

import (
	"os"
	"testing"
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
