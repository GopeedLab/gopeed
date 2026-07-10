package download

import (
	"errors"
	"fmt"
	"math"
	gohttp "net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	internalblob "github.com/GopeedLab/gopeed/internal/blob"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/logger"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const (
	// task info bucket
	bucketTask = "task"
	// task download data bucket
	bucketSave = "save"
	// protocol-level shared client state bucket
	bucketProtocolState = "protocol_state"
	// downloader config bucket
	bucketConfig = "config"
	// downloader extension bucket
	bucketExtension = "extension"
	// downloader extension storage bucket
	bucketExtensionStorage = "extension_storage"
)

var (
	ErrTaskNotFound        = errors.New("task not found")
	ErrUnSupportedProtocol = errors.New("unsupported protocol")
)

type Listener func(event *Event)

// ExtractStatus represents the current status of archive extraction
type ExtractStatus string

const (
	// ExtractStatusNone indicates extraction has not started
	ExtractStatusNone ExtractStatus = ""
	// ExtractStatusQueued indicates extraction is waiting in the queue
	ExtractStatusQueued ExtractStatus = "queued"
	// ExtractStatusWaitingParts indicates waiting for other multi-part archive parts to complete
	ExtractStatusWaitingParts ExtractStatus = "waitingParts"
	// ExtractStatusExtracting indicates extraction is in progress
	ExtractStatusExtracting ExtractStatus = "extracting"
	// ExtractStatusDone indicates extraction completed successfully
	ExtractStatusDone ExtractStatus = "done"
	// ExtractStatusError indicates extraction failed
	ExtractStatusError ExtractStatus = "error"
)

type Progress struct {
	// Total download time(ns)
	Used int64 `json:"used"`
	// Download speed(bytes/s)
	Speed int64 `json:"speed"`
	// Downloaded size(bytes)
	Downloaded int64 `json:"downloaded"`
	// Uploaded speed(bytes/s)
	UploadSpeed int64 `json:"uploadSpeed"`
	// Uploaded size(bytes)
	Uploaded int64 `json:"uploaded"`
	// ExtractStatus indicates the current status of archive extraction
	ExtractStatus ExtractStatus `json:"extractStatus"`
	// ExtractProgress is the percentage of extraction completed (0-100)
	ExtractProgress int `json:"extractProgress"`
	// MultiPartBaseName is set for multi-part archives to group related parts
	MultiPartBaseName string `json:"multiPartBaseName,omitempty"`
	// MultiPartNumber is the part number for multi-part archives (1-indexed)
	MultiPartNumber int `json:"multiPartNumber,omitempty"`
	// MultiPartIsFirst indicates if this is the first part of a multi-part archive
	MultiPartIsFirst bool `json:"multiPartIsFirst,omitempty"`
}

type Downloader struct {
	Logger          *logger.Logger
	ExtensionLogger *logger.Logger

	cfg          *DownloaderConfig
	fetcherCache map[string]fetcher.Fetcher
	storage      Storage
	tasks        []*Task
	waitTasks    []*Task
	watchedTasks sync.Map
	listener     Listener

	lock               *sync.Mutex
	fetcherMapLock     *sync.RWMutex
	checkDuplicateLock *sync.Mutex
	closed             atomic.Bool

	// claimedExtractions tracks which multi-part archives have been claimed for extraction
	// Key: fullBaseName (e.g., "/path/archive.7z"), Value: taskID that claimed it
	claimedExtractions sync.Map

	extensions []*Extension
	blob       *internalblob.Registry
}

func NewDownloader(cfg *DownloaderConfig) *Downloader {
	if cfg == nil {
		cfg = &DownloaderConfig{}
	}
	cfg.Init()

	d := &Downloader{
		cfg:          cfg,
		fetcherCache: make(map[string]fetcher.Fetcher),
		waitTasks:    make([]*Task, 0),
		storage:      cfg.Storage,

		lock:               &sync.Mutex{},
		fetcherMapLock:     &sync.RWMutex{},
		checkDuplicateLock: &sync.Mutex{},

		extensions: make([]*Extension, 0),
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	d.Logger = logger.NewLogger(cfg.ProductionMode, filepath.Join(cfg.StorageDir, "logs", "core.log"))
	d.ExtensionLogger = logger.NewLogger(cfg.ProductionMode, filepath.Join(cfg.StorageDir, "logs", "extension.log"))
	if cfg.ProductionMode {
		logPanic(filepath.Join(cfg.StorageDir, "logs"))
	}
	return d
}

func (d *Downloader) Setup() error {
	d.blob = internalblob.NewRegistry("")

	// setup storage
	if err := d.storage.Setup([]string{bucketTask, bucketSave, bucketProtocolState, bucketConfig, bucketExtension, bucketExtensionStorage}); err != nil {
		return err
	}
	// load config from storage
	var cfg base.DownloaderStoreConfig
	exist, err := d.storage.Get(bucketConfig, "config", &cfg)
	if err != nil {
		return err
	}
	if exist {
		d.cfg.DownloaderStoreConfig = &cfg
	} else {
		d.cfg.DownloaderStoreConfig = &base.DownloaderStoreConfig{
			FirstLoad: true,
		}
	}
	// init default config
	d.cfg.DownloaderStoreConfig.Init()
	// init protocol config, if not exist, use default config
	for _, fm := range d.cfg.FetchManagers {
		protocol := fm.Name()
		if _, ok := d.cfg.DownloaderStoreConfig.ProtocolConfig[protocol]; !ok {
			d.cfg.DownloaderStoreConfig.ProtocolConfig[protocol] = fm.DefaultConfig()
		}
		if sfm, ok := fm.(fetcher.StatefulFetcherManager); ok {
			sfm.SetStateStore(&protocolStateStore{
				storage:  d.storage,
				protocol: protocol,
			})
		}
	}

	// load tasks from storage
	var tasks []*Task
	if err = d.storage.List(bucketTask, &tasks); err != nil {
		return err
	}
	if tasks == nil {
		tasks = make([]*Task, 0)
	} else {
		for i := len(tasks) - 1; i >= 0; i-- {
			task := tasks[i]
			// Remove broken tasks
			if task.Meta == nil {
				tasks = append(tasks[:i], tasks[i+1:]...)
				continue
			}
			d.assignFetcherManager(task)
			initTask(task)
			if task.Status != base.DownloadStatusDone && task.Status != base.DownloadStatusError {
				task.Status = base.DownloadStatusPause
			}
		}
	}
	d.tasks = tasks
	// sort by create time
	sort.Slice(d.tasks, func(i, j int) bool {
		return d.tasks[i].CreatedAt.Before(d.tasks[j].CreatedAt)
	})

	// load extensions from storage
	var extensions []*Extension
	if err = d.storage.List(bucketExtension, &extensions); err != nil {
		return err
	}
	if extensions == nil {
		extensions = make([]*Extension, 0)
	}
	d.extensions = extensions

	// Auto-cleanup non-existing tasks on startup
	d.cleanupNonExistingTasks()

	// handle upload
	go func() {
		for _, task := range d.tasks {
			if task.Status == base.DownloadStatusDone && task.Uploading {
				if err := d.restoreTask(task); err != nil {
					d.Logger.Error().Stack().Err(err).Msgf("task upload restore fetcher failed, task id: %s", task.ID)
				}
				if uploader, ok := task.fetcher.(fetcher.Uploader); ok {
					if err := uploader.Upload(); err != nil {
						d.Logger.Error().Stack().Err(err).Msgf("task upload failed, task id: %s", task.ID)
					}
				}
			}
		}
	}()

	// calculate download speed every tick
	go func() {
		for !d.closed.Load() {
			if len(d.tasks) > 0 {
				for _, task := range d.tasks {
					func() {
						// Do not acquire d.lock (via GetTask) while holding
						// statusLock; scheduling uses the opposite lock order.
						if d.GetTask(task.ID) == nil {
							return
						}
						task.statusLock.Lock()
						if task.Status != base.DownloadStatusRunning && !task.Uploading {
							task.statusLock.Unlock()
							return
						}
						if task.fetcher == nil {
							task.statusLock.Unlock()
							return
						}

						current := task.fetcher.Progress().TotalDownloaded()
						tick := float64(d.cfg.RefreshInterval) / 1000
						downloadDataChanged := false
						if task.Status == base.DownloadStatusRunning {
							downloadDataChanged = current != task.Progress.Downloaded
							task.Progress.Used = task.timer.Used()
							task.Progress.Speed = task.updateSpeed(current-task.Progress.Downloaded, tick)
							task.Progress.Downloaded = current
						}

						uploadDataChanged := false
						if task.Uploading {
							uploader := task.fetcher.(fetcher.Uploader)
							currentUploaded := uploader.UploadedBytes()
							uploadDataChanged = currentUploaded != task.Progress.Uploaded
							task.Progress.UploadSpeed = task.updateUploadSpeed(currentUploaded-task.Progress.Uploaded, tick)
							task.Progress.Uploaded = currentUploaded
						}
						task.statusLock.Unlock()
						// Listener callbacks may Pause/Continue and acquire statusLock.
						d.emit(EventKeyProgress, task)

						// store fetcher progress when download/upload data changed
						if !downloadDataChanged && !uploadDataChanged {
							return
						}
						d.saveTask(task)
					}()
				}
			}
			time.Sleep(time.Millisecond * time.Duration(d.cfg.RefreshInterval))
		}
	}()
	return nil
}

// cleanupNonExistingTasks checks for tasks whose files are missing on disk
// and removes them if the AutoCleanMissingFiles config is enabled.
func (d *Downloader) cleanupNonExistingTasks() {
	cfg, err := d.GetConfig()
	if err != nil {
		return
	}

	// If the feature is disabled, do nothing
	if !cfg.AutoDeleteMissingFileTasks {
		return
	}

	var tasksToDelete []string

	for _, task := range d.tasks {
		if task.Meta == nil || task.Meta.Res == nil {
			continue
		}

		var targetPath string
		// Determine if it is a single file or a directory (multi-file torrent)
		if task.Meta.Res.Name != "" {
			targetPath = task.Meta.FolderPath()
		} else {
			targetPath = task.Meta.SingleFilepath()
		}

		// Skip if path is empty
		if targetPath == "" {
			continue
		}

		// Check if file/folder exists
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			d.Logger.Info().Msgf("Auto-cleanup: task %s file not found at %s, removing from list", task.ID, targetPath)
			tasksToDelete = append(tasksToDelete, task.ID)
		}
	}

	if len(tasksToDelete) > 0 {
		d.Delete(&TaskFilter{IDs: tasksToDelete}, false)
	}
}

func (d *Downloader) parseFm(url string) (fetcher.FetcherManager, error) {
	for _, fm := range d.cfg.FetchManagers {
		for _, filter := range fm.Filters() {
			if filter.Match(url) {
				return fm, nil
			}
		}
	}
	return nil, ErrUnSupportedProtocol
}

func (d *Downloader) setupFetcher(fm fetcher.FetcherManager, fetcher fetcher.Fetcher) {
	ctl := controller.NewController()
	ctl.GetConfig = func(v any) {
		d.getProtocolConfig(fm.Name(), v)
	}
	// Get proxy config, task request proxy config has higher priority, then use global proxy config
	ctl.GetProxy = func(requestProxy *base.RequestProxy) func(*gohttp.Request) (*url.URL, error) {
		// Registry-backed blob URLs are loopback capability URLs and must never be
		// sent through a configured proxy. Only exact, currently registered URLs
		// take this path; everything else keeps normal HTTP proxy behavior.
		if d.blob != nil && fetcher != nil && fetcher.Meta() != nil && fetcher.Meta().Req != nil && d.blob.IsURL(fetcher.Meta().Req.URL) {
			return nil
		}
		if requestProxy == nil {
			return d.cfg.Proxy.ToHandler()
		}
		switch requestProxy.Mode {
		case base.RequestProxyModeNone:
			return nil
		case base.RequestProxyModeCustom:
			return requestProxy.ToHandler()
		default:
			return d.cfg.Proxy.ToHandler()
		}
	}
	fetcher.Setup(ctl)
}

func (d *Downloader) saveTask(task *Task) error {
	data, err := task.fetcherManager.Store(task.fetcher)
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("serialize fetcher failed: %s", task.ID)
		return err
	}
	if data != nil {
		if err := d.storage.Put(bucketSave, task.ID, data); err != nil {
			d.Logger.Error().Stack().Err(err).Msgf("persist fetcher failed: %s", task.ID)
			return err
		}
	} else {
		if err := d.storage.Delete(bucketSave, task.ID); err != nil {
			d.Logger.Error().Stack().Err(err).Msgf("clear fetcher state failed: %s", task.ID)
			return err
		}
	}
	if err := d.storage.Put(bucketTask, task.ID, task); err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("persist task failed: %s", task.ID)
		return err
	}
	return nil
}

func ensureRequestRawURL(req *base.Request) {
	if req == nil {
		return
	}
	if req.RawURL == "" {
		req.RawURL = req.URL
	}
}

func ensureResourceRequestRawURLs(parentReq *base.Request, res *base.Resource) {
	if res == nil {
		return
	}
	for _, file := range res.Files {
		if file == nil || file.Req == nil {
			continue
		}
		ensureRequestRawURL(file.Req)
		if parentReq != nil && parentReq.RawURL != "" && file.Req.RawURL == file.Req.URL {
			file.Req.RawURL = parentReq.RawURL
		}
	}
}

func (d *Downloader) Resolve(req *base.Request, opts *base.Options) (rr *ResolveResult, err error) {
	rrId, err := gonanoid.New()
	if err != nil {
		return
	}
	ensureRequestRawURL(req)

	res, err := d.triggerOnResolve(req)
	if err != nil {
		return
	}
	if res != nil && len(res.Files) > 0 {
		rr = &ResolveResult{
			Res: res,
		}
		return
	}

	fetcher, err := d.buildFetcher(req.URL)
	if err != nil {
		return
	}
	initOpt, err := d.initOptions(opts)
	if err != nil {
		return
	}
	err = fetcher.Resolve(req, initOpt)
	if err != nil {
		return
	}
	d.fetcherMapLock.Lock()
	d.fetcherCache[rrId] = fetcher
	d.fetcherMapLock.Unlock()
	rr = &ResolveResult{
		ID:  rrId,
		Res: fetcher.Meta().Res,
	}
	return
}

func (d *Downloader) notifyRunning() {
	go func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		remainRunningCount := d.remainRunningCount()
		if remainRunningCount == 0 {
			return
		}
		if len(d.waitTasks) > 0 {
			wt := d.waitTasks[0]
			d.waitTasks = d.waitTasks[1:]
			d.doStart(wt)
		}
	}()
}

func (d *Downloader) remainRunningCount() int {
	runningCount := 0
	for _, t := range d.tasks {
		if t.Status == base.DownloadStatusRunning {
			runningCount++
		}
	}
	return d.cfg.MaxRunning - runningCount
}

func (d *Downloader) CreateDirect(req *base.Request, opts *base.Options) (taskId string, err error) {
	ensureRequestRawURL(req)
	var fetcher fetcher.Fetcher
	fetcher, err = d.buildFetcher(req.URL)
	if err != nil {
		return
	}
	fetcher.Meta().Req = req
	initOpt, err := d.initOptions(opts)
	if err != nil {
		return
	}
	return d.doCreate(fetcher, initOpt)
}

func (d *Downloader) CreateDirectBatch(req *base.CreateTaskBatch) (taskId []string, err error) {
	taskIds := make([]string, 0)
	for _, ir := range req.Reqs {
		opts := ir.Opts
		if opts == nil {
			opts = req.Opts
		}
		taskId, err := d.CreateDirect(ir.Req, opts.Clone())
		if err != nil {
			return nil, err
		}
		taskIds = append(taskIds, taskId)
	}
	return taskIds, nil
}

func (d *Downloader) Create(rrId string) (taskId string, err error) {
	d.fetcherMapLock.RLock()
	fetcher, ok := d.fetcherCache[rrId]
	d.fetcherMapLock.RUnlock()
	if !ok {
		return "", errors.New("invalid resource id")
	}
	defer func() {
		d.fetcherMapLock.Lock()
		delete(d.fetcherCache, rrId)
		d.fetcherMapLock.Unlock()
	}()
	return d.doCreate(fetcher, nil)
}

// Patch modifies task-specific data based on the protocol.
// For HTTP protocol, it can modify Request info.
// For BT protocol, it can modify SelectFiles.
func (d *Downloader) Patch(id string, req *base.Request, opts *base.Options) error {
	task := d.GetTask(id)
	if task == nil {
		return ErrTaskNotFound
	}
	if err := func() error {
		task.lock.Lock()
		defer task.lock.Unlock()

		// Restore fetcher if not loaded
		if task.fetcher == nil {
			err := func() error {
				task.statusLock.Lock()
				defer task.statusLock.Unlock()

				return d.restoreFetcher(task)
			}()
			if err != nil {
				return err
			}
		}

		// Call the fetcher's Patch method
		if err := task.fetcher.Patch(req, opts); err != nil {
			return err
		}

		// Update task meta from fetcher and transfer any live Blob lease.
		task.Meta = task.fetcher.Meta()
		if err := d.syncBlobTaskLease(task); err != nil {
			return err
		}

		return d.saveTask(task)
	}(); err != nil {
		return err
	}

	// Emit outside task.lock so listeners may safely call back into Patch/Pause.
	d.emit(EventKeyProgress, task)

	return nil
}

func (d *Downloader) Pause(filter *TaskFilter) (err error) {
	if filter == nil || filter.IsEmpty() {
		return d.pauseAll()
	}

	filter.NotStatuses = []base.Status{base.DownloadStatusPause, base.DownloadStatusError, base.DownloadStatusDone}
	pauseTasks := d.GetTasksByFilter(filter)
	if len(pauseTasks) == 0 {
		return ErrTaskNotFound
	}

	for _, task := range pauseTasks {
		if err = d.doPause(task); err != nil {
			return
		}
	}
	d.notifyRunning()

	return
}

func (d *Downloader) pauseAll() (err error) {
	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		// Clear wait tasks
		d.waitTasks = d.waitTasks[:0]
	}()

	for _, task := range d.tasks {
		if err = d.doPause(task); err != nil {
			return
		}
	}

	return
}

// Continue specific tasks, if continue tasks will exceed maxRunning, it needs pause some running tasks before that
func (d *Downloader) Continue(filter *TaskFilter) (err error) {
	if filter == nil || filter.IsEmpty() {
		return d.continueAll()
	}

	filter.NotStatuses = []base.Status{base.DownloadStatusRunning, base.DownloadStatusDone}
	continueTasks := d.GetTasksByFilter(filter)
	if len(continueTasks) == 0 {
		return ErrTaskNotFound
	}

	realContinueTasks := make([]*Task, 0)
	pausedTasks := make([]*Task, 0)
	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		continueCount := len(continueTasks)
		remainRunningCount := d.remainRunningCount()
		needRunningCount := int(math.Min(float64(d.cfg.MaxRunning), float64(continueCount)))
		needPauseCount := needRunningCount - remainRunningCount
		if needPauseCount > 0 {
			pausedCount := 0
			for _, task := range d.tasks {
				if task.Status == base.DownloadStatusRunning {
					var queued bool
					queued, err = d.doPauseForScheduling(task)
					if err != nil {
						return
					}
					if queued {
						d.waitTasks = append(d.waitTasks, task)
						pausedTasks = append(pausedTasks, task)
						pausedCount++
					}
				}
				if pausedCount == needPauseCount {
					break
				}
			}
		}

		for _, task := range continueTasks {
			if len(realContinueTasks) < needRunningCount {
				realContinueTasks = append(realContinueTasks, task)
			} else {
				task.Status = base.DownloadStatusWait
				d.waitTasks = append(d.waitTasks, task)
			}
		}
	}()
	if err != nil {
		return
	}
	// Scheduling pauses run while d.lock is held, but events must not: listeners
	// are allowed to call back into Downloader methods that acquire d.lock.
	for _, task := range pausedTasks {
		d.emit(EventKeyPause, task)
	}

	for _, task := range realContinueTasks {
		if err = d.doStart(task); err != nil {
			return
		}
	}

	return
}

// continueAll continue all tasks but does not affect tasks already running
func (d *Downloader) continueAll() (err error) {
	continuedTasks := make([]*Task, 0)

	func() {
		d.lock.Lock()
		defer d.lock.Unlock()
		// calculate how many tasks can be continued, can't exceed maxRunning
		remainCount := d.remainRunningCount()
		for _, task := range d.tasks {
			if task.Status != base.DownloadStatusRunning && task.Status != base.DownloadStatusDone {
				if len(continuedTasks) < remainCount {
					continuedTasks = append(continuedTasks, task)
				} else {
					task.Status = base.DownloadStatusWait
					d.waitTasks = append(d.waitTasks, task)
				}
			}
		}
	}()

	for _, task := range continuedTasks {
		if err = d.doStart(task); err != nil {
			return
		}
	}

	return
}

func (d *Downloader) ContinueBatch(filter *TaskFilter) (err error) {
	if filter == nil || filter.IsEmpty() {
		return d.continueAll()
	}

	continueTasks := d.GetTasksByFilter(filter)
	for _, task := range continueTasks {
		if err = d.doStart(task); err != nil {
			return
		}
	}
	return
}

func (d *Downloader) Delete(filter *TaskFilter, force bool) (err error) {
	if filter == nil || filter.IsEmpty() {
		return d.deleteAll(force)
	}

	deleteTasks := d.GetTasksByFilter(filter)
	if len(deleteTasks) == 0 {
		return
	}

	deleteIds := make([]string, 0)
	deleteTasksPtr := make([]*Task, 0)
	for _, task := range deleteTasks {
		deleteIds = append(deleteIds, task.ID)
		deleteTasksPtr = append(deleteTasksPtr, task)
	}
	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		for _, id := range deleteIds {
			for i, t := range d.tasks {
				if t.ID == id {
					d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
					break
				}
			}
			for i, t := range d.waitTasks {
				if t.ID == id {
					d.waitTasks = append(d.waitTasks[:i], d.waitTasks[i+1:]...)
					break
				}
			}
		}
	}()

	for _, task := range deleteTasksPtr {
		err = d.doDelete(task, force)
		if err != nil {
			return
		}
	}

	d.notifyRunning()
	return
}

func (d *Downloader) deleteAll(force bool) (err error) {
	var deleteTasksTemp []*Task
	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		for _, task := range d.tasks {
			deleteTasksTemp = append(deleteTasksTemp, task)
		}
		d.tasks = make([]*Task, 0)
		d.waitTasks = make([]*Task, 0)
	}()

	for _, task := range deleteTasksTemp {
		if err = d.doDelete(task, force); err != nil {
			return
		}
	}
	return
}

func (d *Downloader) Stats(id string) (sr any, err error) {
	task := d.GetTask(id)
	if task == nil {
		return sr, ErrTaskNotFound
	}
	if task.fetcher == nil {
		err = func() error {
			task.statusLock.Lock()
			defer task.statusLock.Unlock()

			return d.restoreFetcher(task)
		}()
		if err != nil {
			return
		}
	}
	sr = task.fetcher.Stats()
	return
}

func (d *Downloader) doDelete(task *Task, force bool) (err error) {
	defer d.releaseBlobTask(task)
	err = func() error {
		if err := d.storage.Delete(bucketTask, task.ID); err != nil {
			return err
		}
		if err := d.storage.Delete(bucketSave, task.ID); err != nil {
			return err
		}

		if task.fetcher != nil {
			if err := task.fetcher.Close(); err != nil {
				return err
			}
		}
		if force && task.Meta.Res != nil {
			if task.Meta.Res.Name != "" {
				if err := os.RemoveAll(task.Meta.FolderPath()); err != nil {
					return err
				}
			} else {
				if err := util.SafeRemove(task.Meta.SingleFilepath()); err != nil {
					return err
				}
			}
		}
		d.emit(EventKeyDelete, task)
		task = nil
		return nil
	}()

	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("delete task failed, task id: %s", task.ID)
	}
	return
}

func (d *Downloader) Close() error {
	d.closed.Store(true)

	closeArr := []func() error{
		d.pauseAll,
	}
	for _, fm := range d.cfg.FetchManagers {
		closeArr = append(closeArr, fm.Close)
	}
	if d.blob != nil {
		closeArr = append(closeArr, d.blob.Close)
	}
	closeArr = append(closeArr, d.storage.Close)
	// Make sure all resources are released, if had error, return the last error
	var lastErr error
	for i, close := range closeArr {
		if err := close(); err != nil {
			lastErr = err
			d.Logger.Error().Stack().Err(err).Msgf("downloader close failed, index: %d", i)
		}
	}
	return lastErr
}

func (d *Downloader) Clear() error {
	if !d.closed.Load() {
		if err := d.Close(); err != nil {
			return err
		}
	}
	d.tasks = make([]*Task, 0)
	d.extensions = make([]*Extension, 0)
	if err := d.storage.Clear(); err != nil {
		return err
	}
	return nil
}

type protocolStateStore struct {
	storage  Storage
	protocol string
}

func (s *protocolStateStore) Load(v any) (bool, error) {
	return s.storage.Get(bucketProtocolState, s.protocol, v)
}

func (s *protocolStateStore) Save(v any) error {
	if v == nil {
		return s.Delete()
	}
	return s.storage.Put(bucketProtocolState, s.protocol, v)
}

func (s *protocolStateStore) Delete() error {
	return s.storage.Delete(bucketProtocolState, s.protocol)
}

func (d *Downloader) Listener(fn Listener) {
	d.listener = fn
}

func (d *Downloader) emit(eventKey EventKey, task *Task, errs ...error) {
	if d.listener != nil {
		var err error
		if len(errs) > 0 {
			err = errs[0]
		}
		d.listener(&Event{
			Key:  eventKey,
			Task: task,
			Err:  err,
		})
	}
}

func (d *Downloader) GetTask(id string) *Task {
	d.lock.Lock()
	defer d.lock.Unlock()

	for _, task := range d.tasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (d *Downloader) GetTasks() []*Task {
	d.lock.Lock()
	defer d.lock.Unlock()

	return d.tasks
}

// GetTasksByFilter get tasks by filter, if filter is nil, return all tasks
// return tasks and if match all tasks
func (d *Downloader) GetTasksByFilter(filter *TaskFilter) []*Task {
	d.lock.Lock()
	defer d.lock.Unlock()

	if filter == nil || filter.IsEmpty() {
		return d.tasks
	}

	idMatch := func(task *Task) bool {
		if len(filter.IDs) == 0 {
			return true
		}
		for _, id := range filter.IDs {
			if task.ID == id {
				return true
			}
		}
		return false
	}
	statusMatch := func(task *Task) bool {
		if len(filter.Statuses) == 0 {
			return true
		}
		for _, status := range filter.Statuses {
			if task.Status == status {
				return true
			}
		}
		return false
	}
	notStatusMatch := func(task *Task) bool {
		if len(filter.NotStatuses) == 0 {
			return true
		}
		for _, status := range filter.NotStatuses {
			if task.Status == status {
				return false
			}
		}
		return true
	}

	tasks := make([]*Task, 0)
	for _, task := range d.tasks {
		if idMatch(task) && statusMatch(task) && notStatusMatch(task) {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (d *Downloader) GetConfig() (*base.DownloaderStoreConfig, error) {
	return d.cfg.DownloaderStoreConfig, nil
}

func (d *Downloader) PutConfig(v *base.DownloaderStoreConfig) error {
	d.cfg.DownloaderStoreConfig = v
	return d.storage.Put(bucketConfig, "config", v)
}

func (d *Downloader) getProtocolConfig(name string, v any) bool {
	cfg, err := d.GetConfig()
	if err != nil {
		return false
	}
	if cfg.ProtocolConfig == nil || cfg.ProtocolConfig[name] == nil {
		return false
	}
	if err := util.MapToStruct(cfg.ProtocolConfig[name], v); err != nil {
		d.Logger.Warn().Err(err).Msgf("get protocol config failed")
		return false
	}
	return true
}

// wait task done
func (d *Downloader) watch(task *Task) {
	if _, loaded := d.watchedTasks.LoadOrStore(task.ID, true); loaded {
		return
	}
	defer func() {
		d.watchedTasks.Delete(task.ID)
	}()

	// wait task upload done
	if task.Uploading {
		if uploader, ok := task.fetcher.(fetcher.Uploader); ok {
			go func() {
				err := uploader.WaitUpload()
				if err != nil {
					d.Logger.Warn().Err(err).Msgf("task wait upload failed, task id: %s", task.ID)
				}

				// Check if the task is deleted
				if d.GetTask(task.ID) != nil {
					task.Uploading = false
					d.storage.Put(bucketTask, task.ID, task.clone())
				}
			}()
		}
	}

	if d.taskStatus(task) == base.DownloadStatusDone {
		return
	}

	for {
		err := task.fetcher.Wait()
		if err != nil {
			status := d.taskStatus(task)
			if status == base.DownloadStatusPause || status == base.DownloadStatusWait {
				return
			}
			var sourceErr error
			if d.blob != nil && task.Meta != nil && task.Meta.Req != nil {
				sourceErr = d.blob.SourceError(task.Meta.Req.URL)
			}
			if sourceErr != nil {
				d.doOnBlobSourceError(task, sourceErr)
			} else {
				d.doOnError(task, err)
			}
			if d.taskStatus(task) == base.DownloadStatusRunning {
				continue
			}
			return
		}

		// When delete a not resolved task, need check if the task resource is nil
		if task.Meta.Res == nil || d.GetTask(task.ID) == nil {
			return
		}
		if d.blob != nil && task.Meta.Req != nil {
			if err := d.blob.SourceError(task.Meta.Req.URL); err != nil {
				d.doOnBlobSourceError(task, err)
				if d.taskStatus(task) == base.DownloadStatusRunning {
					continue
				}
				return
			}
		}

		task.Progress.Used = task.timer.Used()
		if task.Meta.Res.Size == 0 {
			task.Meta.Res.Size = task.fetcher.Progress().TotalDownloaded()
		}
		used := task.Progress.Used / int64(time.Second)
		if used == 0 {
			used = 1
		}
		totalSize := task.Meta.Res.Size
		task.Progress.Speed = totalSize / used
		task.Progress.Downloaded = totalSize
		if !d.markTaskDone(task) {
			return
		}
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.emit(EventKeyDone, task)
		d.emit(EventKeyFinally, task, err)
		d.notifyRunning()
		d.releaseBlobTask(task)
		d.triggerOnDone(task)
		d.triggerWebhooks(WebhookEventDownloadDone, task, nil)
		d.triggerScripts(ScriptEventDownloadDone, task, nil)

		if e, ok := task.Meta.Opts.Extra.(*http.OptsExtra); ok {
			downloadFilePath := task.Meta.SingleFilepath()

			cfg, _ := d.GetConfig()

			// Determine if auto-torrent is enabled (use global config if not explicitly set)
			autoTorrentEnabled := false
			if e.AutoTorrent != nil {
				autoTorrentEnabled = *e.AutoTorrent
			} else if cfg != nil && cfg.AutoTorrent != nil {
				autoTorrentEnabled = cfg.AutoTorrent.Enable
			}

			if autoTorrentEnabled && strings.HasSuffix(downloadFilePath, ".torrent") {
				// Determine if should delete torrent file after creating BT task
				shouldDelete := false
				if e.DeleteTorrentAfterDownload != nil {
					shouldDelete = *e.DeleteTorrentAfterDownload
				} else if cfg != nil && cfg.AutoTorrent != nil {
					shouldDelete = cfg.AutoTorrent.DeleteAfterDownload
				}

				go func() {
					_, err2 := d.CreateDirect(
						&base.Request{
							URL: downloadFilePath,
						},
						&base.Options{
							Path:        task.Meta.Opts.Path,
							SelectFiles: make([]int, 0),
						})
					if err2 != nil {
						d.Logger.Error().Err(err2).Msgf("auto create torrent task failed, task id: %s", task.ID)
						return
					}

					if shouldDelete {
						d.Delete(&TaskFilter{IDs: []string{task.ID}}, true)
					}
				}()
			}

			// Determine if auto-extract is enabled (use global config if not explicitly set)
			autoExtractEnabled := false
			if e.AutoExtract != nil {
				autoExtractEnabled = *e.AutoExtract
			} else if cfg != nil && cfg.Archive != nil {
				autoExtractEnabled = cfg.Archive.AutoExtract
			}

			// Auto-extract archive files using the extraction queue
			// This ensures only one extraction runs at a time to prevent resource exhaustion
			if autoExtractEnabled && isArchiveFile(downloadFilePath) {
				d.enqueueExtraction(task, downloadFilePath, e)
			}
		}
		return
	}
}

func (d *Downloader) doOnError(task *Task, err error) {
	d.handleOnError(task, err, false)
}

func (d *Downloader) doOnBlobSourceError(task *Task, err error) {
	d.handleOnError(task, err, true)
}

func (d *Downloader) handleOnError(task *Task, err error, resetFetcher bool) {
	if task == nil || task.lock == nil {
		return
	}
	task.lock.Lock()
	if !d.markTaskError(task) {
		task.lock.Unlock()
		return
	}
	d.Logger.Warn().Err(err).Msgf("task download failed, task id: %s", task.ID)
	oldURL := ""
	if task.Meta != nil && task.Meta.Req != nil {
		oldURL = task.Meta.Req.URL
	}
	d.triggerOnError(task, err)
	newURL := ""
	if task.Meta != nil && task.Meta.Req != nil {
		newURL = task.Meta.Req.URL
	}
	if oldURL != "" && newURL != oldURL {
		// Extensions are allowed to replace an expired URL from onError. Patch the
		// existing fetcher through its public API so protocol-owned redirect state
		// is reset without teaching the HTTP fetcher about blob URLs.
		if task.fetcher != nil {
			if patchErr := task.fetcher.Patch(&base.Request{URL: newURL}, nil); patchErr != nil {
				d.Logger.Warn().Err(patchErr).Msgf("patch recovered task url failed, task id: %s", task.ID)
			}
		}
		d.releaseBlobTaskURL(task, oldURL)
	}
	if resetFetcher && d.taskStatus(task) == base.DownloadStatusRunning {
		if resetErr := d.resetTaskFetcher(task); resetErr != nil {
			d.Logger.Warn().Err(resetErr).Msgf("reset recovered task fetcher failed, task id: %s", task.ID)
		}
	}
	task.lock.Unlock()
	if d.taskStatus(task) == base.DownloadStatusError {
		d.releaseBlobTask(task)
		d.emit(EventKeyError, task, err)
		d.emit(EventKeyFinally, task, err)
		d.notifyRunning()
		d.triggerWebhooks(WebhookEventDownloadError, task, err)
	}
}

func (d *Downloader) resetTaskFetcher(task *Task) error {
	if task == nil || task.Meta == nil || task.Meta.Req == nil {
		return errors.New("task request is unavailable")
	}
	fm, err := d.parseFm(task.Meta.Req.URL)
	if err != nil {
		return err
	}
	opts := task.Meta.Opts
	f := fm.Build()
	d.setupFetcher(fm, f)
	// A fresh source must never inherit bytes or preallocation from the failed
	// source. HTTP opens existing targets without truncating them, so remove the
	// old single-file target before installing a fresh fetcher.
	if task.Meta.Res != nil && task.Meta.Opts != nil && len(task.Meta.Res.Files) > 0 {
		if err := util.SafeRemove(task.Meta.SingleFilepath()); err != nil {
			return err
		}
	}
	f.Meta().Req = task.Meta.Req
	f.Meta().Opts = opts
	task.fetcherManager = fm
	task.fetcher = f
	task.Meta = f.Meta()
	if task.Progress != nil {
		task.Progress.Downloaded = 0
		task.Progress.Speed = 0
	}
	return d.storage.Delete(bucketSave, task.ID)
}

func (d *Downloader) restoreTask(task *Task) error {
	if task.fetcher == nil {
		if err := d.restoreFetcher(task); err != nil {
			return err
		}
	}
	return nil
}

func (d *Downloader) restoreFetcher(task *Task) error {
	v, f := task.fetcherManager.Restore()
	if v != nil {
		err := d.storage.Pop(bucketSave, task.ID, v)
		if err != nil {
			return err
		}
	}
	task.fetcher = f(task.Meta, v)
	if task.fetcher == nil {
		task.fetcher = task.fetcherManager.Build()
	}
	d.setupFetcher(task.fetcherManager, task.fetcher)
	if task.fetcher.Meta().Req == nil {
		task.fetcher.Meta().Req = task.Meta.Req
	}
	if task.fetcher.Meta().Res == nil {
		task.fetcher.Meta().Res = task.Meta.Res
	}
	if task.fetcher.Meta().Opts == nil {
		task.fetcher.Meta().Opts = task.Meta.Opts
	}
	return nil
}

func (d *Downloader) doCreate(f fetcher.Fetcher, opts *base.Options) (taskId string, err error) {
	if f.Meta().Opts == nil {
		f.Meta().Opts = opts
	}
	ensureRequestRawURL(f.Meta().Req)

	fm, err := d.parseFm(f.Meta().Req.URL)
	if err != nil {
		return
	}
	task := NewTask()
	task.fetcherManager = fm
	task.fetcher = f
	task.Protocol = fm.Name()
	task.Meta = f.Meta()
	task.Progress = &Progress{}
	_, task.Uploading = f.(fetcher.Uploader)
	initTask(task)
	if err = d.syncBlobTaskLease(task); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			d.releaseBlobTask(task)
		}
	}()
	if err = d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
		return
	}
	taskId = task.ID

	func() {
		d.lock.Lock()
		defer d.lock.Unlock()

		d.tasks = append(d.tasks, task)

		remainRunningCount := d.remainRunningCount()
		if remainRunningCount == 0 {
			task.Status = base.DownloadStatusWait
			d.waitTasks = append(d.waitTasks, task)
			return
		}

		err = d.doStart(task)
	}()

	return
}

func (d *Downloader) initOptions(opts *base.Options) (*base.Options, error) {
	if opts == nil {
		opts = &base.Options{}
	}
	if opts.SelectFiles == nil {
		opts.SelectFiles = make([]int, 0)
	}
	if opts.Path == "" {
		storeConfig, err := d.GetConfig()
		if err != nil {
			return nil, err
		}
		opts.Path = storeConfig.DownloadDir
	}
	// Replace placeholders in download path (e.g., %year%, %month%, %day%, %date%)
	opts.Path = util.ReplacePathPlaceholders(opts.Path)

	// if enable white download directory, check if the download directory is in the white list
	if len(d.cfg.WhiteDownloadDirs) > 0 {
		inWhiteList := false
		for _, dir := range d.cfg.WhiteDownloadDirs {
			if match, err := filepath.Match(dir, opts.Path); match && err == nil {
				inWhiteList = true
				break
			}
		}
		if !inWhiteList {
			return nil, errors.New("download directory is not in white list")
		}
	}
	return opts, nil
}

func (d *Downloader) statusMut(task *Task, fn func() (bool, error)) (bool, error) {
	task.statusLock.Lock()
	defer task.statusLock.Unlock()

	return fn()
}

func (d *Downloader) doStart(task *Task) (err error) {
	var isCreate bool
	var generation uint64
	isReturn, err := d.statusMut(task, func() (isReturn bool, err error) {
		if task.Status == base.DownloadStatusRunning || task.Status == base.DownloadStatusDone {
			isReturn = true
			return
		}

		err = d.restoreTask(task)
		if err != nil {
			d.Logger.Error().Stack().Err(err).Msgf("restore fetcher failed, task id: %s", task.ID)
			return
		}
		isCreate = task.Status == base.DownloadStatusReady
		task.updateStatus(base.DownloadStatusRunning)
		task.runGeneration++
		generation = task.runGeneration

		return
	})
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("start task failed, task id: %s", task.ID)
		return
	}
	if isReturn {
		return
	}

	started := false
	handler := func() error {
		task.lock.Lock()
		defer task.lock.Unlock()
		if !d.taskIsRunningGeneration(task, generation) {
			return nil
		}

		if task.fetcher == nil {
			if err := d.restoreFetcher(task); err != nil {
				return err
			}
		}
		d.triggerOnStart(task)
		if !d.taskIsRunningGeneration(task, generation) {
			return nil
		}
		if task.fetcher != nil {
			task.fetcher.Meta().Req = task.Meta.Req
			task.fetcher.Meta().Res = task.Meta.Res
			task.fetcher.Meta().Opts = task.Meta.Opts
		}
		if err := d.syncBlobTaskLease(task); err != nil {
			return err
		}
		if task.Meta.Res == nil {
			err := task.fetcher.Resolve(task.Meta.Req, task.Meta.Opts)
			if err != nil {
				return err
			}
			task.Meta.Res = task.fetcher.Meta().Res
		}

		if isCreate {
			if task.fetcherManager.AutoRename() {
				d.checkDuplicateLock.Lock()
				defer d.checkDuplicateLock.Unlock()
				task.Meta.Opts.Name = util.SafeFilename(task.Meta.Opts.Name)
				// check if the download file is duplicated and rename it automatically.
				if task.Meta.Res.Name != "" {
					task.Meta.Res.Name = util.SafeFilename(task.Meta.Res.Name)
					fullDirPath := task.Meta.FolderPath()
					newName, err := util.CheckDuplicateAndRename(fullDirPath)
					if err != nil {
						return err
					}
					task.Meta.Opts.Name = newName
				} else {
					task.Meta.Res.Files[0].Name = util.SafeFilename(task.Meta.Res.Files[0].Name)
					fullFilePath := task.Meta.SingleFilepath()
					newName, err := util.CheckDuplicateAndRename(fullFilePath)
					if err != nil {
						return err
					}
					task.Meta.Opts.Name = newName
				}
			}

			task.Meta.Res.CalcSize(task.Meta.Opts.SelectFiles)
		}

		task.Progress.Speed = 0
		if !d.taskIsRunningGeneration(task, generation) {
			return nil
		}
		task.timer.Start()
		if err := task.fetcher.Start(); err != nil {
			return err
		}
		started = true
		if err := d.saveTask(task); err != nil {
			return err
		}
		return nil
	}
	go func() {
		if err := handler(); err != nil {
			if d.taskIsRunningGeneration(task, generation) {
				d.doOnError(task, err)
			}
			return
		}
		if started {
			if d.taskIsRunningGeneration(task, generation) {
				d.emit(EventKeyStart, task)
			}
			go d.watch(task)
		}
	}()

	return
}

func (d *Downloader) doPause(task *Task) (err error) {
	generation, isReturn, err := d.preparePause(task)
	if err != nil || isReturn {
		return err
	}
	handled, err := d.runPauseHandler(task, generation, true)
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("pause task handle failed, task id: %s", task.ID)
		return err
	}
	if handled {
		d.emit(EventKeyPause, task)
	}
	return nil
}

// doPauseForScheduling physically pauses a running task without emitting while
// the caller holds d.lock, then atomically moves it to the wait queue state.
func (d *Downloader) doPauseForScheduling(task *Task) (bool, error) {
	generation, isReturn, err := d.preparePause(task)
	if err != nil || isReturn {
		return false, err
	}
	handled, err := d.runPauseHandler(task, generation, true)
	if err != nil || !handled {
		return false, err
	}
	task.statusLock.Lock()
	queued := task.Status == base.DownloadStatusPause && task.runGeneration == generation
	if queued {
		task.Status = base.DownloadStatusWait
	}
	task.statusLock.Unlock()
	return queued, nil
}

func (d *Downloader) preparePause(task *Task) (generation uint64, isReturn bool, err error) {
	isReturn, err = d.statusMut(task, func() (isReturn bool, err error) {
		if task.Status == base.DownloadStatusPause || task.Status == base.DownloadStatusDone {
			isReturn = true
			return
		}

		task.updateStatus(base.DownloadStatusPause)
		task.runGeneration++
		generation = task.runGeneration
		task.timer.Pause()
		return
	})
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("pause task failed, task id: %s", task.ID)
	}
	return generation, isReturn, err
}

func (d *Downloader) runPauseHandler(task *Task, generation uint64, pauseFetcher bool) (bool, error) {
	task.lock.Lock()
	defer task.lock.Unlock()
	if !d.taskIsPausedGeneration(task, generation) {
		return false, nil
	}

	if pauseFetcher && task.fetcher != nil {
		if err := task.fetcher.Pause(); err != nil {
			return false, err
		}
	}
	if task.fetcherManager != nil && task.fetcher != nil {
		if err := d.saveTask(task); err != nil {
			return false, err
		}
	} else {
		if err := d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (d *Downloader) taskIsRunningGeneration(task *Task, generation uint64) bool {
	if task == nil || task.statusLock == nil {
		return false
	}
	task.statusLock.Lock()
	defer task.statusLock.Unlock()
	return task.Status == base.DownloadStatusRunning && task.runGeneration == generation
}

func (d *Downloader) taskIsPausedGeneration(task *Task, generation uint64) bool {
	if task == nil || task.statusLock == nil {
		return false
	}
	task.statusLock.Lock()
	defer task.statusLock.Unlock()
	return (task.Status == base.DownloadStatusPause || task.Status == base.DownloadStatusWait) && task.runGeneration == generation
}

func (d *Downloader) taskStatus(task *Task) base.Status {
	if task == nil || task.statusLock == nil {
		return ""
	}
	task.statusLock.Lock()
	defer task.statusLock.Unlock()
	return task.Status
}

func (d *Downloader) markTaskDone(task *Task) bool {
	if task == nil || task.statusLock == nil {
		return false
	}
	task.statusLock.Lock()
	defer task.statusLock.Unlock()
	if task.Status != base.DownloadStatusRunning {
		return false
	}
	task.updateStatus(base.DownloadStatusDone)
	task.runGeneration++
	return true
}

func (d *Downloader) markTaskError(task *Task) bool {
	if task == nil || task.statusLock == nil {
		return false
	}
	task.statusLock.Lock()
	defer task.statusLock.Unlock()
	if task.Status != base.DownloadStatusRunning {
		return false
	}
	task.updateStatus(base.DownloadStatusError)
	task.runGeneration++
	return true
}

// redirect stderr to log file, when panic happened log it
func logPanic(logDir string) {
	if err := util.CreateDirIfNotExist(logDir); err != nil {
		return
	}
	f, err := os.Create(filepath.Join(logDir, "crash.log"))
	if err != nil {
		return
	}
	debug.SetCrashOutput(f, debug.CrashOptions{})
}

func (d *Downloader) assignFetcherManager(task *Task) error {
	fm, err := d.parseFm(task.Meta.Req.URL)
	if err != nil {
		return err
	}
	task.fetcherManager = fm
	return nil
}

func (d *Downloader) buildFetcher(url string) (fetcher.Fetcher, error) {
	fm, err := d.parseFm(url)
	if err != nil {
		return nil, err
	}
	fetcher := fm.Build()
	d.setupFetcher(fm, fetcher)
	return fetcher, nil
}

// enqueueExtraction adds an extraction job to the global extraction queue
// This ensures only one extraction (or one multi-part archive extraction) runs at a time
// to prevent resource exhaustion
func (d *Downloader) enqueueExtraction(task *Task, downloadFilePath string, opts *http.OptsExtra) {
	partInfo := getArchivePartInfo(downloadFilePath)

	if partInfo.IsMultiPart {
		// For multi-part archives, handle specially
		d.enqueueMultiPartExtraction(task, downloadFilePath, partInfo, opts)
	} else {
		// For single archives, queue immediately
		d.enqueueSingleExtraction(task, downloadFilePath, opts)
	}
}

// enqueueSingleExtraction queues extraction for a single (non-multi-part) archive
func (d *Downloader) enqueueSingleExtraction(task *Task, downloadFilePath string, opts *http.OptsExtra) {
	jobID := "single:" + task.ID

	// Set extraction status to queued
	task.Progress.ExtractStatus = ExtractStatusQueued
	d.emit(EventKeyProgress, task)
	d.storage.Put(bucketTask, task.ID, task.clone())
	d.Logger.Info().Msgf("extraction queued, task id: %s, job id: %s", task.ID, jobID)

	// Create and enqueue the extraction job
	job := NewExtractionJob(jobID, func() {
		d.performExtraction(task, downloadFilePath, task.Meta.Opts.Path, opts)
	})

	go func() {
		GetExtractionQueue().Enqueue(job)
	}()
}

// enqueueMultiPartExtraction handles queueing for multi-part archives
// It ensures only ONE extraction job is queued when ALL parts are ready
func (d *Downloader) enqueueMultiPartExtraction(task *Task, downloadFilePath string, partInfo ArchivePartInfo, opts *http.OptsExtra) {
	// Set multi-part info on the task
	task.Progress.MultiPartBaseName = partInfo.BaseName
	task.Progress.MultiPartNumber = partInfo.PartNumber
	task.Progress.MultiPartIsFirst = isFirstPart(downloadFilePath)

	// Check if all parts are downloaded
	destDir := task.Meta.Opts.Path
	allPartsReady, missingParts := d.checkMultiPartArchiveReady(downloadFilePath, destDir, partInfo)

	if !allPartsReady {
		// Not all parts are ready yet - just set status to waiting, don't queue anything
		task.Progress.ExtractStatus = ExtractStatusWaitingParts
		d.emit(EventKeyProgress, task)
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.Logger.Info().Msgf("multi-part archive waiting for other parts, task id: %s, missing: %v", task.ID, missingParts)
		return
	}

	// All parts are ready! Atomically check if extraction has already been started/queued
	// and if not, mark this task as the one that will handle it
	// Use GetMultiPartArchiveBaseName to get the full path for comparison
	fullBaseName := GetMultiPartArchiveBaseName(downloadFilePath)
	shouldQueue := d.tryClaimMultiPartExtraction(task, fullBaseName)

	if !shouldQueue {
		// Another part already started/queued extraction, mark this task as done
		task.Progress.ExtractStatus = ExtractStatusDone
		task.Progress.ExtractProgress = 100
		d.emit(EventKeyProgress, task)
		d.storage.Put(bucketTask, task.ID, task.clone())
		d.Logger.Info().Msgf("multi-part archive extraction already handled by another part, task id: %s", task.ID)
		return
	}

	// This task claimed the extraction - status already set to queued in tryClaimMultiPartExtraction
	d.emit(EventKeyProgress, task)
	d.storage.Put(bucketTask, task.ID, task.clone())

	jobID := "multipart:" + fullBaseName
	d.Logger.Info().Msgf("multi-part extraction queued, task id: %s, job id: %s", task.ID, jobID)

	// Create and enqueue the extraction job
	job := NewExtractionJob(jobID, func() {
		d.performMultiPartExtraction(task, partInfo.FirstPartPath, destDir, opts)
	})

	go func() {
		GetExtractionQueue().Enqueue(job)
	}()
}

// checkMultiPartArchiveReady checks if all parts of a multi-part archive are downloaded
// by examining task status rather than file existence
func (d *Downloader) checkMultiPartArchiveReady(filePath string, destDir string, partInfo ArchivePartInfo) (bool, []string) {
	// Use task-based checking - find all tasks with the same MultiPartBaseName
	// and verify they are all in Done status
	baseName := GetMultiPartArchiveBaseName(filePath)
	if baseName == "" {
		return true, nil
	}

	return d.checkAllMultiPartTasksDone(baseName)
}

// checkAllMultiPartTasksDone checks if all tasks belonging to a multi-part archive are done
func (d *Downloader) checkAllMultiPartTasksDone(baseName string) (bool, []string) {
	var notDoneParts []string

	d.lock.Lock()
	defer d.lock.Unlock()

	// Find all tasks that belong to this multi-part archive
	var relatedTasks []*Task
	for _, task := range d.tasks {
		taskBaseName := ""
		if task.Meta != nil && task.Meta.Res != nil && len(task.Meta.Res.Files) > 0 {
			taskBaseName = GetMultiPartArchiveBaseName(task.Meta.SingleFilepath())
		}
		if taskBaseName == baseName {
			relatedTasks = append(relatedTasks, task)
		}
	}

	// If we found no related tasks, we can't determine readiness
	if len(relatedTasks) == 0 {
		return false, []string{"no related tasks found for " + baseName}
	}

	// Check if all related tasks are done
	for _, task := range relatedTasks {
		if task.Status != base.DownloadStatusDone {
			notDoneParts = append(notDoneParts, task.Meta.SingleFilepath())
		}
	}

	return len(notDoneParts) == 0, notDoneParts
}

// tryClaimMultiPartExtraction atomically checks if extraction can be claimed for a multi-part archive
// and if so, marks the task as queued. Returns true if this task should proceed with queueing.
// This uses sync.Map.LoadOrStore for atomic claim to prevent race conditions.
func (d *Downloader) tryClaimMultiPartExtraction(task *Task, baseName string) bool {
	// Use LoadOrStore for atomic claim - if another goroutine already stored a value, we get that value back
	_, alreadyClaimed := d.claimedExtractions.LoadOrStore(baseName, task.ID)
	if alreadyClaimed {
		return false // Another task already claimed it
	}

	// This task successfully claimed it
	task.Progress.ExtractStatus = ExtractStatusQueued
	return true
}

// releaseMultiPartExtractionClaim releases the extraction claim for a multi-part archive
// This is primarily used for testing purposes
func (d *Downloader) releaseMultiPartExtractionClaim(baseName string) {
	d.claimedExtractions.Delete(baseName)
}

// performExtraction performs extraction for a regular (non-multi-part) archive
func (d *Downloader) performExtraction(task *Task, archivePath string, destDir string, opts *http.OptsExtra) {
	// Set extraction status to extracting
	task.Progress.ExtractStatus = ExtractStatusExtracting
	task.Progress.ExtractProgress = 0
	d.emit(EventKeyProgress, task)
	d.storage.Put(bucketTask, task.ID, task.clone())

	// Extract the archive
	extractErr := extractArchive(archivePath, destDir, opts.ArchivePassword, func(extractedFiles int, totalFiles int, progress int) {
		task.Progress.ExtractProgress = progress
		d.emit(EventKeyProgress, task)
	})

	d.handleExtractionResult(task, extractErr, []string{archivePath}, opts.DeleteAfterExtract)
}

// performMultiPartExtraction performs extraction for a multi-part archive
func (d *Downloader) performMultiPartExtraction(task *Task, firstPartPath string, destDir string, opts *http.OptsExtra) {
	// Get the baseName for releasing the claim later
	fullBaseName := GetMultiPartArchiveBaseName(firstPartPath)

	// Set extraction status to extracting
	task.Progress.ExtractStatus = ExtractStatusExtracting
	task.Progress.ExtractProgress = 0
	d.emit(EventKeyProgress, task)
	d.storage.Put(bucketTask, task.ID, task.clone())

	d.Logger.Info().Msgf("starting multi-part archive extraction, first part: %s, task id: %s", firstPartPath, task.ID)

	// Extract the multi-part archive
	extractErr := extractMultiPartArchive(firstPartPath, destDir, opts.ArchivePassword, func(extractedFiles int, totalFiles int, progress int) {
		task.Progress.ExtractProgress = progress
		d.emit(EventKeyProgress, task)
	})

	// Collect all part files for potential deletion
	partFiles := d.collectMultiPartFiles(firstPartPath)

	d.handleExtractionResult(task, extractErr, partFiles, opts.DeleteAfterExtract)

	// Update status for all related multi-part tasks
	d.updateMultiPartTasksStatus(task, extractErr)

	// Release the claim so future downloads of the same archive can be extracted
	d.releaseMultiPartExtractionClaim(fullBaseName)
}

// collectMultiPartFiles collects all files belonging to a multi-part archive
func (d *Downloader) collectMultiPartFiles(firstPartPath string) []string {
	var files []string
	partInfo := getArchivePartInfo(firstPartPath)
	dir := filepath.Dir(firstPartPath)

	switch {
	case strings.Contains(partInfo.Pattern, ".7z)"):
		// 7z: .7z.001, .7z.002, etc.
		files = d.collectSequentialFiles(dir, partInfo.BaseName, ".%03d")
	case strings.Contains(partInfo.Pattern, ".part"):
		// RAR new style
		files = d.collectRarNewStyleFiles(dir, partInfo.BaseName)
	case partInfo.Pattern == "rar-old-style" || strings.Contains(partInfo.Pattern, ".r("):
		// RAR old style
		files = d.collectRarOldStyleFiles(dir, partInfo.BaseName)
	case strings.Contains(partInfo.Pattern, ".zip)"):
		// ZIP multi-part
		files = d.collectSequentialFiles(dir, partInfo.BaseName, ".%03d")
	case strings.Contains(partInfo.Pattern, ".z("):
		// ZIP split
		files = d.collectZipSplitFiles(dir, partInfo.BaseName)
	}

	return files
}

// collectSequentialFiles collects sequential numbered files
func (d *Downloader) collectSequentialFiles(dir, baseName, format string) []string {
	var files []string
	suffix := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, suffix)
	partNum := 1

	for {
		partPath := filepath.Join(dir, nameWithoutExt+suffix+fmt.Sprintf(format, partNum))
		if _, err := os.Stat(partPath); os.IsNotExist(err) {
			break
		}
		files = append(files, partPath)
		partNum++
	}

	return files
}

// collectRarNewStyleFiles collects RAR new style part files
func (d *Downloader) collectRarNewStyleFiles(dir, baseName string) []string {
	var files []string
	partNum := 1

	for {
		singleDigitPath := filepath.Join(dir, baseName+fmt.Sprintf(".part%d.rar", partNum))
		doubleDigitPath := filepath.Join(dir, baseName+fmt.Sprintf(".part%02d.rar", partNum))

		if _, err := os.Stat(singleDigitPath); err == nil {
			files = append(files, singleDigitPath)
		} else if _, err := os.Stat(doubleDigitPath); err == nil {
			files = append(files, doubleDigitPath)
		} else {
			break
		}
		partNum++
	}

	return files
}

// collectRarOldStyleFiles collects RAR old style part files
func (d *Downloader) collectRarOldStyleFiles(dir, baseName string) []string {
	var files []string

	// .rar file
	rarPath := filepath.Join(dir, baseName+".rar")
	if _, err := os.Stat(rarPath); err == nil {
		files = append(files, rarPath)
	}

	// .r00, .r01, etc.
	partNum := 0
	for {
		partPath := filepath.Join(dir, baseName+fmt.Sprintf(".r%02d", partNum))
		if _, err := os.Stat(partPath); os.IsNotExist(err) {
			break
		}
		files = append(files, partPath)
		partNum++
	}

	return files
}

// collectZipSplitFiles collects ZIP split files
func (d *Downloader) collectZipSplitFiles(dir, baseName string) []string {
	var files []string

	// .z01, .z02, etc.
	partNum := 1
	for {
		partPath := filepath.Join(dir, baseName+fmt.Sprintf(".z%02d", partNum))
		if _, err := os.Stat(partPath); os.IsNotExist(err) {
			break
		}
		files = append(files, partPath)
		partNum++
	}

	// .zip file
	zipPath := filepath.Join(dir, baseName+".zip")
	if _, err := os.Stat(zipPath); err == nil {
		files = append(files, zipPath)
	}

	return files
}

// handleExtractionResult handles the result of an extraction operation
func (d *Downloader) handleExtractionResult(task *Task, extractErr error, archiveFiles []string, deleteAfterExtract bool) {
	if extractErr != nil {
		d.Logger.Error().Err(extractErr).Msgf("auto extract archive failed, task id: %s", task.ID)
		task.Progress.ExtractStatus = ExtractStatusError
		d.emit(EventKeyProgress, task)
		d.storage.Put(bucketTask, task.ID, task.clone())
	} else {
		d.Logger.Info().Msgf("auto extract archive completed, task id: %s", task.ID)
		task.Progress.ExtractStatus = ExtractStatusDone
		task.Progress.ExtractProgress = 100
		d.emit(EventKeyProgress, task)
		d.storage.Put(bucketTask, task.ID, task.clone())

		// Delete archive files after successful extraction if enabled
		if deleteAfterExtract {
			for _, archiveFile := range archiveFiles {
				deleteErr := os.Remove(archiveFile)
				if deleteErr != nil {
					d.Logger.Error().Err(deleteErr).Msgf("delete archive after extraction failed: %s", archiveFile)
				} else {
					d.Logger.Info().Msgf("archive deleted after extraction: %s", archiveFile)
				}
			}
		}
	}
}

// updateMultiPartTasksStatus updates the extraction status for all tasks that belong to the same multi-part archive
func (d *Downloader) updateMultiPartTasksStatus(sourceTask *Task, extractErr error) {
	if sourceTask.Progress.MultiPartBaseName == "" {
		return
	}

	status := ExtractStatusDone
	progress := 100
	if extractErr != nil {
		status = ExtractStatusError
		progress = 0
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	for _, task := range d.tasks {
		if task.ID == sourceTask.ID {
			continue
		}
		if task.Progress.MultiPartBaseName == sourceTask.Progress.MultiPartBaseName {
			task.Progress.ExtractStatus = status
			task.Progress.ExtractProgress = progress
			d.emit(EventKeyProgress, task)
			d.storage.Put(bucketTask, task.ID, task.clone())
		}
	}
}

func initTask(task *Task) {
	task.timer = util.NewTimer(task.Progress.Used)

	task.statusLock = &sync.Mutex{}
	task.lock = &sync.Mutex{}
	task.blobRefLock = &sync.Mutex{}
	task.speedArr = make([]int64, 0)
	task.uploadSpeedArr = make([]int64, 0)
}

func (d *Downloader) releaseBlobTask(task *Task) {
	if d.blob == nil || task == nil {
		return
	}
	if task.blobRefLock != nil {
		task.blobRefLock.Lock()
		raw := task.blobURL
		task.blobURL = ""
		task.blobRefLock.Unlock()
		if raw != "" {
			_ = d.blob.Release(raw)
		}
	}
}

func (d *Downloader) syncBlobTaskLease(task *Task) error {
	if d.blob == nil || task == nil || task.Meta == nil || task.Meta.Req == nil {
		return nil
	}
	if task.blobRefLock == nil {
		task.blobRefLock = &sync.Mutex{}
	}
	task.blobRefLock.Lock()
	defer task.blobRefLock.Unlock()
	terminal := false
	if task.statusLock != nil {
		task.statusLock.Lock()
		terminal = task.Status == base.DownloadStatusDone || task.Status == base.DownloadStatusError
		task.statusLock.Unlock()
	}
	newURL := ""
	if !terminal && d.blob.IsURL(task.Meta.Req.URL) {
		newURL = task.Meta.Req.URL
	}
	if task.blobURL == newURL {
		return nil
	}
	if newURL != "" {
		if err := d.blob.Acquire(newURL); err != nil {
			return err
		}
	}
	oldURL := task.blobURL
	task.blobURL = newURL
	if oldURL != "" {
		_ = d.blob.Release(oldURL)
	}
	return nil
}

func (d *Downloader) releaseBlobTaskURL(task *Task, raw string) {
	if d.blob == nil || task == nil || raw == "" {
		return
	}
	if task.blobRefLock == nil {
		task.blobRefLock = &sync.Mutex{}
	}
	task.blobRefLock.Lock()
	claimed := task.blobURL == raw
	if claimed {
		task.blobURL = ""
	}
	task.blobRefLock.Unlock()
	if claimed {
		_ = d.blob.Release(raw)
	}
}

var defaultDownloader = NewDownloader(nil)

type boot struct {
	url      string
	extra    interface{}
	listener Listener
}

func (b *boot) URL(url string) *boot {
	b.url = url
	return b
}

func (b *boot) Extra(extra interface{}) *boot {
	b.extra = extra
	return b
}

func (b *boot) Listener(listener Listener) *boot {
	b.listener = listener
	return b
}

func (b *boot) Create(opts *base.Options) (string, error) {
	defaultDownloader.Listener(b.listener)
	return defaultDownloader.CreateDirect(&base.Request{
		URL:   b.url,
		Extra: b.extra,
	}, opts)
}

func Boot() *boot {
	err := defaultDownloader.Setup()
	if err != nil {
		panic(err)
	}
	return &boot{}
}
