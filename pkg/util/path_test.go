package util

import (
	"os"
	"path/filepath"
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
				t.Errorf("Filepath() = %v, want %v", got, tt.want)
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

func TestSafeRemoveAll(t *testing.T) {
	testDir := "test_dir"
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	doSafeRemoveAll(t, testDir, []string{
		"1.txt",
	}, []string{})

	doSafeRemoveAll(t, testDir, []string{
		"1.txt",
	}, []string{
		"2.txt",
	}, "2.txt")

	doSafeRemoveAll(t, testDir, []string{
		"1.txt",
		"a/b/c/1.txt",
		"a/b/1.txt",
		"a/1.txt",
	}, []string{})

	doSafeRemoveAll(t, testDir, []string{
		"1.txt",
		"a/b/c/1.txt",
		"a/b/1.txt",
		"a/1.txt",
	}, []string{
		"a/b/2.txt",
	}, "a/b/2.txt")

	doSafeRemoveAll(t, testDir, []string{
		"1.txt",
		"a/b/c/1.txt",
		"a/b/1.txt",
		"a/1.txt",
	}, []string{
		"a/2.txt",
	}, "a/2.txt")
}

func doSafeRemoveAll(t *testing.T, path string, downloadNames []string, otherNames []string, exist ...string) {
	preCreate(t, path, downloadNames)
	preCreate(t, path, otherNames)

	if err := SafeRemoveAll(path, downloadNames); err != nil {
		t.Fatal(err)
	}

	if len(exist) == 0 {
		for _, name := range downloadNames {
			filePath := filepath.Join(path, name)
			if isExist(filePath) {
				t.Fatalf("file %s should not exist", filePath)
			}
			subDirPath := filepath.Dir(name)
			if subDirPath != "." {
				dirPath := filepath.Join(path, subDirPath)
				if isExist(dirPath) {
					t.Fatalf("dir %s should not exist", dirPath)
				}
			}
		}
		return
	}

	for _, name := range exist {
		if !isExist(filepath.Join(path, name)) {
			t.Fatalf("file %s should exist", name)
		}
	}
}

func preCreate(t *testing.T, path string, names []string) {
	for _, name := range names {
		fullPath := filepath.Join(path, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
