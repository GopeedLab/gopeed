// Package mcpserver exposes Gopeed's task operations over the Model Context
// Protocol.
package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const developmentVersion = "dev"

const serverInstructions = "Gopeed is a download manager for HTTP, HTTPS, BitTorrent, magnet, and ED2K resources. " +
	"When the user asks to download a resource and provides a supported URL or URI, use create_task. " +
	"Use resolve_task first when the user needs resource metadata or wants to select files before creating the task. " +
	"If the user has not provided a concrete URL or URI, ask for one instead of inventing it."

// NewHandler returns a stateless Streamable HTTP MCP handler.
func NewHandler(downloader *download.Downloader) http.Handler {
	server := NewServer(downloader)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	return http.NewCrossOriginProtection().Handler(handler)
}

// NewServer builds an MCP server backed by downloader.
func NewServer(downloader *download.Downloader) *mcp.Server {
	if downloader == nil {
		panic("mcpserver: nil downloader")
	}

	version := base.Version
	if version == "" {
		version = developmentVersion
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "gopeed", Version: version}, &mcp.ServerOptions{
		Instructions: serverInstructions,
	})
	registerTools(server, downloader)
	return server
}

type requestExtra map[string]any

type httpRequestExtra struct {
	Method string            `json:"method,omitempty" jsonschema:"Optional HTTP method; defaults to GET"`
	Header map[string]string `json:"header,omitempty" jsonschema:"Optional HTTP request headers"`
	Body   string            `json:"body,omitempty" jsonschema:"Optional HTTP request body"`
}

type btRequestExtra struct {
	Trackers []string `json:"trackers" jsonschema:"BitTorrent tracker URLs to add to the task"`
}

type httpTaskExtra struct {
	Connections                int    `json:"connections,omitempty" jsonschema:"Number of HTTP download connections"`
	AutoTorrent                *bool  `json:"autoTorrent,omitempty" jsonschema:"Automatically create a BitTorrent task after downloading a torrent file"`
	DeleteTorrentAfterDownload *bool  `json:"deleteTorrentAfterDownload,omitempty" jsonschema:"Delete the torrent file after automatically creating its BitTorrent task"`
	AutoExtract                *bool  `json:"autoExtract,omitempty" jsonschema:"Automatically extract the downloaded archive"`
	ArchivePassword            string `json:"archivePassword,omitempty" jsonschema:"Password for an encrypted archive"`
	DeleteAfterExtract         bool   `json:"deleteAfterExtract,omitempty" jsonschema:"Delete archive files after successful extraction"`
}

// downloadRequest is the MCP-facing form of base.Request. The REST model uses
// any for Extra, which produces a boolean JSON Schema and is misread as a
// string by some MCP clients. The MCP schema replaces it with a oneOf union of
// the supported protocol-specific request objects.
type downloadRequest struct {
	URL            string             `json:"url" jsonschema:"Download URL or URI. Supports HTTP, HTTPS, magnet, BitTorrent torrent sources, and ED2K links"`
	RawURL         string             `json:"rawUrl,omitempty" jsonschema:"Original user-provided URL or URI when it differs from url"`
	Extra          requestExtra       `json:"extra,omitempty" jsonschema:"Optional protocol-specific request settings"`
	Labels         map[string]string  `json:"labels,omitempty" jsonschema:"Optional labels to attach to the download task"`
	Proxy          *base.RequestProxy `json:"proxy,omitempty" jsonschema:"Optional proxy settings for this download request"`
	SkipVerifyCert bool               `json:"skipVerifyCert,omitempty" jsonschema:"Skip TLS certificate verification for this request"`
}

func (r *downloadRequest) baseRequest() (*base.Request, error) {
	if r == nil {
		return nil, nil
	}
	if err := r.validateExtra(); err != nil {
		return nil, err
	}
	return &base.Request{
		RawURL:         r.RawURL,
		URL:            r.URL,
		Extra:          r.Extra,
		Labels:         r.Labels,
		Proxy:          r.Proxy,
		SkipVerifyCert: r.SkipVerifyCert,
	}, nil
}

func (r *downloadRequest) validateExtra() error {
	if len(r.Extra) == 0 {
		return nil
	}
	if _, isBTExtra := r.Extra["trackers"]; isBTExtra {
		if !isBitTorrentRequest(r.URL) {
			return fmt.Errorf("req.extra contains BitTorrent settings, but req.url is not a magnet or torrent resource")
		}
		return nil
	}

	scheme := strings.ToUpper(strings.SplitN(r.URL, ":", 2)[0])
	if scheme != "HTTP" && scheme != "HTTPS" {
		return fmt.Errorf("req.extra contains HTTP settings, but req.url is not an HTTP or HTTPS URL")
	}
	return nil
}

func isBitTorrentRequest(rawURL string) bool {
	upperURL := strings.ToUpper(rawURL)
	path := strings.SplitN(upperURL, "?", 2)[0]
	return strings.HasPrefix(upperURL, "MAGNET:") ||
		strings.HasSuffix(path, ".TORRENT") ||
		strings.HasPrefix(upperURL, "DATA:APPLICATION/X-BITTORRENT;BASE64,")
}

// downloadOptions is the MCP-facing form of base.Options. Only HTTP currently
// defines protocol-specific task options, so Extra has one concrete schema.
type downloadOptions struct {
	Name          string         `json:"name,omitempty" jsonschema:"Optional output file or directory name"`
	Path          string         `json:"path,omitempty" jsonschema:"Optional download directory"`
	AsDefaultPath bool           `json:"asDefaultPath,omitempty" jsonschema:"Save path as the default download directory after task creation"`
	SelectFiles   []int          `json:"selectFiles,omitempty" jsonschema:"Optional zero-based file indexes to download from a multi-file resource"`
	Extra         *httpTaskExtra `json:"extra,omitempty" jsonschema:"Optional HTTP task settings"`
}

func requestExtraSchema() *jsonschema.Schema {
	httpSchema, err := jsonschema.For[httpRequestExtra](nil)
	if err != nil {
		panic(fmt.Sprintf("mcpserver: build HTTP request extra schema: %v", err))
	}
	httpSchema.Title = "HTTP request settings"
	httpSchema.Description = "Protocol-specific request settings for HTTP and HTTPS downloads. Include at least one setting."
	httpSchema.AnyOf = []*jsonschema.Schema{
		{Required: []string{"method"}},
		{Required: []string{"header"}},
		{Required: []string{"body"}},
	}

	btSchema, err := jsonschema.For[btRequestExtra](nil)
	if err != nil {
		panic(fmt.Sprintf("mcpserver: build BitTorrent request extra schema: %v", err))
	}
	btSchema.Title = "BitTorrent request settings"
	btSchema.Description = "Protocol-specific request settings for magnet and BitTorrent resources."
	// A slice is nullable in Go, but an explicitly supplied trackers property
	// should be an array in the MCP contract.
	btSchema.Properties["trackers"].Types = nil
	btSchema.Properties["trackers"].Type = "array"

	return &jsonschema.Schema{
		Type:        "object",
		Description: "Protocol-specific request settings. Must match exactly one of the HTTP or BitTorrent schemas.",
		OneOf:       []*jsonschema.Schema{httpSchema, btSchema},
	}
}

func downloadInputSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[requestExtra](): requestExtraSchema(),
		},
	})
	if err != nil {
		panic(fmt.Sprintf("mcpserver: build tool input schema: %v", err))
	}
	return schema
}

func (o *downloadOptions) baseOptions() *base.Options {
	if o == nil {
		return nil
	}
	return &base.Options{
		Name:          o.Name,
		Path:          o.Path,
		AsDefaultPath: o.AsDefaultPath,
		SelectFiles:   o.SelectFiles,
		Extra:         o.Extra,
	}
}

type resolveTaskInput struct {
	Req  *downloadRequest `json:"req" jsonschema:"Download request to resolve"`
	Opts *downloadOptions `json:"opts,omitempty" jsonschema:"Optional download settings used while resolving"`
}

type resolveTaskOutput struct {
	ID  string         `json:"id,omitempty"`
	Res *base.Resource `json:"res"`
}

type createTaskInput struct {
	RID  string           `json:"rid,omitempty" jsonschema:"Optional resource ID returned by resolve_task"`
	Req  *downloadRequest `json:"req,omitempty" jsonschema:"Direct download request; required when rid is omitted"`
	Opts *downloadOptions `json:"opts,omitempty" jsonschema:"Optional task settings"`
}

type createTaskOutput struct {
	ID string `json:"id"`
}

type taskIDInput struct {
	ID string `json:"id" jsonschema:"Gopeed task ID"`
}

type deleteTaskInput struct {
	ID    string `json:"id" jsonschema:"Gopeed task ID"`
	Force bool   `json:"force,omitempty" jsonschema:"Also delete downloaded files from disk"`
}

type listTasksInput struct {
	IDs         []string `json:"id,omitempty" jsonschema:"Only return tasks with these IDs"`
	Statuses    []string `json:"status,omitempty" jsonschema:"Only return tasks in these states: ready, running, pause, wait, error, or done"`
	NotStatuses []string `json:"notStatus,omitempty" jsonschema:"Exclude tasks in these states: ready, running, pause, wait, error, or done"`
}

type TaskSummary struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Protocol        string                 `json:"protocol"`
	Status          base.Status            `json:"status"`
	Uploading       bool                   `json:"uploading"`
	Downloaded      int64                  `json:"downloaded"`
	Total           int64                  `json:"total"`
	Speed           int64                  `json:"speed"`
	Uploaded        int64                  `json:"uploaded"`
	UploadSpeed     int64                  `json:"uploadSpeed"`
	ExtractStatus   download.ExtractStatus `json:"extractStatus,omitempty"`
	ExtractProgress int                    `json:"extractProgress,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type taskDetails struct {
	TaskSummary
	Req  *base.Request  `json:"req,omitempty"`
	Res  *base.Resource `json:"res,omitempty"`
	Opts *base.Options  `json:"opts,omitempty"`
}

type listTasksOutput struct {
	Tasks []TaskSummary `json:"tasks"`
}

type getTaskOutput struct {
	Task taskDetails `json:"task"`
}

type getTaskStatusOutput struct {
	Status *download.TaskRuntimeStatus `json:"status"`
}

type getTaskStatsOutput struct {
	Stats any `json:"stats"`
}

type taskActionOutput struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

func registerTools(server *mcp.Server, downloader *download.Downloader) {
	closedWorld := false
	openWorld := true
	nondestructive := false
	destructive := true

	mcp.AddTool(server, &mcp.Tool{
		Name:        "resolve_task",
		Title:       "Resolve task",
		Description: "Resolve an HTTP, HTTPS, BitTorrent, magnet, or ED2K download request and return its resource metadata and files. Use this before create_task when the user wants to inspect the resource or select files.",
		InputSchema: downloadInputSchema[resolveTaskInput](),
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *resolveTaskInput) (*mcp.CallToolResult, *resolveTaskOutput, error) {
		if input.Req == nil {
			return nil, nil, fmt.Errorf("req is required")
		}
		req, err := input.Req.baseRequest()
		if err != nil {
			return nil, nil, err
		}
		result, err := downloader.Resolve(req, input.Opts.baseOptions())
		if err != nil {
			return nil, nil, err
		}
		return nil, &resolveTaskOutput{ID: result.ID, Res: result.Res}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_task",
		Title:       "Create task",
		Description: "Create and start a Gopeed download task. Use this when the user asks to download an HTTP or HTTPS URL, a BitTorrent torrent, a magnet URI, or an ED2K link. Accepts either a resource ID from resolve_task or a direct request.",
		InputSchema: downloadInputSchema[createTaskInput](),
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &nondestructive, OpenWorldHint: &openWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *createTaskInput) (*mcp.CallToolResult, *createTaskOutput, error) {
		var (
			id  string
			err error
		)
		switch {
		case input.RID != "":
			id, err = downloader.Create(input.RID)
		case input.Req != nil:
			var req *base.Request
			req, err = input.Req.baseRequest()
			if err == nil {
				id, err = downloader.CreateDirect(req, input.Opts.baseOptions())
			}
		default:
			return nil, nil, fmt.Errorf("rid or req is required")
		}
		if err != nil {
			return nil, nil, err
		}
		return nil, &createTaskOutput{ID: id}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Title:       "List tasks",
		Description: "List Gopeed tasks, optionally filtered by ID or task status.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closedWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *listTasksInput) (*mcp.CallToolResult, *listTasksOutput, error) {
		statuses, err := parseStatuses(input.Statuses)
		if err != nil {
			return nil, nil, err
		}
		notStatuses, err := parseStatuses(input.NotStatuses)
		if err != nil {
			return nil, nil, err
		}
		tasks := downloader.GetTasksByFilter(&download.TaskFilter{
			IDs:         input.IDs,
			Statuses:    statuses,
			NotStatuses: notStatuses,
		})
		result := make([]TaskSummary, 0, len(tasks))
		for _, task := range tasks {
			summary, err := summarizeTask(downloader, task)
			if err != nil {
				return nil, nil, err
			}
			result = append(result, summary)
		}
		return nil, &listTasksOutput{Tasks: result}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task",
		Title:       "Get task",
		Description: "Get one Gopeed task including its request, resource, options, and current progress.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closedWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *taskIDInput) (*mcp.CallToolResult, *getTaskOutput, error) {
		task, err := requireTask(downloader, input.ID)
		if err != nil {
			return nil, nil, err
		}
		summary, err := summarizeTask(downloader, task)
		if err != nil {
			return nil, nil, err
		}
		details := taskDetails{TaskSummary: summary}
		if task.Meta != nil {
			details.Req = task.Meta.Req
			details.Res = task.Meta.Res
			details.Opts = task.Meta.Opts
		}
		return nil, &getTaskOutput{Task: details}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task_status",
		Title:       "Get task status",
		Description: "Get the lightweight runtime status and per-file progress for one Gopeed task.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closedWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *taskIDInput) (*mcp.CallToolResult, *getTaskStatusOutput, error) {
		if err := validateTaskID(input.ID); err != nil {
			return nil, nil, err
		}
		status, err := downloader.RuntimeStatus(input.ID)
		if err != nil {
			return nil, nil, err
		}
		return nil, &getTaskStatusOutput{Status: status}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task_stats",
		Title:       "Get task statistics",
		Description: "Get protocol-specific statistics for one Gopeed task, such as HTTP connections or BitTorrent peers and seeding data.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closedWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *taskIDInput) (*mcp.CallToolResult, *getTaskStatsOutput, error) {
		if err := validateTaskID(input.ID); err != nil {
			return nil, nil, err
		}
		stats, err := downloader.Stats(input.ID)
		if err != nil {
			return nil, nil, err
		}
		return nil, &getTaskStatsOutput{Stats: stats}, nil
	})

	registerTaskAction(server, downloader, "pause_task", "Pause task", "Pause one Gopeed task.", &nondestructive, &closedWorld, downloader.Pause)
	registerTaskAction(server, downloader, "continue_task", "Continue task", "Continue a paused or failed Gopeed task.", &nondestructive, &openWorld, downloader.Continue)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_task",
		Title:       "Delete task",
		Description: "Delete one Gopeed task. Downloaded files are preserved unless force is true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: true, OpenWorldHint: &closedWorld},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *deleteTaskInput) (*mcp.CallToolResult, *taskActionOutput, error) {
		if _, err := requireTask(downloader, input.ID); err != nil {
			return nil, nil, err
		}
		if err := downloader.Delete(&download.TaskFilter{IDs: []string{input.ID}}, input.Force); err != nil {
			return nil, nil, err
		}
		return nil, &taskActionOutput{ID: input.ID, Success: true}, nil
	})
}

func registerTaskAction(
	server *mcp.Server,
	downloader *download.Downloader,
	name string,
	title string,
	description string,
	destructiveHint *bool,
	openWorldHint *bool,
	action func(*download.TaskFilter) error,
) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: destructiveHint,
			IdempotentHint:  true,
			OpenWorldHint:   openWorldHint,
		},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input *taskIDInput) (*mcp.CallToolResult, *taskActionOutput, error) {
		if _, err := requireTask(downloader, input.ID); err != nil {
			return nil, nil, err
		}
		if err := action(&download.TaskFilter{IDs: []string{input.ID}}); err != nil {
			return nil, nil, err
		}
		return nil, &taskActionOutput{ID: input.ID, Success: true}, nil
	})
}

func requireTask(downloader *download.Downloader, id string) (*download.Task, error) {
	if err := validateTaskID(id); err != nil {
		return nil, err
	}
	task := downloader.GetTask(id)
	if task == nil {
		return nil, fmt.Errorf("task %q not found", id)
	}
	return task, nil
}

func validateTaskID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

func summarizeTask(downloader *download.Downloader, task *download.Task) (TaskSummary, error) {
	status, err := downloader.RuntimeStatus(task.ID)
	if err != nil {
		return TaskSummary{}, err
	}
	summary := TaskSummary{
		ID:              task.ID,
		Protocol:        task.Protocol,
		Status:          status.Status,
		Uploading:       task.Uploading,
		Downloaded:      status.Downloaded,
		Total:           status.Total,
		Speed:           status.Speed,
		Uploaded:        status.Uploaded,
		UploadSpeed:     status.UploadSpeed,
		ExtractStatus:   status.ExtractStatus,
		ExtractProgress: status.ExtractProgress,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
	if task.Meta != nil && task.Meta.Res != nil && len(task.Meta.Res.Files) > 0 {
		summary.Name = task.Name()
	}
	return summary, nil
}

func parseStatuses(values []string) ([]base.Status, error) {
	statuses := make([]base.Status, 0, len(values))
	for _, value := range values {
		status := base.Status(value)
		switch status {
		case base.DownloadStatusReady, base.DownloadStatusRunning, base.DownloadStatusPause,
			base.DownloadStatusWait, base.DownloadStatusError, base.DownloadStatusDone:
			statuses = append(statuses, status)
		default:
			return nil, fmt.Errorf("invalid task status %q", value)
		}
	}
	return statuses, nil
}
