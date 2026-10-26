package download

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
)

func newTestCategoryConfig() *base.DownloaderStoreConfig {
	return &base.DownloaderStoreConfig{
		DownloadDir:    "./downloads",
		AutoCategorize: util.BoolPtr(true),
		Categories: []*base.DownloadCategory{
			{Name: "Music", Path: filepath.Join("./downloads", "Music"), Extensions: []string{"mp3", "flac"}},
			{Name: "Video", Path: filepath.Join("./downloads", "Video"), Extensions: []string{"mp4", "mkv"}},
			{Name: "Program", Path: filepath.Join("./downloads", "Program"), Extensions: []string{"exe", "msi"}},
		},
	}
}

// singleFileResource is the resource shape of a one file task: no folder name
// and a single file entry.
func singleFileResource(name string) *base.Resource {
	return &base.Resource{Files: []*base.FileInfo{{Name: name, Size: 1}}}
}

// folderResource is the resource shape of a multi file task: a folder name and
// one entry per file, even when only one of them is selected later.
func folderResource(name string, files ...string) *base.Resource {
	res := &base.Resource{Name: name}
	for _, file := range files {
		res.Files = append(res.Files, &base.FileInfo{Name: file, Size: 1})
	}
	return res
}

func newUnitCategoryDownloader(cfg *base.DownloaderStoreConfig) *Downloader {
	d := NewDownloader(nil)
	d.cfg.DownloaderStoreConfig = cfg
	return d
}

func TestDecideDownloadDir(t *testing.T) {
	programDir := filepath.Join("./downloads", "Program")

	tests := []struct {
		name string
		cfg  func(cfg *base.DownloaderStoreConfig)
		opts *base.Options
		res  *base.Resource
		want string
	}{
		{
			name: "matching single file routes to the category",
			opts: &base.Options{},
			res:  singleFileResource("setup.exe"),
			want: programDir,
		},
		{
			name: "an explicit base directory is routed too",
			opts: &base.Options{Path: "./elsewhere"},
			res:  singleFileResource("setup.exe"),
			want: programDir,
		},
		{
			name: "skip flag keeps the configured download directory",
			opts: &base.Options{Path: "./downloads", SkipAutoCategorize: true},
			res:  singleFileResource("setup.exe"),
			want: "./downloads",
		},
		{
			name: "skip flag keeps an explicit directory",
			opts: &base.Options{Path: "./elsewhere", SkipAutoCategorize: true},
			res:  singleFileResource("setup.exe"),
			want: "./elsewhere",
		},
		{
			name: "switch off keeps the base directory",
			cfg:  func(cfg *base.DownloaderStoreConfig) { cfg.AutoCategorize = util.BoolPtr(false) },
			opts: &base.Options{},
			res:  singleFileResource("setup.exe"),
			want: "./downloads",
		},
		{
			name: "no categories keeps the base directory",
			cfg:  func(cfg *base.DownloaderStoreConfig) { cfg.Categories = nil },
			opts: &base.Options{},
			res:  singleFileResource("setup.exe"),
			want: "./downloads",
		},
		{
			name: "unknown extension keeps the base directory",
			opts: &base.Options{},
			res:  singleFileResource("archive.zip"),
			want: "./downloads",
		},
		{
			name: "a name without an extension keeps the base directory",
			opts: &base.Options{},
			res:  singleFileResource("README"),
			want: "./downloads",
		},
		{
			name: "a deleted category is skipped",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "Program", Path: programDir, Extensions: []string{"exe"}, IsDeleted: true},
				}
			},
			opts: &base.Options{},
			res:  singleFileResource("setup.exe"),
			want: "./downloads",
		},
		{
			name: "a category without a path is skipped",
			cfg: func(cfg *base.DownloaderStoreConfig) {
				cfg.Categories = []*base.DownloadCategory{
					{Name: "Program", Path: "  ", Extensions: []string{"exe"}},
				}
			},
			opts: &base.Options{},
			res:  singleFileResource("setup.exe"),
			want: "./downloads",
		},
		{
			name: "a multi file resource keeps its layout",
			opts: &base.Options{},
			res:  folderResource("bundle", "setup.exe", "readme.txt"),
			want: "./downloads",
		},
		{
			name: "a multi file resource with one match still keeps its layout",
			opts: &base.Options{SelectFiles: []int{0}},
			res:  folderResource("bundle", "setup.exe", "readme.txt"),
			want: "./downloads",
		},
		{
			name: "a single file inside a folder resource is not routed",
			opts: &base.Options{},
			res:  folderResource("bundle", "setup.exe"),
			want: "./downloads",
		},
		{
			name: "a missing resource keeps the base directory",
			opts: &base.Options{},
			res:  nil,
			want: "./downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestCategoryConfig()
			if tt.cfg != nil {
				tt.cfg(cfg)
			}
			d := newUnitCategoryDownloader(cfg)
			got, err := d.decideDownloadDir(tt.opts, tt.res)
			if err != nil {
				t.Fatalf("decideDownloadDir() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("decideDownloadDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecideDownloadDirWhiteList(t *testing.T) {
	defaultDir := filepath.ToSlash(t.TempDir())
	categoryDir := defaultDir + "/Program"
	newConfig := func() *base.DownloaderStoreConfig {
		cfg := newTestCategoryConfig()
		cfg.DownloadDir = defaultDir
		cfg.Categories = []*base.DownloadCategory{
			{Name: "Program", Path: categoryDir, Extensions: []string{"exe"}},
		}
		return cfg
	}

	t.Run("a rejected category directory falls back to the base directory", func(t *testing.T) {
		d := newUnitCategoryDownloader(newConfig())
		d.cfg.WhiteDownloadDirs = []string{defaultDir}

		got, err := d.decideDownloadDir(&base.Options{Path: defaultDir}, singleFileResource("setup.exe"))
		if err != nil {
			t.Fatalf("expected white list fallback, got %v", err)
		}
		if got != defaultDir {
			t.Fatalf("path = %q, want the base directory %q", got, defaultDir)
		}
	})

	t.Run("a rejected base directory is an error", func(t *testing.T) {
		d := newUnitCategoryDownloader(newConfig())
		d.cfg.WhiteDownloadDirs = []string{defaultDir}

		_, err := d.decideDownloadDir(&base.Options{Path: defaultDir + "/Other", SkipAutoCategorize: true}, singleFileResource("setup.exe"))
		if err == nil {
			t.Fatal("expected a white list error for an explicit directory")
		}
	})

	t.Run("equivalent category paths still fall back to the allowed spelling", func(t *testing.T) {
		cfg := newConfig()
		cfg.Categories[0].Path = defaultDir + "/."
		d := newUnitCategoryDownloader(cfg)
		d.cfg.WhiteDownloadDirs = []string{defaultDir}
		got, err := d.decideDownloadDir(&base.Options{Path: defaultDir}, singleFileResource("setup.exe"))
		if err != nil || got != defaultDir {
			t.Fatalf("decideDownloadDir() = %q, %v; want %q, nil", got, err, defaultDir)
		}
	})

	t.Run("an empty base directory is an error", func(t *testing.T) {
		cfg := newConfig()
		cfg.DownloadDir = ""
		d := newUnitCategoryDownloader(cfg)

		_, err := d.decideDownloadDir(&base.Options{SkipAutoCategorize: true}, singleFileResource("setup.exe"))
		if err == nil {
			t.Fatal("expected an error for an empty download directory")
		}
	})
}

func TestSuggestedDownloadDir(t *testing.T) {
	programDir := filepath.Join("./downloads", "Program")

	tests := []struct {
		name string
		cfg  func(cfg *base.DownloaderStoreConfig)
		res  *base.Resource
		want string
	}{
		{name: "a matching single file is suggested", res: singleFileResource("setup.exe"), want: programDir},
		{
			name: "a padded category path matches the creation decision",
			cfg:  func(cfg *base.DownloaderStoreConfig) { cfg.Categories[2].Path = "  " + programDir + "  " },
			res:  singleFileResource("setup.exe"),
			want: programDir,
		},
		{
			name: "the switch off suggests nothing",
			cfg:  func(cfg *base.DownloaderStoreConfig) { cfg.AutoCategorize = util.BoolPtr(false) },
			res:  singleFileResource("setup.exe"),
			want: "",
		},
		{name: "a multi file resource suggests nothing", res: folderResource("bundle", "setup.exe"), want: ""},
		{name: "a folder resource suggests nothing", res: folderResource("bundle", "setup.exe", "a.txt"), want: ""},
		{name: "an unknown extension suggests nothing", res: singleFileResource("archive.zip"), want: ""},
		{name: "a missing resource suggests nothing", res: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestCategoryConfig()
			if tt.cfg != nil {
				tt.cfg(cfg)
			}
			d := newUnitCategoryDownloader(cfg)
			if got := d.suggestedDownloadDir(tt.res); got != tt.want {
				t.Fatalf("suggestedDownloadDir() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("a rejected category directory suggests nothing", func(t *testing.T) {
		dir := filepath.ToSlash(t.TempDir())
		cfg := newTestCategoryConfig()
		cfg.DownloadDir = dir
		cfg.Categories = []*base.DownloadCategory{
			{Name: "Program", Path: dir + "/Program", Extensions: []string{"exe"}},
		}
		d := newUnitCategoryDownloader(cfg)
		d.cfg.WhiteDownloadDirs = []string{dir}

		if got := d.suggestedDownloadDir(singleFileResource("setup.exe")); got != "" {
			t.Fatalf("suggestedDownloadDir() = %q, want an empty suggestion", got)
		}
	})
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

// startCategoryFileServer serves a nameless url whose real file name only
// shows up in the Content-Disposition header.
func startCategoryFileServer(t *testing.T, fileName string, data []byte) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeContent(w, r, fileName, time.Time{}, bytes.NewReader(data))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL + "/download"
}

func newSetupCategoryDownloader(t *testing.T) *Downloader {
	t.Helper()
	downloader := NewDownloader(nil)
	if err := downloader.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		downloader.Delete(nil, true)
		downloader.Clear()
	})
	return downloader
}

func putCategoryConfig(t *testing.T, d *Downloader, downloadDir string, autoCategorize bool, extensions []string) string {
	t.Helper()
	categoryDir := filepath.Join(downloadDir, "Program")
	cfg, err := d.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.DownloadDir = downloadDir
	cfg.AutoCategorize = util.BoolPtr(autoCategorize)
	cfg.Categories = []*base.DownloadCategory{
		{Name: "Program", Path: categoryDir, Extensions: extensions},
	}
	if err := d.PutConfig(cfg); err != nil {
		t.Fatal(err)
	}
	return categoryDir
}

// watchTaskDone subscribes to the done event before the task is created, so a
// fast task can not finish before the test starts listening.
func watchTaskDone(t *testing.T, d *Downloader) <-chan struct{} {
	t.Helper()
	done := make(chan struct{}, 1)
	d.Listener(func(event *Event) {
		if event.Key == EventKeyDone {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})
	return done
}

func waitTaskDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("task did not finish")
	}
}

func TestDownloader_CreateDirectAutoCategorize(t *testing.T) {
	// The url carries no extension, the category is matched on the file name
	// of the resolved resource.
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	done := watchTaskDone(t, downloader)

	taskID, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskDone(t, done)
	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("task path = %q, want the category dir %q", task.Meta.Opts.Path, categoryDir)
	}
	if _, err := os.Stat(filepath.Join(categoryDir, "setup.data")); err != nil {
		t.Fatalf("categorized file missing: %v", err)
	}
}

func TestDownloader_CreateDirectDoesNotMatchTheRename(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	putCategoryConfig(t, downloader, downloadDir, true, []string{"exe"})
	done := watchTaskDone(t, downloader)

	// opts.Name only renames the saved file, it never decides the category.
	taskID, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{
		Path: downloadDir,
		Name: "renamed.exe",
	})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskDone(t, done)
	if task.Meta.Opts.Path != downloadDir {
		t.Fatalf("task path = %q, want the base dir %q", task.Meta.Opts.Path, downloadDir)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "renamed.exe")); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
}

func TestDownloader_CreateDirectSkipAutoCategorize(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	chosenDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	done := watchTaskDone(t, downloader)

	taskID, err := downloader.CreateDirect(&base.Request{URL: url}, &base.Options{
		Path:               chosenDir,
		SkipAutoCategorize: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskDone(t, done)
	if task.Meta.Opts.Path != chosenDir {
		t.Fatalf("task path = %q, want the chosen dir %q", task.Meta.Opts.Path, chosenDir)
	}
	if _, err := os.Stat(filepath.Join(chosenDir, "setup.data")); err != nil {
		t.Fatalf("file missing in the chosen dir: %v", err)
	}
}

func TestDownloader_CreateWithOptionsAutoCategorize(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	done := watchTaskDone(t, downloader)

	rr, err := downloader.Resolve(&base.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	if rr.SuggestedDownloadDir != categoryDir {
		t.Fatalf("suggested dir = %q, want %q", rr.SuggestedDownloadDir, categoryDir)
	}
	taskID, err := downloader.CreateWithOptions(rr.ID, &base.Options{Path: downloadDir})
	if err != nil {
		t.Fatal(err)
	}
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	waitTaskDone(t, done)
	if task.Meta.Opts.Path != categoryDir {
		t.Fatalf("task path = %q, want the category dir %q", task.Meta.Opts.Path, categoryDir)
	}
}

func TestDownloader_CreateWithOptionsKeepsTheDefaultDirectory(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})
	done := watchTaskDone(t, downloader)

	rr, err := downloader.Resolve(&base.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := downloader.CreateWithOptions(rr.ID, &base.Options{
		Path:          downloadDir,
		AsDefaultPath: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitTaskDone(t, done)

	// The task lands in the category directory, but that must not become the
	// global default download directory.
	task := downloader.GetTask(taskID)
	if task == nil {
		t.Fatal("task not found")
	}
	if task.Meta.Opts.Path == downloadDir {
		t.Fatalf("task path = %q, want the category dir", task.Meta.Opts.Path)
	}
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DownloadDir != downloadDir {
		t.Fatalf("default download dir = %q, want the base dir %q", cfg.DownloadDir, downloadDir)
	}
}

func TestDownloader_ResolveSuggestsWithoutSideEffects(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	categoryDir := putCategoryConfig(t, downloader, downloadDir, true, []string{"data"})

	rr, err := downloader.Resolve(&base.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	if rr.SuggestedDownloadDir != categoryDir {
		t.Fatalf("suggested dir = %q, want %q", rr.SuggestedDownloadDir, categoryDir)
	}
	if _, err := os.Stat(categoryDir); !os.IsNotExist(err) {
		t.Fatalf("resolve created the suggested directory: %v", err)
	}
	cfg, err := downloader.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DownloadDir != downloadDir {
		t.Fatalf("resolve changed the download dir to %q", cfg.DownloadDir)
	}
}

func TestDownloader_ResolveSuggestsNothingWhenTheSwitchIsOff(t *testing.T) {
	url := startCategoryFileServer(t, "setup.data", []byte("0123456789"))
	downloadDir := t.TempDir()
	downloader := newSetupCategoryDownloader(t)
	putCategoryConfig(t, downloader, downloadDir, false, []string{"data"})

	rr, err := downloader.Resolve(&base.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	if rr.SuggestedDownloadDir != "" {
		t.Fatalf("suggested dir = %q, want an empty suggestion", rr.SuggestedDownloadDir)
	}
}

func TestDownloader_SetupMigratesLegacyExtraCategories(t *testing.T) {
	storage := NewMemStorage()
	if err := storage.Setup([]string{bucketConfig}); err != nil {
		t.Fatal(err)
	}
	legacy := &base.DownloaderStoreConfig{
		DownloadDir: t.TempDir(),
		Extra: map[string]any{
			"downloadCategories": []any{
				map[string]any{"name": "", "path": filepath.Join(t.TempDir(), "Music"), "isBuiltIn": true, "nameKey": "categoryMusic"},
			},
		},
	}
	if err := storage.Put(bucketConfig, "config", legacy); err != nil {
		t.Fatal(err)
	}

	downloader := NewDownloader(&DownloaderConfig{Storage: storage, StorageDir: t.TempDir()})
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
	if len(cfg.Categories) != 1 || cfg.Categories[0].NameKey != "categoryMusic" || len(cfg.Categories[0].Extensions) == 0 {
		t.Fatalf("categories = %+v, want the migrated built-in with its default extensions", cfg.Categories)
	}
	if _, ok := cfg.Extra["downloadCategories"]; ok {
		t.Fatal("in memory config still carries the legacy key")
	}

	// The cleared key has to be persisted, not only held in memory.
	var stored base.DownloaderStoreConfig
	exist, err := storage.Get(bucketConfig, "config", &stored)
	if err != nil {
		t.Fatal(err)
	}
	if !exist {
		t.Fatal("stored config missing")
	}
	if _, ok := stored.Extra["downloadCategories"]; ok {
		t.Fatal("stored config still carries the legacy key")
	}
	if len(stored.Categories) != 1 || stored.Categories[0].NameKey != "categoryMusic" {
		t.Fatalf("stored categories = %+v, want the migrated list", stored.Categories)
	}
}

// readOnlyConfigStorage refuses every config write, so the setup can not
// persist a migrated category list.
type readOnlyConfigStorage struct {
	Storage
}

func (s *readOnlyConfigStorage) Put(bucket string, key string, v any) error {
	if bucket == bucketConfig {
		return errors.New("config storage is read only")
	}
	return s.Storage.Put(bucket, key, v)
}

func TestDownloader_SetupFailsWhenTheMigrationCanNotBePersisted(t *testing.T) {
	storage := NewMemStorage()
	if err := storage.Setup([]string{bucketConfig}); err != nil {
		t.Fatal(err)
	}
	legacy := &base.DownloaderStoreConfig{
		DownloadDir: t.TempDir(),
		Extra: map[string]any{
			"downloadCategories": []any{
				map[string]any{"name": "Music", "path": filepath.Join(t.TempDir(), "Music")},
			},
		},
	}
	if err := storage.Put(bucketConfig, "config", legacy); err != nil {
		t.Fatal(err)
	}

	// The migrated list can not be persisted, so the setup reports the write
	// error instead of continuing with a category list that is lost on restart.
	downloader := NewDownloader(&DownloaderConfig{Storage: &readOnlyConfigStorage{Storage: storage}, StorageDir: t.TempDir()})
	if err := downloader.Setup(); err == nil {
		t.Fatal("setup reported no error, want the failed config write")
	}
}

// categoryMatchingCases is the shared extension rule table of the backend and
// the ui. The ui runs the same file, see
// ui/flutter/test/api/downloader_config_test.dart, so a rule that only changes
// on one side breaks the other side's test.
type categoryMatchingCases struct {
	NormalizeExtensions []struct {
		Input    []string `json:"input"`
		Expected []string `json:"expected"`
	} `json:"normalizeExtensions"`
}

func TestSharedCategoryMatchingRules(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "category_matching_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases categoryMatchingCases
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases.NormalizeExtensions {
		got := normalizeCategoryExtensions(c.Input)
		if !slices.Equal(got, c.Expected) {
			t.Errorf("normalizeCategoryExtensions(%q) = %q, want %q", c.Input, got, c.Expected)
		}
	}
}
