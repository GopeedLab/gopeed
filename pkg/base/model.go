package base

import (
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/mattn/go-ieproxy"
	"golang.org/x/exp/slices"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Request download request
type Request struct {
	URL   string `json:"url"`
	Extra any    `json:"extra"`
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
	if r.Name == "" {
		return fmt.Errorf("invalid resource name")
	}
	if r.Files == nil || len(r.Files) == 0 {
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
	// Select file indexes to download
	SelectFiles []int `json:"selectFiles"`
	// Extra info for specific fetcher
	Extra any `json:"extra"`
}

func (o *Options) InitSelectFiles(fileSize int) {
	// if selectFiles is empty, select all files
	if len(o.SelectFiles) == 0 {
		o.SelectFiles = make([]int, fileSize)
		for i := 0; i < fileSize; i++ {
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

func ParseOptsExtra[E any](opts *Options) error {
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

	DownloadDir    string                 `json:"downloadDir"`    // DownloadDir is the default directory to save the downloaded files
	MaxRunning     int                    `json:"maxRunning"`     // MaxRunning is the max running download count
	ProtocolConfig map[string]any         `json:"protocolConfig"` // ProtocolConfig is special config for each protocol
	Extra          map[string]any         `json:"extra"`
	Proxy          *DownloaderProxyConfig `json:"proxy"`
	Webhook        *WebhookConfig         `json:"webhook"` // Webhook is the webhook configuration
	Archive        *ArchiveConfig         `json:"archive"` // Archive is the archive extraction configuration
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
	if cfg.Archive == nil {
		cfg.Archive = &ArchiveConfig{
			AutoExtract:        false,
			DeleteAfterExtract: true,
		}
	}
	return cfg
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
	if cfg.Archive == nil {
		cfg.Archive = beforeCfg.Archive
	}
	return cfg
}

// WebhookConfig is the webhook configuration
type WebhookConfig struct {
	Enable bool     `json:"enable"` // Enable is the flag to enable/disable webhooks
	URLs   []string `json:"urls"`   // URLs is the list of webhook URLs
}

// ArchiveConfig is the archive extraction configuration
type ArchiveConfig struct {
	AutoExtract        bool `json:"autoExtract"`        // AutoExtract enables automatic extraction of archives after download
	DeleteAfterExtract bool `json:"deleteAfterExtract"` // DeleteAfterExtract deletes the archive after successful extraction
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
