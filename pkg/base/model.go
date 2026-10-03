package base

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/mattn/go-ieproxy"
	"golang.org/x/exp/slices"
)

// Request download request
type Request struct {
	RawURL string `json:"rawUrl"`
	URL    string `json:"url"`
	Extra  any    `json:"extra"`
	// Labels is used to mark the download task
	Labels map[string]string `json:"labels"`
	// Proxy is special proxy config for request
	Proxy *RequestProxy `json:"proxy"`
	// SkipVerifyCert is the flag that skip verify cert
	SkipVerifyCert bool `json:"skipVerifyCert"`
}

func (r *Request) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("invalid request url")
	}
	return nil
}

type RequestProxyMode string

const (
	// RequestProxyModeFollow follow setting proxy
	RequestProxyModeFollow RequestProxyMode = "follow"
	// RequestProxyModeNone not use proxy
	RequestProxyModeNone RequestProxyMode = "none"
	// RequestProxyModeCustom custom proxy
	RequestProxyModeCustom RequestProxyMode = "custom"
)

type RequestProxy struct {
	Mode   RequestProxyMode `json:"mode"`
	Scheme string           `json:"scheme"`
	Host   string           `json:"host"`
	Usr    string           `json:"usr"`
	Pwd    string           `json:"pwd"`
}

func (p *RequestProxy) ToHandler() func(r *http.Request) (*url.URL, error) {
	if p == nil || p.Mode != RequestProxyModeCustom {
		return nil
	}

	if p.Scheme == "" || p.Host == "" {
		return nil
	}

	return http.ProxyURL(util.BuildProxyUrl(p.Scheme, p.Host, p.Usr, p.Pwd))
}

// Resource download resource
type Resource struct {
	// if name is not empty, the resource is a folder and the name is the folder name
	Name string `json:"name"`
	Size int64  `json:"size"`
	// is support range download
	Range bool `json:"range"`
	// file list
	Files []*FileInfo `json:"files"`
	Hash  string      `json:"hash"`
}

func (r *Resource) Validate() error {
	if len(r.Files) == 0 {
		return fmt.Errorf("invalid resource files")
	}
	for _, file := range r.Files {
		if file.Name == "" {
			return fmt.Errorf("invalid resource file name")
		}
	}
	return nil
}

func (r *Resource) CalcSize(selectFiles []int) {
	var size int64
	for i, file := range r.Files {
		if len(selectFiles) == 0 || slices.Contains(selectFiles, i) {
			size += file.Size
		}
	}
	r.Size = size
}

type FileInfo struct {
	Name  string     `json:"name"`
	Path  string     `json:"path"`
	Size  int64      `json:"size"`
	Ctime *time.Time `json:"ctime"`

	Req *Request `json:"req"`
}

// Options for download
type Options struct {
	// Download file name
	Name string `json:"name"`
	// Download file path
	Path string `json:"path"`
	// Use this path as the default download directory after creation succeeds
	AsDefaultPath bool `json:"asDefaultPath,omitempty"`
	// Select file indexes to download
	SelectFiles []int `json:"selectFiles"`
	// Extra info for specific fetcher
	Extra any `json:"extra"`
}

func (o *Options) InitSelectFiles(fileSize int) {
	// if selectFiles is empty, select all files
	if len(o.SelectFiles) == 0 {
		o.SelectFiles = make([]int, fileSize)
		for i := range fileSize {
			o.SelectFiles[i] = i
		}
	}
}

func (o *Options) Clone() *Options {
	return util.DeepClone(o)
}

func ParseReqExtra[E any](req *Request) error {
	if req.Extra == nil {
		return nil
	}
	if _, ok := req.Extra.(*E); ok {
		return nil
	}
	var t E
	if err := util.MapToStruct(req.Extra, &t); err != nil {
		return err
	}
	req.Extra = &t
	return nil
}

func ParseOptExtra[E any](opts *Options) error {
	if opts.Extra == nil {
		return nil
	}
	if _, ok := opts.Extra.(*E); ok {
		return nil
	}
	var t E
	if err := util.MapToStruct(opts.Extra, &t); err != nil {
		return err
	}
	opts.Extra = &t
	return nil
}

type CreateTaskBatch struct {
	Reqs []*CreateTaskBatchItem `json:"reqs"`
	Opts *Options               `json:"opts"`
}

type CreateTaskBatchItem struct {
	Req  *Request `json:"req"`
	Opts *Options `json:"opts"`
}

// DownloaderStoreConfig is the config that can restore the downloader.
type DownloaderStoreConfig struct {
	FirstLoad bool `json:"-"` // FirstLoad is the flag that the config is first time init and not from store

	DownloadDir                string                 `json:"downloadDir"`    // DownloadDir is the default directory to save the downloaded files
	MaxRunning                 int                    `json:"maxRunning"`     // MaxRunning is the max running download count
	ProtocolConfig             map[string]any         `json:"protocolConfig"` // ProtocolConfig is special config for each protocol
	Extra                      map[string]any         `json:"extra"`
	Proxy                      *DownloaderProxyConfig `json:"proxy"`
	Webhook                    *WebhookConfig         `json:"webhook"`                    // Webhook is the webhook configuration
	Script                     *ScriptConfig          `json:"script"`                     // Script is the script execution configuration
	AutoTorrent                *AutoTorrentConfig     `json:"autoTorrent"`                // AutoTorrent is the auto torrent task creation configuration
	Archive                    *ArchiveConfig         `json:"archive"`                    // Archive is the archive extraction configuration
	Categories                 []*DownloadCategory    `json:"categories"`                 // Categories is the download directory category configuration
	AutoCategorize             *bool                  `json:"autoCategorize,omitempty"`   // AutoCategorize routes downloads to category directories by file extension; unset defaults to enabled
	API                        *APIServerConfig       `json:"api"`                        // API is the optional REST server configuration
	AutoStartTasks             bool                   `json:"autoStartTasks"`             // AutoStartTasks continues all unfinished tasks when the backend starts
	AutoDeleteMissingFileTasks bool                   `json:"autoDeleteMissingFileTasks"` // AutoDeleteMissingFileTasks enables automatic deletion of tasks with missing files
}

func (cfg *DownloaderStoreConfig) Init() *DownloaderStoreConfig {
	if cfg.MaxRunning == 0 {
		cfg.MaxRunning = 5
	}
	if cfg.ProtocolConfig == nil {
		cfg.ProtocolConfig = make(map[string]any)
	}
	if cfg.Proxy == nil {
		cfg.Proxy = &DownloaderProxyConfig{}
	}
	if cfg.Webhook == nil {
		cfg.Webhook = &WebhookConfig{}
	}
	if cfg.Script == nil {
		cfg.Script = &ScriptConfig{}
	}
	if cfg.AutoTorrent == nil {
		cfg.AutoTorrent = &AutoTorrentConfig{
			Enable:              false,
			DeleteAfterDownload: false,
		}
	}
	if cfg.Archive == nil {
		cfg.Archive = &ArchiveConfig{
			AutoExtract:        false,
			DeleteAfterExtract: false,
		}
	}
	// Seed the built-in categories for deployments that never run the flutter
	// startup flow, e.g. headless api servers.
	if len(cfg.Categories) == 0 && cfg.DownloadDir != "" {
		cfg.Categories = []*DownloadCategory{
			builtinSeedCategory("categoryMusic", "Music", cfg.DownloadDir),
			builtinSeedCategory("categoryVideo", "Video", cfg.DownloadDir),
			builtinSeedCategory("categoryDocument", "Document", cfg.DownloadDir),
			builtinSeedCategory("categoryProgram", "Program", cfg.DownloadDir),
		}
	}
	return cfg
}

// builtinCategoryExtensions mirrors the flutter side
// kDefaultCategoryExtensions, keep both maps in sync.
var builtinCategoryExtensions = map[string][]string{
	"categoryProgram":  {"exe", "msi", "msix", "apk", "dmg", "deb", "rpm", "pkg", "appimage"},
	"categoryVideo":    {"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm", "m4v", "mpg", "mpeg", "3gp"},
	"categoryMusic":    {"mp3", "flac", "wav", "aac", "ogg", "m4a", "wma", "ape"},
	"categoryDocument": {"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "md", "csv", "epub", "rtf"},
}

func builtinSeedCategory(nameKey, name, downloadDir string) *DownloadCategory {
	return &DownloadCategory{
		Name:       "",
		Path:       filepath.Join(downloadDir, name),
		NameKey:    nameKey,
		IsBuiltIn:  true,
		Extensions: append([]string(nil), builtinCategoryExtensions[nameKey]...),
	}
}

// MigrateLegacyExtraCategories moves the extra.downloadCategories list written
// by v2.0.0-beta builds into the top level categories field and clears the
// legacy key. The legacy list predates extension based routing, so built-in
// entries regain their default extension lists from their name key, while user
// created folders keep their empty list until edited in settings. It reports
// whether the config changed so the caller can persist the cleared key.
func (cfg *DownloaderStoreConfig) MigrateLegacyExtraCategories() bool {
	if len(cfg.Categories) > 0 || cfg.Extra == nil {
		return false
	}
	raw, ok := cfg.Extra["downloadCategories"]
	if !ok {
		return false
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return false
	}
	var legacy []*DownloadCategory
	if err := json.Unmarshal(data, &legacy); err != nil {
		return false
	}
	categories := make([]*DownloadCategory, 0, len(legacy))
	for _, category := range legacy {
		if category == nil {
			continue
		}
		if extensions, exist := builtinCategoryExtensions[category.NameKey]; exist && len(category.Extensions) == 0 {
			category.Extensions = append([]string(nil), extensions...)
		}
		categories = append(categories, category)
	}
	if len(categories) == 0 {
		return false
	}
	cfg.Categories = categories
	delete(cfg.Extra, "downloadCategories")
	return true
}

// AutoCategorizeEnabled reports whether downloads are routed to category
// directories by file extension. The switch is optional: an unset value
// defaults to enabled so new installs route without extra configuration.
func (cfg *DownloaderStoreConfig) AutoCategorizeEnabled() bool {
	return cfg.AutoCategorize == nil || *cfg.AutoCategorize
}

func (cfg *DownloaderStoreConfig) Merge(beforeCfg *DownloaderStoreConfig) *DownloaderStoreConfig {
	if beforeCfg == nil {
		return cfg
	}
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = beforeCfg.DownloadDir
	}
	if cfg.MaxRunning == 0 {
		cfg.MaxRunning = beforeCfg.MaxRunning
	}
	if cfg.ProtocolConfig == nil {
		cfg.ProtocolConfig = beforeCfg.ProtocolConfig
	}
	if cfg.Extra == nil {
		cfg.Extra = beforeCfg.Extra
	}
	if cfg.Proxy == nil {
		cfg.Proxy = beforeCfg.Proxy
	}
	if cfg.Webhook == nil {
		cfg.Webhook = beforeCfg.Webhook
	}
	if cfg.Script == nil {
		cfg.Script = beforeCfg.Script
	}
	if cfg.AutoTorrent == nil {
		cfg.AutoTorrent = beforeCfg.AutoTorrent
	}
	if cfg.Archive == nil {
		cfg.Archive = beforeCfg.Archive
	}
	if cfg.API == nil {
		cfg.API = beforeCfg.API
	}
	return cfg
}

type APIServerConfig struct {
	Enable    bool   `json:"enable"`
	MCPEnable bool   `json:"mcpEnable"`
	Network   string `json:"network"`
	Address   string `json:"address"`
	Token     string `json:"token"`
}

func (cfg *APIServerConfig) Init() *APIServerConfig {
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:9999"
	}
	return cfg
}

// WebhookConfig is the webhook configuration
type WebhookConfig struct {
	Enable bool     `json:"enable"` // Enable is the flag to enable/disable webhooks
	URLs   []string `json:"urls"`   // URLs is the list of webhook URLs
}

// ScriptConfig is the script execution configuration
type ScriptConfig struct {
	Enable bool     `json:"enable"` // Enable is the flag to enable/disable script execution
	Paths  []string `json:"paths"`  // Paths is the list of script paths to execute
}

// AutoTorrentConfig is the auto torrent task creation configuration
type AutoTorrentConfig struct {
	Enable              bool `json:"enable"`              // Enable enables automatic BT task creation when downloading .torrent files
	DeleteAfterDownload bool `json:"deleteAfterDownload"` // DeleteAfterDownload deletes the .torrent file after BT task creation
}

// ArchiveConfig is the archive extraction configuration
type ArchiveConfig struct {
	AutoExtract        bool `json:"autoExtract"`        // AutoExtract enables automatic extraction of archives after download
	DeleteAfterExtract bool `json:"deleteAfterExtract"` // DeleteAfterExtract deletes the archive after successful extraction
}

// DownloadCategory is a download directory category that files can be routed to by extension
type DownloadCategory struct {
	Name       string   `json:"name"`                 // Name is the category display name, empty for built-in categories that use NameKey
	Path       string   `json:"path"`                 // Path is the download directory of the category
	NameKey    string   `json:"nameKey,omitempty"`    // NameKey is the i18n key of the built-in category name
	IsBuiltIn  bool     `json:"isBuiltIn,omitempty"`  // IsBuiltIn marks the category as built-in
	IsDeleted  bool     `json:"isDeleted,omitempty"`  // IsDeleted marks the built-in category as deleted
	Extensions []string `json:"extensions,omitempty"` // Extensions is the file extension list routed to the category, without leading dots
}

type DownloaderProxyConfig struct {
	Enable bool `json:"enable"`
	// System is the flag that use system proxy
	System bool   `json:"system"`
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Usr    string `json:"usr"`
	Pwd    string `json:"pwd"`
}

func (cfg *DownloaderProxyConfig) ToHandler() func(r *http.Request) (*url.URL, error) {
	if cfg == nil || cfg.Enable == false {
		return nil
	}
	if cfg.System {
		safeProxyReloadConf()
		return ieproxy.GetProxyFunc()
	}
	if cfg.Scheme == "" || cfg.Host == "" {
		return nil
	}
	return http.ProxyURL(util.BuildProxyUrl(cfg.Scheme, cfg.Host, cfg.Usr, cfg.Pwd))
}

// ToUrl returns the proxy url, just for git clone
func (cfg *DownloaderProxyConfig) ToUrl() *url.URL {
	if cfg == nil || cfg.Enable == false {
		return nil
	}
	if cfg.System {
		safeProxyReloadConf()
		static := ieproxy.GetConf().Static
		if static.Active && len(static.Protocols) > 0 {
			// If only one protocol, use it
			if len(static.Protocols) == 1 {
				for _, v := range static.Protocols {
					return parseUrlSafe(v)
				}
			}
			// Check https
			if v, ok := static.Protocols["https"]; ok {
				return parseUrlSafe(v)
			}
			// Check http
			if v, ok := static.Protocols["http"]; ok {
				return parseUrlSafe(v)
			}
		}
		return nil
	}
	if cfg.Scheme == "" || cfg.Host == "" {
		return nil
	}
	return util.BuildProxyUrl(cfg.Scheme, cfg.Host, cfg.Usr, cfg.Pwd)
}

var prcLock sync.Mutex

func safeProxyReloadConf() {
	prcLock.Lock()
	defer prcLock.Unlock()

	ieproxy.ReloadConf()
}

func parseUrlSafe(rawUrl string) *url.URL {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil
	}
	return u
}
