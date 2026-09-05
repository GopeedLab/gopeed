package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	restmodel "github.com/GopeedLab/gopeed/pkg/rest/model"
)

type Context struct {
	Method     string
	Path       string
	Query      url.Values
	Body       []byte
	Headers    http.Header
	PathParams map[string]string
}

func (c *Context) PathParam(name string) string {
	if c == nil || c.PathParams == nil {
		return ""
	}
	return c.PathParams[name]
}

type Response struct {
	StatusCode int
	Body       any
}

type Service struct {
	Downloader *download.Downloader
	routes     []RouteSpec
	taskEvents taskEventSubscription
}

func NewService(downloader *download.Downloader) (*Service, error) {
	if downloader == nil {
		return nil, fmt.Errorf("downloader is nil")
	}
	service := &Service{Downloader: downloader}
	service.routes = []RouteSpec{
		newRoute(http.MethodGet, "/api/v1/info", service.info),
		newRoute(http.MethodPost, "/api/v1/resolve", service.resolve),
		newRoute(http.MethodPost, "/api/v1/tasks", service.createTask),
		newRoute(http.MethodPost, "/api/v1/tasks/batch", service.createTaskBatch),
		newRoute(http.MethodPatch, "/api/v1/tasks/{id}", service.patchTask),
		newRoute(http.MethodPut, "/api/v1/tasks/{id}/pause", service.pauseTask),
		newRoute(http.MethodPut, "/api/v1/tasks/pause", service.pauseTasks),
		newRoute(http.MethodPut, "/api/v1/tasks/{id}/continue", service.continueTask),
		newRoute(http.MethodPut, "/api/v1/tasks/continue", service.continueTasks),
		newRoute(http.MethodDelete, "/api/v1/tasks/{id}", service.deleteTask),
		newRoute(http.MethodDelete, "/api/v1/tasks", service.deleteTasks),
		newRoute(http.MethodGet, "/api/v1/tasks/{id}", service.getTask),
		newRoute(http.MethodGet, "/api/v1/tasks/{id}/status", service.getTaskStatus),
		newRoute(http.MethodGet, "/api/v1/tasks", service.getTasks),
		newRoute(http.MethodGet, "/api/v1/tasks/{id}/stats", service.getStats),
		newRoute(http.MethodGet, "/api/v1/config", service.getConfig),
		newRoute(http.MethodPut, "/api/v1/config", service.putConfig),
		newRoute(http.MethodPost, "/api/v1/extensions", service.installExtension),
		newRoute(http.MethodGet, "/api/v1/extensions", service.getExtensions),
		newRoute(http.MethodGet, "/api/v1/extensions/{identity}", service.getExtension),
		newRoute(http.MethodPut, "/api/v1/extensions/{identity}/settings", service.updateExtensionSettings),
		newRoute(http.MethodPut, "/api/v1/extensions/{identity}/switch", service.switchExtension),
		newRoute(http.MethodDelete, "/api/v1/extensions/{identity}", service.deleteExtension),
		newRoute(http.MethodGet, "/api/v1/extensions/{identity}/update", service.updateCheckExtension),
		newRoute(http.MethodPost, "/api/v1/extensions/{identity}/update", service.updateExtension),
		newRoute(http.MethodPost, "/api/v1/webhook/test", service.testWebhook),
	}
	if err := validateRoutes(service.routes); err != nil {
		return nil, err
	}
	downloader.Listener(service.emitTaskEvent)
	return service, nil
}

func (s *Service) RouteSpecs() []RouteSpec {
	return append([]RouteSpec(nil), s.routes...)
}

func (s *Service) Dispatch(method, rawPath string, query url.Values, body []byte, headers http.Header) (response *Response) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response = newErrorResponse(fmt.Sprintf("%v", recovered))
		}
	}()

	method = strings.ToUpper(strings.TrimSpace(method))
	path := cleanPath(rawPath)
	for _, route := range s.routes {
		if route.Method != method {
			continue
		}
		pathParams, matched := route.Match(path)
		if !matched {
			continue
		}
		return route.Handler(&Context{
			Method:     method,
			Path:       path,
			Query:      query,
			Body:       body,
			Headers:    headers,
			PathParams: pathParams,
		})
	}
	return newStatusResponse(http.StatusNotFound, restmodel.NewErrorResult("not found"))
}

func (s *Service) JSON(method, rawPath string, query url.Values, body []byte, headers http.Header) (int, []byte) {
	response := s.Dispatch(method, rawPath, query, body, headers)
	data, err := json.Marshal(response.Body)
	if err != nil {
		response = newErrorResponse(err.Error())
		data, _ = json.Marshal(response.Body)
	}
	return response.StatusCode, data
}

func (s *Service) info(_ *Context) *Response {
	return newOKResponse(map[string]any{
		"version":  base.Version,
		"runtime":  runtime.Version(),
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"inDocker": base.InDocker == "true",
	})
}

func (s *Service) resolve(ctx *Context) *Response {
	var request restmodel.ResolveTask
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	result, err := s.Downloader.Resolve(request.Req, request.Opts)
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(result)
}

func (s *Service) createTask(ctx *Context) *Response {
	var request restmodel.CreateTask
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	var (
		taskID string
		err    error
	)
	if request.Rid != "" {
		taskID, err = s.Downloader.Create(request.Rid)
	} else if request.Req != nil {
		taskID, err = s.Downloader.CreateDirect(request.Req, request.Opts)
	} else {
		return newErrorResponse("param invalid: rid or req", restmodel.CodeInvalidParam)
	}
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(taskID)
}

func (s *Service) createTaskBatch(ctx *Context) *Response {
	var request base.CreateTaskBatch
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	if len(request.Reqs) == 0 {
		return newErrorResponse("param invalid: reqs", restmodel.CodeInvalidParam)
	}
	taskIDs, err := s.Downloader.CreateDirectBatch(&request)
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(taskIDs)
}

func (s *Service) patchTask(ctx *Context) *Response {
	taskID := ctx.PathParam("id")
	if taskID == "" {
		return invalidIDResponse()
	}
	var request restmodel.ResolveTask
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	if err := s.Downloader.Patch(taskID, request.Req, request.Opts); err != nil {
		if err == download.ErrTaskNotFound {
			return taskNotFoundResponse()
		}
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) pauseTask(ctx *Context) *Response {
	return s.applyTaskAction(ctx, s.Downloader.Pause)
}

func (s *Service) pauseTasks(ctx *Context) *Response {
	return applyFilterAction(taskFilterFromQuery(ctx.Query), s.Downloader.Pause)
}

func (s *Service) continueTask(ctx *Context) *Response {
	return s.applyTaskAction(ctx, s.Downloader.Continue)
}

func (s *Service) continueTasks(ctx *Context) *Response {
	return applyFilterAction(taskFilterFromQuery(ctx.Query), s.Downloader.Continue)
}

func (s *Service) applyTaskAction(ctx *Context, action func(*download.TaskFilter) error) *Response {
	taskID := ctx.PathParam("id")
	if taskID == "" {
		return invalidIDResponse()
	}
	return applyFilterAction(&download.TaskFilter{IDs: []string{taskID}}, action)
}

func applyFilterAction(filter *download.TaskFilter, action func(*download.TaskFilter) error) *Response {
	if err := action(filter); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) deleteTask(ctx *Context) *Response {
	taskID := ctx.PathParam("id")
	if taskID == "" {
		return invalidIDResponse()
	}
	return deleteTasks(s.Downloader, &download.TaskFilter{IDs: []string{taskID}}, ctx.Query.Get("force") == "true")
}

func (s *Service) deleteTasks(ctx *Context) *Response {
	return deleteTasks(s.Downloader, taskFilterFromQuery(ctx.Query), ctx.Query.Get("force") == "true")
}

func deleteTasks(downloader *download.Downloader, filter *download.TaskFilter, force bool) *Response {
	if err := downloader.Delete(filter, force); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) getTask(ctx *Context) *Response {
	taskID := ctx.PathParam("id")
	if taskID == "" {
		return invalidIDResponse()
	}
	task := s.Downloader.GetTask(taskID)
	if task == nil {
		return taskNotFoundResponse()
	}
	return newOKResponse(task)
}

func (s *Service) getTaskStatus(ctx *Context) *Response {
	taskID := ctx.PathParam("id")
	if taskID == "" {
		return invalidIDResponse()
	}
	status, err := s.Downloader.RuntimeStatus(taskID)
	if err != nil {
		if err == download.ErrTaskNotFound {
			return taskNotFoundResponse()
		}
		return newErrorResponse(err.Error())
	}
	return newOKResponse(status)
}

func (s *Service) getTasks(ctx *Context) *Response {
	return newOKResponse(s.Downloader.GetTasksByFilter(taskFilterFromQuery(ctx.Query)))
}

func (s *Service) getStats(ctx *Context) *Response {
	taskID := ctx.PathParam("id")
	if taskID == "" {
		return invalidIDResponse()
	}
	stats, err := s.Downloader.Stats(taskID)
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(stats)
}

func (s *Service) getConfig(_ *Context) *Response {
	config, err := s.Downloader.GetConfig()
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(config)
}

func (s *Service) putConfig(ctx *Context) *Response {
	var config base.DownloaderStoreConfig
	if response := decodeRequest(ctx.Body, &config); response != nil {
		return response
	}
	if err := s.Downloader.PutConfig(&config); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) installExtension(ctx *Context) *Response {
	var request restmodel.InstallExtension
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	var (
		extension *download.Extension
		err       error
	)
	if request.DevMode {
		extension, err = s.Downloader.InstallExtensionByFolder(request.URL, true)
	} else {
		extension, err = s.Downloader.InstallExtensionByGit(request.URL)
	}
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(extension.Identity)
}

func (s *Service) getExtensions(_ *Context) *Response {
	return newOKResponse(s.Downloader.GetExtensions())
}

func (s *Service) getExtension(ctx *Context) *Response {
	identity := ctx.PathParam("identity")
	if identity == "" {
		return newErrorResponse("param invalid: identity", restmodel.CodeInvalidParam)
	}
	extension, err := s.Downloader.GetExtension(identity)
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(extension)
}

func (s *Service) updateExtensionSettings(ctx *Context) *Response {
	identity := ctx.PathParam("identity")
	var request restmodel.UpdateExtensionSettings
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	if err := s.Downloader.UpdateExtensionSettings(identity, request.Settings); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) switchExtension(ctx *Context) *Response {
	identity := ctx.PathParam("identity")
	var request restmodel.SwitchExtension
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	if err := s.Downloader.SwitchExtension(identity, request.Status); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) deleteExtension(ctx *Context) *Response {
	if err := s.Downloader.DeleteExtension(ctx.PathParam("identity")); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) updateCheckExtension(ctx *Context) *Response {
	version, err := s.Downloader.UpgradeCheckExtension(ctx.PathParam("identity"))
	if err != nil {
		return newErrorResponse(err.Error())
	}
	return newOKResponse(&restmodel.UpdateCheckExtensionResp{NewVersion: version})
}

func (s *Service) updateExtension(ctx *Context) *Response {
	if err := s.Downloader.UpgradeExtension(ctx.PathParam("identity")); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func (s *Service) testWebhook(ctx *Context) *Response {
	var request restmodel.TestWebhookReq
	if response := decodeRequest(ctx.Body, &request); response != nil {
		return response
	}
	if err := s.Downloader.TestWebhookUrl(request.URL); err != nil {
		return newErrorResponse(err.Error())
	}
	return newNilResponse()
}

func decodeRequest(body []byte, target any) *Response {
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(target); err != nil {
		return newErrorResponse(err.Error())
	}
	return nil
}

func taskFilterFromQuery(query url.Values) *download.TaskFilter {
	if query == nil {
		query = url.Values{}
	}
	return &download.TaskFilter{
		IDs:         query["id"],
		Statuses:    convertStatuses(query["status"]),
		NotStatuses: convertStatuses(query["notStatus"]),
	}
}

func convertStatuses(values []string) []base.Status {
	statuses := make([]base.Status, 0, len(values))
	for _, value := range values {
		statuses = append(statuses, base.Status(value))
	}
	return statuses
}

func invalidIDResponse() *Response {
	return newErrorResponse("param invalid: id", restmodel.CodeInvalidParam)
}

func taskNotFoundResponse() *Response {
	return newErrorResponse("task not found", restmodel.CodeTaskNotFound)
}

func newOKResponse(data any) *Response {
	return &Response{StatusCode: http.StatusOK, Body: restmodel.NewOkResult(data)}
}

func newNilResponse() *Response {
	return &Response{StatusCode: http.StatusOK, Body: restmodel.NewNilResult()}
}

func newErrorResponse(message string, code ...restmodel.RespCode) *Response {
	return &Response{StatusCode: http.StatusOK, Body: restmodel.NewErrorResult(message, code...)}
}

func newStatusResponse(statusCode int, body any) *Response {
	return &Response{StatusCode: statusCode, Body: body}
}
