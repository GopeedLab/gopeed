package api

import (
	"context"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/api/model"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/util"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"sync"
)

type Instance struct {
	startCfg   *model.StartConfig
	downloader *download.Downloader

	httpLock sync.Mutex
	srv      *http.Server
	listener net.Listener
}

func Create(startCfg *model.StartConfig) (*Instance, error) {
	if startCfg == nil {
		startCfg = &model.StartConfig{}
	}
	startCfg.Init()

	i := &Instance{
		startCfg: startCfg,
	}

	downloadCfg := &download.DownloaderConfig{
		ProductionMode:  startCfg.ProductionMode,
		RefreshInterval: startCfg.RefreshInterval,
	}
	if startCfg.Storage == model.StorageBolt {
		downloadCfg.Storage = download.NewBoltStorage(startCfg.StorageDir)
	} else {
		downloadCfg.Storage = download.NewMemStorage()
	}
	downloadCfg.StorageDir = startCfg.StorageDir
	downloadCfg.Init()
	downloader := download.NewDownloader(downloadCfg)
	if err := downloader.Setup(); err != nil {
		return nil, err
	}
	i.downloader = downloader
	i.httpLock = sync.Mutex{}

	return i, nil
}

func (i *Instance) Listener(listener download.Listener) *model.Result[any] {
	i.downloader.Listener(listener)
	return model.NewNilResult()
}

func (i *Instance) StartHttp() *model.Result[int] {
	i.httpLock.Lock()
	defer i.httpLock.Unlock()

	return i.startHttp()
}

func (i *Instance) startHttp() *model.Result[int] {
	httpCfg := i.getHttpConfig()
	if httpCfg == nil || !httpCfg.Enable {
		return model.NewErrorResult[int]("HTTP API server not enabled")
	}
	if i.srv != nil {
		return model.NewErrorResult[int]("HTTP API server already started")
	}

	port, start, err := ListenHttp(httpCfg, i)
	if err != nil {
		return model.NewErrorResult[int](err.Error())
	}
	httpCfg.RunningPort = port
	go start()
	return model.NewOkResult(port)
}

func (i *Instance) getHttpConfig() *base.DownloaderHttpConfig {
	var httpCfg *base.DownloaderHttpConfig
	if i.startCfg.DownloadConfig != nil && i.startCfg.DownloadConfig.Http != nil {
		httpCfg = i.startCfg.DownloadConfig.Http
	} else {
		cfg, _ := i.downloader.GetConfig()
		httpCfg = cfg.Http
	}
	return httpCfg
}

func (i *Instance) StopHttp() *model.Result[any] {
	i.httpLock.Lock()
	defer i.httpLock.Unlock()

	return i.stopHttp()
}

func (i *Instance) stopHttp() *model.Result[any] {
	if i.srv != nil {
		if err := i.srv.Shutdown(context.Background()); err != nil {
			i.downloader.Logger.Warn().Err(err).Msg("shutdown http server failed")
		}
		i.srv = nil
		i.listener = nil
	}
	httpCfg := i.getHttpConfig()
	if httpCfg != nil {
		httpCfg.RunningPort = 0
	}
	return model.NewNilResult()
}

func (i *Instance) RestartHttp() *model.Result[int] {
	i.httpLock.Lock()
	defer i.httpLock.Unlock()

	i.stopHttp()
	return i.startHttp()
}

func (i *Instance) Info() *model.Result[map[string]any] {
	return model.NewOkResult(map[string]any{
		"version":  base.Version,
		"runtime":  runtime.Version(),
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"inDocker": base.InDocker == "true",
	})
}

func (i *Instance) Resolve(req *base.Request) *model.Result[*download.ResolveResult] {
	rr, err := i.downloader.Resolve(req)
	if err != nil {
		return model.NewErrorResult[*download.ResolveResult](err.Error())
	}
	return model.NewOkResult(rr)
}

func (i *Instance) CreateTask(req *model.CreateTask) *model.Result[string] {
	var (
		taskId string
		err    error
	)
	if req.Rid != "" {
		taskId, err = i.downloader.Create(req.Rid, req.Opt)
	} else if req.Req != nil {
		taskId, err = i.downloader.CreateDirect(req.Req, req.Opt)
	} else {
		return model.NewErrorResult[string]("param invalid: rid or req", model.CodeInvalidParam)
	}
	if err != nil {
		return model.NewErrorResult[string](err.Error())
	}
	return model.NewOkResult(taskId)
}

func (i *Instance) CreateTaskBatch(req *model.CreateTaskBatch) *model.Result[[]string] {
	if len(req.Reqs) == 0 {
		return model.NewErrorResult[[]string]("param invalid: reqs", model.CodeInvalidParam)
	}
	taskIds, err := i.downloader.CreateDirectBatch(req.Reqs, req.Opt)
	if err != nil {
		return model.NewErrorResult[[]string](err.Error())
	}
	return model.NewOkResult(taskIds)
}

func (i *Instance) PauseTask(taskId string) *model.Result[any] {
	err := i.downloader.Pause(&download.TaskFilter{
		ID: []string{taskId},
	})
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) PauseTasks(filter *download.TaskFilter) *model.Result[any] {
	err := i.downloader.Pause(filter)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) ContinueTask(taskId string) *model.Result[any] {
	err := i.downloader.Continue(&download.TaskFilter{
		ID: []string{taskId},
	})
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) ContinueTasks(filter *download.TaskFilter) *model.Result[any] {
	err := i.downloader.Continue(filter)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) DeleteTask(taskId string, force bool) *model.Result[any] {
	err := i.downloader.Delete(&download.TaskFilter{
		ID: []string{taskId},
	}, force)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) DeleteTasks(filter *download.TaskFilter, force bool) *model.Result[any] {
	err := i.downloader.Delete(filter, force)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) GetTask(taskId string) *model.Result[*download.Task] {
	if taskId == "" {
		return model.NewErrorResult[*download.Task]("param invalid: id", model.CodeInvalidParam)
	}
	task := i.downloader.GetTask(taskId)
	if task == nil {
		return model.NewErrorResult[*download.Task]("task not found", model.CodeTaskNotFound)
	}
	return model.NewOkResult(task)
}

func (i *Instance) GetTasks(filter *download.TaskFilter) *model.Result[[]*download.Task] {
	return model.NewOkResult(i.downloader.GetTasksByFilter(filter))
}

func (i *Instance) GetTaskStats(taskId string) *model.Result[any] {
	stats, err := i.downloader.GetTaskStats(taskId)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewOkResult(stats)
}

func (i *Instance) GetConfig() *model.Result[*base.DownloaderStoreConfig] {
	config, err := i.downloader.GetConfig()
	if err != nil {
		return model.NewErrorResult[*base.DownloaderStoreConfig](err.Error())
	}
	return model.NewOkResult(config)
}

func (i *Instance) PutConfig(cfg *base.DownloaderStoreConfig) *model.Result[any] {
	err := i.downloader.PutConfig(cfg)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) InstallExtension(req *model.InstallExtension) *model.Result[string] {
	var (
		installedExt *download.Extension
		err          error
	)
	if req.DevMode {
		installedExt, err = i.downloader.InstallExtensionByFolder(req.URL, true)
	} else {
		installedExt, err = i.downloader.InstallExtensionByGit(req.URL)
	}
	if err != nil {
		return model.NewErrorResult[string](err.Error())
	}

	return model.NewOkResult(installedExt.Identity)
}

func (i *Instance) GetExtension(identity string) *model.Result[*download.Extension] {
	extension, err := i.downloader.GetExtension(identity)
	if err != nil {
		return model.NewErrorResult[*download.Extension](err.Error())
	}
	return model.NewOkResult(extension)
}

func (i *Instance) GetExtensions() *model.Result[[]*download.Extension] {
	return model.NewOkResult(i.downloader.GetExtensions())
}

func (i *Instance) UpdateExtensionSettings(identity string, req *model.UpdateExtensionSettings) *model.Result[any] {
	err := i.downloader.UpdateExtensionSettings(identity, req.Settings)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) SwitchExtension(identity string, req *model.SwitchExtension) *model.Result[any] {
	err := i.downloader.SwitchExtension(identity, req.Status)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) DeleteExtension(identity string) *model.Result[any] {
	err := i.downloader.DeleteExtension(identity)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) UpgradeCheckExtension(identity string) *model.Result[*model.UpgradeCheckExtensionResp] {
	newVersion, err := i.downloader.UpgradeCheckExtension(identity)
	if err != nil {
		return model.NewErrorResult[*model.UpgradeCheckExtensionResp](err.Error())
	}

	return model.NewOkResult(&model.UpgradeCheckExtensionResp{
		NewVersion: newVersion,
	})
}

func (i *Instance) UpgradeExtension(identity string) *model.Result[any] {
	err := i.downloader.UpgradeExtension(identity)
	if err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

func (i *Instance) Close() *model.Result[any] {
	i.StopHttp()
	if i.downloader != nil {
		if err := i.downloader.Close(); err != nil {
			i.downloader.Logger.Warn().Err(err).Msg("close downloader failed")
		}
		i.downloader = nil
	}

	return model.NewNilResult()
}

func (i *Instance) Clear() *model.Result[any] {
	if err := i.downloader.Clear(); err != nil {
		return model.NewErrorResult[any](err.Error())
	}
	return model.NewNilResult()
}

type Request struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

// Invoke support dynamic call method
func Invoke(instance *Instance, request *Request) (ret any) {
	defer func() {
		if err := recover(); err != nil {
			ret = model.NewErrorResult[any](fmt.Sprintf("%v", err))
		}
	}()

	method, args := request.Method, request.Params
	dsType := reflect.ValueOf(instance)
	fn := dsType.MethodByName(method)
	numIn := fn.Type().NumIn()
	in := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		paramType := fn.Type().In(i)
		arg := args[i]
		if arg == nil {
			in[i] = reflect.Zero(fn.Type().In(i))
			continue
		}
		var param reflect.Value
		var paramPtr any
		if paramType.Kind() == reflect.Ptr {
			param = reflect.New(paramType.Elem())
			paramPtr = param.Interface()
		} else {
			param = reflect.New(paramType).Elem()
			paramPtr = param.Addr().Interface()
		}
		if err := util.MapToStruct(arg, paramPtr); err != nil {
			panic(err)
		}
		in[i] = param
	}
	retVals := fn.Call(in)
	return retVals[0].Interface()
}
