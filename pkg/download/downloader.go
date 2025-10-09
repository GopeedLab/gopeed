package download

import (
	"errors"
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

	extensions []*Extension
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
	// setup storage
	if err := d.storage.Setup([]string{bucketTask, bucketSave, bucketConfig, bucketExtension, bucketExtensionStorage}); err != nil {
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
						task.statusLock.Lock()
						defer task.statusLock.Unlock()
						if task.Status != base.DownloadStatusRunning && !task.Uploading {
							return
						}
						// check if task is deleted
						if d.GetTask(task.ID) == nil || task.fetcher == nil {
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
	if err := d.storage.Put(bucketSave, task.ID, data); err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("persist fetcher failed: %s", task.ID)
		return err
	}
	if err := d.storage.Put(bucketTask, task.ID, task); err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("persist task failed: %s", task.ID)
		return err
	}
	return nil
}

func (d *Downloader) Resolve(req *base.Request) (rr *ResolveResult, err error) {
	rrId, err := gonanoid.New()
	if err != nil {
		return
	}

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
	err = fetcher.Resolve(req)
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
	var fetcher fetcher.Fetcher
	fetcher, err = d.buildFetcher(req.URL)
	if err != nil {
		return
	}
	fetcher.Meta().Req = req
	return d.doCreate(fetcher, opts)
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

func (d *Downloader) Create(rrId string, opts *base.Options) (taskId string, err error) {
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
	return d.doCreate(fetcher, opts)
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
					if err = d.doPause(task); err != nil {
						return
					}
					task.Status = base.DownloadStatusWait
					d.waitTasks = append(d.waitTasks, task)
					pausedCount++
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
	for _, task := range d.tasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (d *Downloader) GetTasks() []*Task {
	return d.tasks
}

// GetTasksByFilter get tasks by filter, if filter is nil, return all tasks
// return tasks and if match all tasks
func (d *Downloader) GetTasksByFilter(filter *TaskFilter) []*Task {
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
				d.lock.Lock()
				defer d.lock.Unlock()

				// Check if the task is deleted
				if d.GetTask(task.ID) != nil {
					task.Uploading = false
					d.storage.Put(bucketTask, task.ID, task.clone())
				}
			}()
		}
	}

	if task.Status == base.DownloadStatusDone {
		return
	}

	err := task.fetcher.Wait()
	if err != nil {
		d.doOnError(task, err)
		return
	}

	// When delete a not resolved task, need check if the task resource is nil
	if task.Meta.Res == nil || d.GetTask(task.ID) == nil {
		return
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
	task.updateStatus(base.DownloadStatusDone)
	d.storage.Put(bucketTask, task.ID, task.clone())
	d.emit(EventKeyDone, task)
	d.emit(EventKeyFinally, task, err)
	d.notifyRunning()
	d.triggerOnDone(task)

	if e, ok := task.Meta.Opts.Extra.(*http.OptsExtra); ok {
		downloadFilePath := task.Meta.SingleFilepath()
		if e.AutoTorrent && strings.HasSuffix(downloadFilePath, ".torrent") {
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
				}

			}()
		}
	}
}

func (d *Downloader) doOnError(task *Task, err error) {
	d.Logger.Warn().Err(err).Msgf("task download failed, task id: %s", task.ID)
	task.updateStatus(base.DownloadStatusError)
	d.triggerOnError(task, err)
	if task.Status == base.DownloadStatusError {
		d.emit(EventKeyError, task, err)
		d.emit(EventKeyFinally, task, err)
		d.notifyRunning()
	}
}

func (d *Downloader) restoreTask(task *Task) error {
	if task.fetcher == nil {
		if err := d.restoreFetcher(task); err != nil {
			return err
		}
	}
	go d.watch(task)
	task.fetcher.Create(task.Meta.Opts)
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
	return nil
}

func (d *Downloader) doCreate(f fetcher.Fetcher, opts *base.Options) (taskId string, err error) {
	if opts == nil {
		opts = &base.Options{}
	}
	if opts.SelectFiles == nil {
		opts.SelectFiles = make([]int, 0)
	}

	meta := f.Meta()
	meta.Opts = opts
	if opts.Path == "" {
		storeConfig, err := d.GetConfig()
		if err != nil {
			return "", err
		}
		opts.Path = storeConfig.DownloadDir
	}

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
			return "", errors.New("download directory is not in white list")
		}
	}

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
	if err = f.Create(opts); err != nil {
		return
	}
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

	go d.watch(task)
	return
}

func (d *Downloader) statusMut(task *Task, fn func() (bool, error)) (bool, error) {
	task.statusLock.Lock()
	defer task.statusLock.Unlock()

	return fn()
}

func (d *Downloader) doStart(task *Task) (err error) {
	var isCreate bool
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

		return
	})
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("start task failed, task id: %s", task.ID)
		return
	}
	if isReturn {
		return
	}

	handler := func() error {
		task.lock.Lock()
		defer task.lock.Unlock()

		d.triggerOnStart(task)
		if task.Meta.Res == nil {
			err := task.fetcher.Resolve(task.Meta.Req)
			if err != nil {
				return err
			}
			task.Meta.Res = task.fetcher.Meta().Res
		}

		if isCreate {
			if task.fetcherManager.AutoRename() {
				d.checkDuplicateLock.Lock()
				defer d.checkDuplicateLock.Unlock()
				task.Meta.Opts.Name = util.ReplaceInvalidFilename(task.Meta.Opts.Name)
				// check if the download file is duplicated and rename it automatically.
				if task.Meta.Res.Name != "" {
					task.Meta.Res.Name = util.ReplaceInvalidFilename(task.Meta.Res.Name)
					fullDirPath := task.Meta.FolderPath()
					newName, err := util.CheckDuplicateAndRename(fullDirPath)
					if err != nil {
						return err
					}
					task.Meta.Opts.Name = newName
				} else {
					task.Meta.Res.Files[0].Name = util.ReplaceInvalidFilename(task.Meta.Res.Files[0].Name)
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
		task.timer.Start()
		if err := task.fetcher.Start(); err != nil {
			return err
		}
		if err := d.saveTask(task); err != nil {
			return err
		}
		d.emit(EventKeyStart, task)
		return nil
	}
	go func() {
		if err := handler(); err != nil {
			d.doOnError(task, err)
		}
	}()

	return
}

func (d *Downloader) doPause(task *Task) (err error) {
	isReturn, err := d.statusMut(task, func() (isReturn bool, err error) {
		if task.Status == base.DownloadStatusPause || task.Status == base.DownloadStatusDone {
			isReturn = true
			return
		}

		task.updateStatus(base.DownloadStatusPause)
		task.timer.Pause()
		return
	})
	if err != nil {
		d.Logger.Error().Stack().Err(err).Msgf("pause task failed, task id: %s", task.ID)
		return
	}
	if isReturn {
		return
	}

	handler := func() error {
		task.lock.Lock()
		defer task.lock.Unlock()

		if task.fetcher != nil {
			if err := task.fetcher.Pause(); err != nil {
				return err
			}
		}
		if err := d.storage.Put(bucketTask, task.ID, task.clone()); err != nil {
			return err
		}
		d.emit(EventKeyPause, task)
		return nil
	}
	go func() {
		if err := handler(); err != nil {
			d.Logger.Error().Stack().Err(err).Msgf("pause task handle failed, task id: %s", task.ID)
		}
	}()
	return
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

func initTask(task *Task) {
	task.timer = util.NewTimer(task.Progress.Used)

	task.statusLock = &sync.Mutex{}
	task.lock = &sync.Mutex{}
	task.speedArr = make([]int64, 0)
	task.uploadSpeedArr = make([]int64, 0)
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

func (b *boot) Resolve() (*ResolveResult, error) {
	return defaultDownloader.Resolve(&base.Request{
		URL:   b.url,
		Extra: b.extra,
	})
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
