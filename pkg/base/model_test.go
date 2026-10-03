package base

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/util"
)

func TestDownloaderStoreConfig_Init(t *testing.T) {
	tests := []struct {
		name   string
		fields *DownloaderStoreConfig
		want   *DownloaderStoreConfig
	}{
		{
			"Init",
			&DownloaderStoreConfig{},
			&DownloaderStoreConfig{
				MaxRunning:     5,
				ProtocolConfig: map[string]any{},
				Proxy:          &DownloaderProxyConfig{},
				Webhook:        &WebhookConfig{},
				Script:         &ScriptConfig{},
				AutoTorrent: &AutoTorrentConfig{
					Enable:              false,
					DeleteAfterDownload: false,
				},
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: false,
				},
			},
		},
		{
			"Init MaxRunning",
			&DownloaderStoreConfig{
				MaxRunning: 10,
			},
			&DownloaderStoreConfig{
				MaxRunning:     10,
				ProtocolConfig: map[string]any{},
				Proxy:          &DownloaderProxyConfig{},
				Webhook:        &WebhookConfig{},
				Script:         &ScriptConfig{},
				AutoTorrent: &AutoTorrentConfig{
					Enable:              false,
					DeleteAfterDownload: false,
				},
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: false,
				},
			},
		},
		{
			"Init ProtocolConfig",
			&DownloaderStoreConfig{
				ProtocolConfig: map[string]any{
					"key": "value",
				},
			},
			&DownloaderStoreConfig{
				MaxRunning: 5,
				ProtocolConfig: map[string]any{
					"key": "value",
				},
				Proxy:   &DownloaderProxyConfig{},
				Webhook: &WebhookConfig{},
				Script:  &ScriptConfig{},
				AutoTorrent: &AutoTorrentConfig{
					Enable:              false,
					DeleteAfterDownload: false,
				},
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: false,
				},
			},
		},
		{
			"Init Proxy",
			&DownloaderStoreConfig{
				Proxy: &DownloaderProxyConfig{
					Enable: true,
				},
			},
			&DownloaderStoreConfig{
				MaxRunning:     5,
				ProtocolConfig: map[string]any{},
				Proxy: &DownloaderProxyConfig{
					Enable: true,
				},
				Webhook: &WebhookConfig{},
				Script:  &ScriptConfig{},
				AutoTorrent: &AutoTorrentConfig{
					Enable:              false,
					DeleteAfterDownload: false,
				},
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: false,
				},
			},
		},
		{
			"Init AutoTorrent",
			&DownloaderStoreConfig{
				AutoTorrent: &AutoTorrentConfig{
					Enable:              true,
					DeleteAfterDownload: true,
				},
			},
			&DownloaderStoreConfig{
				MaxRunning:     5,
				ProtocolConfig: map[string]any{},
				Proxy:          &DownloaderProxyConfig{},
				Webhook:        &WebhookConfig{},
				Script:         &ScriptConfig{},
				AutoTorrent: &AutoTorrentConfig{
					Enable:              true,
					DeleteAfterDownload: true,
				},
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: false,
				},
			},
		},
		{
			"Init Archive",
			&DownloaderStoreConfig{
				Archive: &ArchiveConfig{
					AutoExtract:        true,
					DeleteAfterExtract: false,
				},
			},
			&DownloaderStoreConfig{
				MaxRunning:     5,
				ProtocolConfig: map[string]any{},
				Proxy:          &DownloaderProxyConfig{},
				Webhook:        &WebhookConfig{},
				Script:         &ScriptConfig{},
				AutoTorrent: &AutoTorrentConfig{
					Enable:              false,
					DeleteAfterDownload: false,
				},
				Archive: &ArchiveConfig{
					AutoExtract:        true,
					DeleteAfterExtract: false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &DownloaderStoreConfig{
				FirstLoad:      tt.fields.FirstLoad,
				DownloadDir:    tt.fields.DownloadDir,
				MaxRunning:     tt.fields.MaxRunning,
				ProtocolConfig: tt.fields.ProtocolConfig,
				Extra:          tt.fields.Extra,
				Proxy:          tt.fields.Proxy,
				Webhook:        tt.fields.Webhook,
				Script:         tt.fields.Script,
				AutoTorrent:    tt.fields.AutoTorrent,
				Archive:        tt.fields.Archive,
			}
			if got := cfg.Init(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Init() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDownloaderStoreConfig_InitSeedsBuiltInCategories(t *testing.T) {
	t.Run("fresh config with a download dir gets the built-in categories", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &DownloaderStoreConfig{DownloadDir: dir}
		cfg.Init()

		wantKeys := []string{"categoryMusic", "categoryVideo", "categoryDocument", "categoryProgram"}
		wantNames := []string{"Music", "Video", "Document", "Program"}
		if len(cfg.Categories) != len(wantKeys) {
			t.Fatalf("categories = %d, want %d", len(cfg.Categories), len(wantKeys))
		}
		for i, category := range cfg.Categories {
			if category.NameKey != wantKeys[i] {
				t.Errorf("category[%d].NameKey = %q, want %q", i, category.NameKey, wantKeys[i])
			}
			if !category.IsBuiltIn {
				t.Errorf("category[%d] %q is not built-in", i, category.NameKey)
			}
			if category.Path != filepath.Join(dir, wantNames[i]) {
				t.Errorf("category[%d].Path = %q, want %q", i, category.Path, filepath.Join(dir, wantNames[i]))
			}
			if len(category.Extensions) == 0 {
				t.Errorf("category[%d] %q has no default extensions", i, category.NameKey)
			}
		}
	})

	// Init itself never reads the legacy key: Setup calls
	// MigrateLegacyExtraCategories first, and only a config that skipped the
	// migration ends up with the built-in seed.
	t.Run("legacy extra downloadCategories is not read by Init alone", func(t *testing.T) {
		cfg := &DownloaderStoreConfig{
			DownloadDir: t.TempDir(),
			Extra: map[string]any{
				"downloadCategories": []any{map[string]any{"name": "Music", "path": "/old/Music"}},
			},
		}
		cfg.Init()
		if len(cfg.Categories) != 4 {
			t.Fatalf("categories = %d, want the 4 built-ins seeded anyway", len(cfg.Categories))
		}
	})

	t.Run("existing categories and missing download dirs are untouched", func(t *testing.T) {
		existing := []*DownloadCategory{{Name: "Mine", Path: "/downloads/Mine"}}
		cfg := &DownloaderStoreConfig{DownloadDir: t.TempDir(), Categories: existing}
		cfg.Init()
		if len(cfg.Categories) != 1 || cfg.Categories[0].Name != "Mine" {
			t.Fatalf("categories = %v, want the existing one", cfg.Categories)
		}

		empty := &DownloaderStoreConfig{}
		empty.Init()
		if len(empty.Categories) != 0 {
			t.Fatalf("categories = %d, want 0 without a download dir", len(empty.Categories))
		}
	})
}

func TestDownloaderStoreConfig_AutoCategorizeEnabled(t *testing.T) {
	if !(&DownloaderStoreConfig{}).AutoCategorizeEnabled() {
		t.Fatal("unset auto categorize = disabled, want enabled by default")
	}
	if (&DownloaderStoreConfig{AutoCategorize: util.BoolPtr(false)}).AutoCategorizeEnabled() {
		t.Fatal("explicit off = enabled, want the opt-out preserved")
	}
}

func TestDownloaderStoreConfig_MigrateLegacyExtraCategories(t *testing.T) {
	t.Run("legacy built-ins move to the top level with their default extensions", func(t *testing.T) {
		cfg := &DownloaderStoreConfig{
			DownloadDir: "/downloads",
			Extra: map[string]any{
				"downloadCategories": []any{
					map[string]any{"name": "", "path": "/old/Music", "isBuiltIn": true, "nameKey": "categoryMusic"},
					map[string]any{"name": "Games", "path": "/old/Games"},
				},
				"themeMode": "dark",
			},
		}
		if !cfg.MigrateLegacyExtraCategories() {
			t.Fatal("migration reported no change")
		}
		if len(cfg.Categories) != 2 {
			t.Fatalf("categories = %d, want the 2 legacy entries", len(cfg.Categories))
		}
		builtIn := cfg.Categories[0]
		if builtIn.NameKey != "categoryMusic" || !builtIn.IsBuiltIn || builtIn.Path != "/old/Music" {
			t.Fatalf("built-in category = %+v, want the legacy entry", builtIn)
		}
		if len(builtIn.Extensions) == 0 {
			t.Fatal("migrated built-in must regain its default extensions")
		}
		custom := cfg.Categories[1]
		if custom.Name != "Games" || custom.Path != "/old/Games" || len(custom.Extensions) != 0 {
			t.Fatalf("custom category = %+v, want the legacy folder untouched", custom)
		}
		if _, ok := cfg.Extra["downloadCategories"]; ok {
			t.Fatal("legacy key must be cleared")
		}
		if cfg.Extra["themeMode"] != "dark" {
			t.Fatalf("unrelated extra entries must survive, got %v", cfg.Extra)
		}

		// Setup runs Init after the migration, which must not seed over it.
		cfg.Init()
		if len(cfg.Categories) != 2 {
			t.Fatalf("categories after Init = %d, want the migrated list kept", len(cfg.Categories))
		}
	})

	t.Run("existing top level categories are left alone", func(t *testing.T) {
		cfg := &DownloaderStoreConfig{
			Extra: map[string]any{
				"downloadCategories": []any{map[string]any{"name": "Old", "path": "/old/Default"}},
			},
			Categories: []*DownloadCategory{{Name: "Mine", Path: "/downloads/Mine"}},
		}
		if cfg.MigrateLegacyExtraCategories() {
			t.Fatal("migration must not overwrite existing categories")
		}
		if len(cfg.Categories) != 1 || cfg.Categories[0].Name != "Mine" {
			t.Fatalf("categories = %v, want the existing one", cfg.Categories)
		}
		if _, ok := cfg.Extra["downloadCategories"]; !ok {
			t.Fatal("legacy key must stay for a manual move")
		}
	})

	t.Run("missing, empty and malformed legacy lists are a no-op", func(t *testing.T) {
		if (&DownloaderStoreConfig{}).MigrateLegacyExtraCategories() {
			t.Fatal("empty config reported a change")
		}
		empty := &DownloaderStoreConfig{
			Extra: map[string]any{"downloadCategories": []any{}},
		}
		if empty.MigrateLegacyExtraCategories() {
			t.Fatal("empty legacy list reported a change")
		}
		malformed := &DownloaderStoreConfig{
			Extra: map[string]any{"downloadCategories": "not-a-list"},
		}
		if malformed.MigrateLegacyExtraCategories() {
			t.Fatal("malformed legacy list reported a change")
		}
		if _, ok := malformed.Extra["downloadCategories"]; !ok {
			t.Fatal("malformed legacy key must stay untouched")
		}
	})
}

func TestDownloaderStoreConfig_AutoCategorizeJSON(t *testing.T) {
	unset, err := json.Marshal(&DownloaderStoreConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(unset), `"autoCategorize"`) {
		t.Fatalf("unset autoCategorize must be omitted from the api json, got %s", unset)
	}
	off, err := json.Marshal(&DownloaderStoreConfig{AutoCategorize: util.BoolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(off), `"autoCategorize":false`) {
		t.Fatalf("explicit autoCategorize=false must be emitted, got %s", off)
	}
}

func TestDownloaderStoreConfig_Merge(t *testing.T) {
	type args struct {
		beforeCfg *DownloaderStoreConfig
	}
	tests := []struct {
		name   string
		fields *DownloaderStoreConfig
		args   args
		want   *DownloaderStoreConfig
	}{
		{
			"Merge Nil",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: nil,
			},
			&DownloaderStoreConfig{},
		},
		{
			"Merge DownloadDir No Override",
			&DownloaderStoreConfig{
				DownloadDir: "before",
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					DownloadDir: "after",
				},
			},
			&DownloaderStoreConfig{
				DownloadDir: "before",
			},
		},
		{
			"Merge DownloadDir Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					DownloadDir: "after",
				},
			},
			&DownloaderStoreConfig{
				DownloadDir: "after",
			},
		},
		{
			"Merge MaxRunning No Override",
			&DownloaderStoreConfig{
				MaxRunning: 1,
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					MaxRunning: 10,
				},
			},
			&DownloaderStoreConfig{
				MaxRunning: 1,
			},
		},
		{
			"Merge MaxRunning Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					MaxRunning: 10,
				},
			},
			&DownloaderStoreConfig{
				MaxRunning: 10,
			},
		},
		{
			"Merge ProtocolConfig No Override",
			&DownloaderStoreConfig{
				ProtocolConfig: map[string]any{},
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					ProtocolConfig: map[string]any{
						"key": "after",
					},
				},
			},
			&DownloaderStoreConfig{
				ProtocolConfig: map[string]any{},
			},
		},
		{
			"Merge ProtocolConfig Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					ProtocolConfig: map[string]any{
						"key": "after",
					},
				},
			},
			&DownloaderStoreConfig{
				ProtocolConfig: map[string]any{
					"key": "after",
				},
			},
		},
		{
			"Merge Extra No Override",
			&DownloaderStoreConfig{
				Extra: map[string]any{},
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					Extra: map[string]any{
						"key": "after",
					},
				},
			},
			&DownloaderStoreConfig{
				Extra: map[string]any{},
			},
		},
		{
			"Merge Extra Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					Extra: map[string]any{
						"key": "after",
					},
				},
			},
			&DownloaderStoreConfig{
				Extra: map[string]any{
					"key": "after",
				},
			},
		},
		{
			"Merge Proxy No Override",
			&DownloaderStoreConfig{
				Proxy: &DownloaderProxyConfig{},
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					Proxy: &DownloaderProxyConfig{
						Scheme: "http",
					},
				},
			},
			&DownloaderStoreConfig{
				Proxy: &DownloaderProxyConfig{},
			},
		},
		{
			"Merge Proxy Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					Proxy: &DownloaderProxyConfig{
						Scheme: "http",
					},
				},
			},
			&DownloaderStoreConfig{
				Proxy: &DownloaderProxyConfig{
					Scheme: "http",
				},
			},
		},
		{
			"Merge AutoTorrent No Override",
			&DownloaderStoreConfig{
				AutoTorrent: &AutoTorrentConfig{
					Enable: true,
				},
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					AutoTorrent: &AutoTorrentConfig{
						Enable:              false,
						DeleteAfterDownload: false,
					},
				},
			},
			&DownloaderStoreConfig{
				AutoTorrent: &AutoTorrentConfig{
					Enable: true,
				},
			},
		},
		{
			"Merge AutoTorrent Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					AutoTorrent: &AutoTorrentConfig{
						Enable:              true,
						DeleteAfterDownload: true,
					},
				},
			},
			&DownloaderStoreConfig{
				AutoTorrent: &AutoTorrentConfig{
					Enable:              true,
					DeleteAfterDownload: true,
				},
			},
		},
		{
			"Merge Archive No Override",
			&DownloaderStoreConfig{
				Archive: &ArchiveConfig{
					AutoExtract: true,
				},
			},
			args{
				beforeCfg: &DownloaderStoreConfig{
					Archive: &ArchiveConfig{
						AutoExtract:        false,
						DeleteAfterExtract: false,
					},
				},
			},
			&DownloaderStoreConfig{
				Archive: &ArchiveConfig{
					AutoExtract: true,
				},
			},
		},
		{
			"Merge Archive Override",
			&DownloaderStoreConfig{},
			args{
				beforeCfg: &DownloaderStoreConfig{
					Archive: &ArchiveConfig{
						AutoExtract:        false,
						DeleteAfterExtract: false,
					},
				},
			},
			&DownloaderStoreConfig{
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &DownloaderStoreConfig{
				FirstLoad:      tt.fields.FirstLoad,
				DownloadDir:    tt.fields.DownloadDir,
				MaxRunning:     tt.fields.MaxRunning,
				ProtocolConfig: tt.fields.ProtocolConfig,
				Extra:          tt.fields.Extra,
				Proxy:          tt.fields.Proxy,
				Webhook:        tt.fields.Webhook,
				AutoTorrent:    tt.fields.AutoTorrent,
				Archive:        tt.fields.Archive,
			}
			if got := cfg.Merge(tt.args.beforeCfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
