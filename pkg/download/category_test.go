package download

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func newTestCategoryConfig() *base.DownloaderStoreConfig {
	return &base.DownloaderStoreConfig{
		DownloadDir:    "./downloads",
		AutoCategorize: true,
		Categories: []*base.DownloadCategory{
			{Name: "Music", Path: filepath.Join("./downloads", "Music"), Extensions: []string{"mp3", "flac"}},
			{Name: "Video", Path: filepath.Join("./downloads", "Video"), Extensions: []string{"mp4", "mkv"}},
			{Name: "Program", Path: filepath.Join("./downloads", "Program"), Extensions: []string{"exe", "msi"}},
		},
	}
}

func TestRouteDownloadPath(t *testing.T) {
	d := NewDownloader(nil)
	programDir := filepath.Join("./downloads", "Program")
	musicDir := filepath.Join("./downloads", "Music")

	tests := []struct {
		name string
		cfg  func(cfg *base.DownloaderStoreConfig)
		req  *base.Request
		opts *base.Options
		want string
	}{
		{
			name: "empty path routes to category",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{},
			want: programDir,
		},
		{
			name: "default dir routes to category",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: programDir,
		},
		{
			name: "custom dir is never touched",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./elsewhere"},
			want: "./elsewhere",
		},
		{
			name: "unknown extension stays",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/archive.zip"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "url name is matched case insensitively",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/Song.MP3"},
			opts: &base.Options{Path: "./downloads"},
			want: musicDir,
		},
		{
			name: "query without extension stays",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/download?id=1"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "rename without extension falls back to url",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads", Name: "readme"},
			want: programDir,
		},
		{
			name: "rename with extension wins over url",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads", Name: "song.mp3"},
			want: musicDir,
		},
		{
			name: "disabled switch keeps directory",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.AutoCategorize = false
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "no categories keeps directory",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = nil
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "deleted category is skipped",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories[2].IsDeleted = true
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "category without path is skipped",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories[2].Path = ""
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "built-in category without persisted extensions never matches",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "", NameKey: "categoryProgram", Path: filepath.Join("./downloads", "Program")},
				}
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "non-matching extension keeps the download dir",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "", NameKey: "categoryProgram", Path: filepath.Join("./downloads", "Program"), Extensions: []string{"msi"}},
				}
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: "./downloads",
		},
		{
			name: "messy stored extensions still match after normalization",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "Program", Path: filepath.Join("./downloads", "Program"), Extensions: []string{" exe, msi ", "..exe", "MSI"}},
				}
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: filepath.Join("./downloads", "Program"),
		},
		{
			name: "dot prefixed extension still matches",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "Program", Path: filepath.Join("./downloads", "Program"), Extensions: []string{".exe"}},
				}
			},
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads"},
			want: filepath.Join("./downloads", "Program"),
		},
		{
			name: "magnet task never routes",
			cfg:  nil,
			req: &base.Request{
				URL: "magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a",
			},
			opts: &base.Options{Path: "./downloads", Name: "setup.exe"},
			want: "./downloads",
		},
		{
			name: "https torrent file routes like a plain http download",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "Torrent", Path: filepath.Join("./downloads", "Torrent"), Extensions: []string{"torrent"}},
				}
			},
			req:  &base.Request{URL: "https://example.com/files/setup.torrent"},
			opts: &base.Options{Path: "./downloads"},
			want: filepath.Join("./downloads", "Torrent"),
		},
		{
			name: "ed2k url routes by file name",
			cfg:  nil,
			req:  &base.Request{URL: "ed2k://|file|setup.exe|1024|0123456789ABCDEF0123456789ABCDEF|/"},
			opts: &base.Options{Path: "./downloads"},
			want: programDir,
		},
		{
			name: "placeholder paths compare equal",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.DownloadDir = "./downloads/%year%"
				cfg.Categories[0].Path = filepath.Join("./downloads/%year%", "Music")
				cfg.Categories[1].Path = filepath.Join("./downloads/%year%", "Video")
				cfg.Categories[2].Path = filepath.Join("./downloads/%year%", "Program")
			},
			req:  &base.Request{URL: "https://example.com/files/Song.mp3"},
			opts: &base.Options{Path: "./downloads/%year%"},
			want: filepath.Join("./downloads/%year%", "Music"),
		},
		{
			name: "trailing separator still equals default dir",
			cfg:  nil,
			req:  &base.Request{URL: "https://example.com/files/setup.exe"},
			opts: &base.Options{Path: "./downloads/"},
			want: programDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestCategoryConfig()
			if tt.cfg != nil {
				tt.cfg(cfg)
			}
			got := d.routeDownloadPath(tt.req, tt.opts, cfg)
			if got != tt.want {
				t.Fatalf("routeDownloadPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeCategoryExtensions(t *testing.T) {
	tests := []struct {
		raw  []string
		want []string
	}{
		{[]string{"exe, msi"}, []string{"exe", "msi"}},
		{[]string{" mp3 ", "..exe", "MUSIC"}, []string{"mp3", "exe", "music"}},
		{[]string{"zip", "ZIP", "zip"}, []string{"zip"}},
		{[]string{"  ", "..", ""}, nil},
		{nil, nil},
	}
	for _, tt := range tests {
		got := normalizeCategoryExtensions(tt.raw)
		if len(got) != len(tt.want) {
			t.Fatalf("normalizeCategoryExtensions(%q) = %q, want %q", tt.raw, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("normalizeCategoryExtensions(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		}
	}
}

func TestFileNameFromURL(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
	}{
		{"https://example.com/path/setup.exe", "setup.exe"},
		{"https://example.com/path/setup.exe?token=1#frag", "setup.exe"},
		{"https://example.com/download?id=1", "download"},
		{"file:///C:/dir/setup.exe", "setup.exe"},
		{`file://C:\dir\setup.exe`, "setup.exe"},
		{"ed2k://|file|setup.exe|1024|0123456789ABCDEF0123456789ABCDEF|/", "setup.exe"},
		{"data:application/octet-stream;base64,AAAA", ""},
		{"magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a", ""},
	}
	for _, tt := range tests {
		if got := fileNameFromURL(tt.rawURL); got != tt.want {
			t.Errorf("fileNameFromURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
		}
	}
}

func TestInitOptionsWhiteListFallsBackFromCategoryDir(t *testing.T) {
	defaultDir := filepath.ToSlash(t.TempDir())
	categoryDir := defaultDir + "/Program"

	downloader := NewDownloader(&DownloaderConfig{WhiteDownloadDirs: []string{defaultDir}})
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DownloadDir = defaultDir
	cfg.AutoCategorize = true
	cfg.Categories = []*base.DownloadCategory{
		{Name: "Program", Path: categoryDir, Extensions: []string{"exe"}},
	}
	if err := downloader.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// The routed category directory falls outside the white list, so the
	// task falls back to the default directory instead of failing.
	opts, err := downloader.initOptions(&base.Request{URL: "https://example.com/setup.exe"}, &base.Options{Path: defaultDir})
	if err != nil {
		t.Fatalf("expected white list fallback, got %v", err)
	}
	if opts.Path != defaultDir {
		t.Fatalf("path = %q, want fallback to %q", opts.Path, defaultDir)
	}

	// An explicitly chosen directory outside the white list still fails.
	_, err = downloader.initOptions(&base.Request{URL: "https://example.com/setup.exe"}, &base.Options{Path: defaultDir + "/Other"})
	if err == nil {
		t.Fatal("expected white list error for an explicit directory")
	}
}

func TestDownloader_CreateDirectAutoCategorize(t *testing.T) {
	listener := test.StartTestFileServer()
	defer listener.Close()

	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	}()

	defaultDir := t.TempDir()
	categoryDir := filepath.Join(defaultDir, "Program")
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DownloadDir = defaultDir
	cfg.AutoCategorize = true
	cfg.Categories = []*base.DownloadCategory{
		{Name: "Program", Path: categoryDir, Extensions: []string{"exe"}},
	}
	if err := downloader.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{}, 2)
	downloader.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			done <- struct{}{}
		}
	})
	waitDone := func() {
		t.Helper()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("task did not finish")
		}
	}

	// The request url carries no usable extension, routing relies on the task
	// name only, exactly like the create dialog rename field does.
	req := &base.Request{URL: "http://" + listener.Addr().String() + "/" + test.BuildName}
	taskID, err := downloader.CreateDirect(req, &base.Options{
		Path: defaultDir,
		Name: "setup.exe",
	})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("task path = %q, want category dir %q", task.Meta.Opts.Path, categoryDir)
	}
	waitDone()
	if _, err := os.Stat(filepath.Join(categoryDir, "setup.exe")); err != nil {
		t.Fatalf("categorized file missing: %v", err)
	}

	// An explicitly chosen directory must win over the category rule.
	customDir := t.TempDir()
	taskID, err = downloader.CreateDirect(&base.Request{URL: "http://" + listener.Addr().String() + "/" + test.BuildName}, &base.Options{
		Path: customDir,
		Name: "setup.exe",
	})
	if err != nil {
		t.Fatal(err)
	}
	task = downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	if task.Meta.Opts.Path != customDir {
		t.Fatalf("task path = %q, want custom dir %q", task.Meta.Opts.Path, customDir)
	}
	waitDone()
	if _, err := os.Stat(filepath.Join(customDir, "setup.exe")); err != nil {
		t.Fatalf("custom dir file missing: %v", err)
	}
}
