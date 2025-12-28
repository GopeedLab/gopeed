package base

import (
	"reflect"
	"testing"
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
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: true,
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
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: true,
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
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: true,
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
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: true,
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
				Archive:        tt.fields.Archive,
			}
			if got := cfg.Init(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Init() = %v, want %v", got, tt.want)
			}
		})
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
						DeleteAfterExtract: true,
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
						DeleteAfterExtract: true,
					},
				},
			},
			&DownloaderStoreConfig{
				Archive: &ArchiveConfig{
					AutoExtract:        false,
					DeleteAfterExtract: true,
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
				Archive:        tt.fields.Archive,
			}
			if got := cfg.Merge(tt.args.beforeCfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
