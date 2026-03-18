package ed2k

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	ped2k "github.com/GopeedLab/gopeed/pkg/protocol/ed2k"
	"github.com/monkeyWie/goed2k"
	gprotocol "github.com/monkeyWie/goed2k/protocol"
)

type clientStateStore struct {
	store fetcher.ProtocolStateStore
}

func (s *clientStateStore) Load() (*goed2k.ClientState, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	var state goed2k.ClientState
	exist, err := s.store.Load(&state)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return &state, nil
}

func (s *clientStateStore) Save(state *goed2k.ClientState) error {
	if s == nil || s.store == nil {
		return nil
	}
	if state == nil {
		return s.store.Delete()
	}
	return s.store.Save(state)
}

type Fetcher struct {
	ctl    *controller.Controller
	config *config

	manager *FetcherManager
	meta    *fetcher.FetcherMeta
	handle  goed2k.TransferHandle

	waitCtx    context.Context
	waitCancel context.CancelFunc
}

func (f *Fetcher) Setup(ctl *controller.Controller) {
	f.ctl = ctl
	if f.meta == nil {
		f.meta = &fetcher.FetcherMeta{}
	}
	f.waitCtx, f.waitCancel = context.WithCancel(context.Background())
	f.ctl.GetConfig(&f.config)
}

func (f *Fetcher) Resolve(req *base.Request, opts *base.Options) error {
	link, err := parseLink(req.URL)
	if err != nil {
		return err
	}

	f.meta.Req = req
	f.meta.Opts = opts
	if f.meta.Opts == nil {
		f.meta.Opts = &base.Options{}
	}
	f.meta.Res = buildResource(link)
	return nil
}

func (f *Fetcher) Start() error {
	link, err := parseLink(f.meta.Req.URL)
	if err != nil {
		return err
	}
	if f.meta.Res == nil {
		f.meta.Res = buildResource(link)
	}

	client, err := f.getClient()
	if err != nil {
		return err
	}

	targetPath := f.meta.SingleFilepath()
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	handle := client.FindTransfer(link.Hash)
	if handle.IsValid() {
		f.handle = handle
		if handle.IsPaused() {
			if err := client.ResumeTransfer(handle.GetHash()); err != nil {
				return err
			}
		}
		return nil
	}

	atp := goed2k.AddTransferParams{
		Hash:       link.Hash,
		CreateTime: time.Now().UnixMilli(),
		Size:       link.NumberValue,
		FilePath:   targetPath,
	}
	handle, err = client.AddTransfer(atp)
	if err != nil {
		return err
	}
	f.handle = handle
	if handle.IsValid() && handle.IsPaused() {
		if err := client.ResumeTransfer(handle.GetHash()); err != nil {
			return err
		}
	}
	return nil
}

func (f *Fetcher) Patch(req *base.Request, opts *base.Options) error {
	handle := f.currentHandle()

	if opts != nil && (opts.Name != "" || opts.Path != "") && handle.IsValid() {
		return errors.New("cannot change ed2k target path after transfer started")
	}
	if req != nil && req.URL != "" && handle.IsValid() {
		return errors.New("cannot change ed2k link after transfer started")
	}

	if req != nil {
		if req.URL != "" {
			link, err := parseLink(req.URL)
			if err != nil {
				return err
			}
			f.meta.Req.URL = req.URL
			f.meta.Res = buildResource(link)
		}
		if req.Labels != nil {
			if f.meta.Req.Labels == nil {
				f.meta.Req.Labels = make(map[string]string)
			}
			for k, v := range req.Labels {
				f.meta.Req.Labels[k] = v
			}
		}
		if req.Proxy != nil {
			f.meta.Req.Proxy = req.Proxy
		}
	}

	if opts != nil {
		if opts.Name != "" {
			f.meta.Opts.Name = opts.Name
		}
		if opts.Path != "" {
			f.meta.Opts.Path = opts.Path
		}
	}
	return nil
}

func (f *Fetcher) Pause() error {
	handle := f.currentHandle()
	if !handle.IsValid() {
		return nil
	}
	client, err := f.getClient()
	if err != nil {
		return err
	}
	if err := client.PauseTransfer(handle.GetHash()); err != nil {
		return err
	}
	f.handle = handle
	return nil
}

func (f *Fetcher) Close() error {
	if f.waitCancel != nil {
		f.waitCancel()
	}
	handle := f.currentHandle()
	if !handle.IsValid() {
		return nil
	}
	client, err := f.getClient()
	if err != nil {
		return err
	}
	f.handle = handle
	return client.RemoveTransfer(handle.GetHash(), false)
}

func (f *Fetcher) Meta() *fetcher.FetcherMeta {
	return f.meta
}

func (f *Fetcher) Stats() any {
	handle := f.currentHandle()
	if !handle.IsValid() {
		return &ped2k.Stats{}
	}
	status := handle.GetStatus()
	return &ped2k.Stats{
		State:         string(status.State),
		ActivePeers:   handle.ActiveConnections(),
		TotalPeers:    status.NumPeers,
		DownloadRate:  status.DownloadRate,
		Upload:        status.Upload,
		UploadRate:    status.UploadRate,
		TotalDone:     status.TotalDone,
		TotalReceived: status.TotalReceived,
		TotalWanted:   status.TotalWanted,
	}
}

func (f *Fetcher) Progress() fetcher.Progress {
	handle := f.currentHandle()
	if !handle.IsValid() {
		return fetcher.Progress{0}
	}
	return fetcher.Progress{handle.GetStatus().TotalReceived}
}

func (f *Fetcher) Wait() error {
	client, err := f.getClient()
	if err != nil {
		return err
	}

	hash, err := f.hash()
	if err != nil {
		return err
	}

	handle := f.currentHandle()
	if handle.IsValid() && handle.IsFinished() {
		return nil
	}

	progressCh, cancel := client.SubscribeTransferProgress()
	defer cancel()

	for {
		select {
		case <-f.waitCtx.Done():
			return nil
		case event, ok := <-progressCh:
			if !ok {
				return nil
			}
			for _, transfer := range event.Transfers {
				if transfer.Hash.Compare(hash) != 0 {
					continue
				}
				// Removal can happen during task deletion or client shutdown, both of
				// which should unblock Wait without treating it as a download failure.
				if transfer.Removed || transfer.State == goed2k.Finished {
					return nil
				}
			}
		}
	}
}

func (f *Fetcher) getClient() (*goed2k.Client, error) {
	if f.manager == nil {
		f.manager = &FetcherManager{}
	}
	return f.manager.initClient(f.config)
}

func (f *Fetcher) currentHandle() goed2k.TransferHandle {
	if f.handle.IsValid() {
		return f.handle
	}
	if f.manager == nil {
		return f.handle
	}
	client := f.manager.currentClient()
	if client == nil {
		return f.handle
	}
	hash, err := f.hash()
	if err != nil {
		return f.handle
	}
	handle := client.FindTransfer(hash)
	if handle.IsValid() {
		f.handle = handle
	}
	return f.handle
}

func (f *Fetcher) hash() (gprotocol.Hash, error) {
	if f.meta == nil || f.meta.Req == nil {
		return gprotocol.Invalid, errors.New("ed2k link is empty")
	}
	link, err := parseLink(f.meta.Req.URL)
	if err != nil {
		return gprotocol.Invalid, err
	}
	return link.Hash, nil
}

type FetcherManager struct {
	mu         sync.Mutex
	client     *goed2k.Client
	stateStore *clientStateStore
}

func (fm *FetcherManager) SetStateStore(store fetcher.ProtocolStateStore) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.stateStore == nil {
		fm.stateStore = &clientStateStore{}
	}
	fm.stateStore.store = store
	if fm.client != nil {
		fm.client.SetStateStore(fm.stateStore)
	}
}

func (fm *FetcherManager) Name() string {
	return "ed2k"
}

func (fm *FetcherManager) Filters() []*fetcher.SchemeFilter {
	return []*fetcher.SchemeFilter{
		{
			Type:    fetcher.FilterTypeUrl,
			Pattern: "ED2K",
		},
	}
}

func (fm *FetcherManager) Build() fetcher.Fetcher {
	return &Fetcher{manager: fm}
}

func (fm *FetcherManager) ParseName(u string) string {
	link, err := parseLink(u)
	if err != nil {
		return ""
	}
	return link.StringValue
}

func (fm *FetcherManager) AutoRename() bool {
	return true
}

func (fm *FetcherManager) DefaultConfig() any {
	return &config{
		ListenPort: 0,
		UDPPort:    0,
		ServerAddr: defaultServerList,
		ServerMet:  defaultServerMet,
		NodesDat:   defaultNodesDat,
	}
}

func (fm *FetcherManager) Store(f fetcher.Fetcher) (any, error) {
	return nil, nil
}

func (fm *FetcherManager) Restore() (v any, f func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher) {
	return nil, func(meta *fetcher.FetcherMeta, v any) fetcher.Fetcher {
		return &Fetcher{
			manager: fm,
			meta:    meta,
		}
	}
}

func (fm *FetcherManager) Close() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.client != nil {
		fm.client.Close()
		fm.client = nil
	}
	return nil
}

func parseLink(raw string) (goed2k.EMuleLink, error) {
	link, err := goed2k.ParseEMuleLink(raw)
	if err != nil {
		return goed2k.EMuleLink{}, err
	}
	if link.Type != goed2k.LinkFile {
		return goed2k.EMuleLink{}, errors.New("unsupported ed2k link type")
	}
	return link, nil
}

func buildResource(link goed2k.EMuleLink) *base.Resource {
	return &base.Resource{
		Size:  link.NumberValue,
		Range: false,
		Hash:  link.Hash.String(),
		Files: []*base.FileInfo{
			{
				Name: link.StringValue,
				Size: link.NumberValue,
			},
		},
	}
}

func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (fm *FetcherManager) getStateStoreLocked() *clientStateStore {
	if fm.stateStore == nil {
		fm.stateStore = &clientStateStore{}
	}
	return fm.stateStore
}

func (fm *FetcherManager) currentClient() *goed2k.Client {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.client
}

func (fm *FetcherManager) initClient(cfg *config) (*goed2k.Client, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.client != nil {
		return fm.client, nil
	}
	if cfg == nil {
		cfg = fm.DefaultConfig().(*config)
	}

	settings := goed2k.NewSettings()
	settings.ListenPort = cfg.ListenPort
	settings.UDPPort = cfg.UDPPort
	settings.EnableDHT = true
	settings.EnableUPnP = true
	settings.ReconnectToServer = true

	client := goed2k.NewClient(settings)
	client.SetStateStore(fm.getStateStoreLocked())
	if err := client.LoadState(""); err != nil {
		return nil, err
	}
	if err := client.Start(); err != nil {
		return nil, err
	}
	fm.client = client
	// Bootstrap is best-effort: downloads can still proceed later even if
	// server list or DHT initialization fails during startup.
	bootstrapClient(client, cfg)
	return fm.client, nil
}

func bootstrapClient(client *goed2k.Client, cfg *config) {
	for _, serverAddr := range splitCommaList(cfg.ServerAddr) {
		go func(serverAddr string) {
			_ = client.Connect(serverAddr)
		}(serverAddr)
	}
	for _, source := range splitCommaList(cfg.ServerMet) {
		go func(source string) {
			_ = client.ConnectServerMet(source)
		}(source)
	}
	if cfg.NodesDat != "" {
		go func() {
			_ = client.LoadDHTNodesDat(cfg.NodesDat)
		}()
	}
}
