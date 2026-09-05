package model

import (
	"errors"

	"github.com/GopeedLab/gopeed/pkg/base"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
	"io/fs"
)

type Storage string

const (
	StorageMem  Storage = "mem"
	StorageBolt Storage = "bolt"
)

type StartConfig struct {
	Network           string                      `json:"network"`
	Address           string                      `json:"address"`
	ApiEnable         *bool                       `json:"apiEnable,omitempty"`
	MCPEnable         bool                        `json:"mcpEnable"`
	RefreshInterval   int                         `json:"refreshInterval"`
	Storage           Storage                     `json:"storage"`
	StorageDir        string                      `json:"storageDir"`
	WhiteDownloadDirs []string                    `json:"whiteDownloadDirs"`
	ApiToken          string                      `json:"apiToken"`
	DownloadConfig    *base.DownloaderStoreConfig `json:"downloadConfig"`
	WebViewRPCConfig  *enginewebview.RPCConfig    `json:"webViewRpcConfig,omitempty"`

	ProductionMode  bool
	NativeMode      bool
	WebViewProvider enginewebview.Provider `json:"-"`

	WebEnable bool
	WebFS     fs.FS
	WebAuth   *WebAuth
}

func (cfg *StartConfig) APIEnabled() bool {
	if cfg == nil || cfg.WebEnable {
		return true
	}
	// Preserve the behavior of existing CLI and library callers. Native
	// Flutter explicitly sends false when it wants an in-process-only runtime.
	return cfg.ApiEnable == nil || *cfg.ApiEnable
}

func (cfg *StartConfig) Validate() error {
	if cfg == nil {
		return nil
	}
	if cfg.WebEnable && cfg.ApiToken != "" &&
		(cfg.WebAuth == nil || cfg.WebAuth.Username == "" || cfg.WebAuth.Password == "") {
		return errors.New("web authentication username and password are required when API token is enabled for the web server")
	}
	return nil
}

func (cfg *StartConfig) Init() *StartConfig {
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:0"
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 350
	}
	if cfg.Storage == "" {
		cfg.Storage = StorageBolt
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./"
	}
	return cfg
}

type WebAuth struct {
	Username string
	Password string
}

// APIServerState describes the listener that is actually running. The
// persisted APIServerConfig remains the desired state.
type APIServerState struct {
	Enabled      bool   `json:"enabled"`
	MCPEnabled   bool   `json:"mcpEnabled"`
	Running      bool   `json:"running"`
	Network      string `json:"network"`
	Address      string `json:"address"`
	RunningPort  int    `json:"runningPort"`
	PendingApply bool   `json:"pendingApply"`
	LastError    string `json:"lastError"`
}

type APIServerOperationResult struct {
	State *APIServerState `json:"state"`
	Error string          `json:"error"`
}
