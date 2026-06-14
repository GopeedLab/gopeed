package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GopeedLab/gopeed/internal/webview/goprovider"
	"github.com/GopeedLab/gopeed/internal/webview/rpcprovider"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

type kvFlag map[string]string

type webViewConfig struct {
	provider string
	network  string
	address  string
	token    string
}

func (l *kvFlag) String() string {
	if l == nil {
		return ""
	}
	data, _ := json.Marshal(map[string]string(*l))
	return string(data)
}

func (l *kvFlag) Set(value string) error {
	if *l == nil {
		*l = make(map[string]string)
	}
	idx := -1
	for i := 0; i < len(value); i++ {
		if value[i] == '=' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return fmt.Errorf("invalid label %q, expected key=value", value)
	}
	(*l)[value[:idx]] = value[idx+1:]
	return nil
}

func main() {
	os.Exit(goprovider.RunMainThreadLoop(run))
}

func run() int {
	if len(os.Args) < 2 {
		usage()
		return 2
	}

	switch os.Args[1] {
	case "resolve":
		if err := runResolve(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	case "download":
		if err := runDownload(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	case "eval":
		if err := runEval(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		usage()
		return 2
	}
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `gopeed extmock CLI

Usage:
  extmock resolve [flags] <extension-dir> <url>
  extmock download [flags] <extension-dir> <url> <output-dir>
  extmock eval [flags] [script-file]

Flags:
  -label key=value   Request label, repeatable
  -setting key=value Extension setting override, repeatable
  -webview-provider  WebView provider: go|rpc
  -webview-network   RPC network when provider=rpc: unix|tcp
  -webview-address   RPC address when provider=rpc
  -webview-token     RPC bearer token when provider=rpc
  -verb              Print download progress while running
  -pretty            Pretty-print JSON output

Examples:
  extmock resolve ./ https://www.google.com
  extmock download ./ https://example.com/video /tmp/extmock-download
  extmock resolve ./ https://example.com/video -label env=test -setting token=abc
  extmock eval ./test.js
  extmock eval -e "new Blob(['hello']).size"
`)
}

func runResolve(args []string) error {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var labels kvFlag
	var settings kvFlag
	var pretty bool
	cfg := webViewConfig{}
	fs.Var(&labels, "label", "Request label in key=value form (repeatable)")
	fs.Var(&settings, "setting", "Extension setting override in key=value form (repeatable)")
	fs.StringVar(&cfg.provider, "webview-provider", "", "WebView provider: go|rpc")
	fs.StringVar(&cfg.network, "webview-network", "", "WebView RPC network: unix|tcp")
	fs.StringVar(&cfg.address, "webview-address", "", "WebView RPC address")
	fs.StringVar(&cfg.token, "webview-token", "", "WebView RPC bearer token")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")

	flagArgs, positionalArgs, err := splitResolveArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected trailing args: %v", fs.Args())
	}
	if len(positionalArgs) != 2 {
		return fmt.Errorf("resolve expects exactly 2 args: <extension-dir> <url>")
	}

	extDir, err := filepath.Abs(positionalArgs[0])
	if err != nil {
		return err
	}
	url := positionalArgs[1]

	d, cleanup, err := newExtMockDownloader(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	ext, err := d.InstallExtensionByFolder(extDir, true)
	if err != nil {
		return err
	}
	if len(settings) > 0 {
		if err := d.UpdateExtensionSettings(ext.Identity, toAnyMap(settings)); err != nil {
			return err
		}
	}
	result, err := d.Resolve(&base.Request{
		URL:    url,
		Labels: map[string]string(labels),
	}, nil)
	if err != nil {
		return err
	}
	return printJSON(result, pretty)
}

type downloadResult struct {
	Resolve   *download.ResolveResult `json:"resolve"`
	OutputDir string                  `json:"outputDir"`
	Files     []*downloadedFileResult `json:"files"`
}

type downloadedFileResult struct {
	Index  int           `json:"index"`
	Name   string        `json:"name"`
	Path   string        `json:"path"`
	TaskID string        `json:"taskId"`
	Status base.Status   `json:"status"`
	Size   int64         `json:"size"`
	Req    *base.Request `json:"req,omitempty"`
}

func runDownload(args []string) error {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var labels kvFlag
	var settings kvFlag
	var pretty bool
	var verbose bool
	cfg := webViewConfig{}
	fs.Var(&labels, "label", "Request label in key=value form (repeatable)")
	fs.Var(&settings, "setting", "Extension setting override in key=value form (repeatable)")
	fs.StringVar(&cfg.provider, "webview-provider", "", "WebView provider: go|rpc")
	fs.StringVar(&cfg.network, "webview-network", "", "WebView RPC network: unix|tcp")
	fs.StringVar(&cfg.address, "webview-address", "", "WebView RPC address")
	fs.StringVar(&cfg.token, "webview-token", "", "WebView RPC bearer token")
	fs.BoolVar(&verbose, "verb", false, "Print download progress while running")
	fs.BoolVar(&verbose, "verbose", false, "Print download progress while running")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")

	flagArgs, positionalArgs, err := splitDownloadArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected trailing args: %v", fs.Args())
	}
	if len(positionalArgs) != 3 {
		return fmt.Errorf("download expects exactly 3 args: <extension-dir> <url> <output-dir>")
	}

	extDir, err := filepath.Abs(positionalArgs[0])
	if err != nil {
		return err
	}
	url := positionalArgs[1]
	outputDir, err := filepath.Abs(positionalArgs[2])
	if err != nil {
		return err
	}

	d, cleanup, err := newExtMockDownloader(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	ext, err := d.InstallExtensionByFolder(extDir, true)
	if err != nil {
		return err
	}
	if len(settings) > 0 {
		if err := d.UpdateExtensionSettings(ext.Identity, toAnyMap(settings)); err != nil {
			return err
		}
	}

	result, err := d.Resolve(&base.Request{
		URL:    url,
		Labels: map[string]string(labels),
	}, nil)
	if err != nil {
		return err
	}
	if result == nil || result.Res == nil {
		return fmt.Errorf("resolve returned empty resource")
	}
	if len(result.Res.Files) == 0 {
		return fmt.Errorf("resolve returned no files")
	}

	resourceOutputDir := outputDir
	if result.Res.Name != "" {
		resourceOutputDir = filepath.Join(outputDir, result.Res.Name)
	}
	if err := os.MkdirAll(resourceOutputDir, 0o755); err != nil {
		return err
	}

	waiter := newTaskTerminalWaiter(d)
	summary := &downloadResult{
		Resolve:   result,
		OutputDir: resourceOutputDir,
		Files:     make([]*downloadedFileResult, 0, len(result.Res.Files)),
	}

	for idx, file := range result.Res.Files {
		if file == nil {
			return fmt.Errorf("file %d is nil", idx)
		}
		if file.Req == nil {
			return fmt.Errorf("file %d request is nil", idx)
		}

		targetDir := resourceOutputDir
		if file.Path != "" {
			targetDir = filepath.Join(resourceOutputDir, file.Path)
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}

		taskID, err := d.CreateDirect(file.Req, &base.Options{
			Path: targetDir,
			Name: file.Name,
		})
		if err != nil {
			return fmt.Errorf("create task for %q failed: %w", file.Name, err)
		}
		waiter.Register(taskID)

		summary.Files = append(summary.Files, &downloadedFileResult{
			Index:  idx,
			Name:   file.Name,
			Path:   filepath.Join(targetDir, file.Name),
			TaskID: taskID,
			Req:    file.Req,
		})
	}

	var stopVerbose func()
	if verbose {
		stopVerbose = startVerboseTaskListener(d, summary.Files)
		defer stopVerbose()
	}

	for _, file := range summary.Files {
		task, err := waiter.Wait(file.TaskID, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("download %q failed: %w", file.Name, err)
		}
		file.Status = task.Status
		if info, statErr := os.Stat(file.Path); statErr == nil {
			file.Size = info.Size()
		}
	}

	return printJSON(summary, pretty)
}

func runEval(args []string) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		script string
		pretty bool
	)
	cfg := webViewConfig{}
	fs.StringVar(&script, "e", "", "Inline JavaScript to execute")
	fs.StringVar(&script, "eval", "", "Inline JavaScript to execute")
	fs.StringVar(&cfg.provider, "webview-provider", "", "WebView provider: go|rpc")
	fs.StringVar(&cfg.network, "webview-network", "", "WebView RPC network: unix|tcp")
	fs.StringVar(&cfg.address, "webview-address", "", "WebView RPC address")
	fs.StringVar(&cfg.token, "webview-token", "", "WebView RPC bearer token")
	fs.BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")

	flagArgs, positionalArgs, err := splitEvalArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected trailing args: %v", fs.Args())
	}
	if script == "" && len(positionalArgs) == 0 {
		return fmt.Errorf("eval expects inline code via -e/--eval or a script file path")
	}
	if script != "" && len(positionalArgs) > 0 {
		return fmt.Errorf("eval accepts either inline code or a script file path, not both")
	}
	if len(positionalArgs) > 1 {
		return fmt.Errorf("eval expects at most 1 script file path")
	}

	d, cleanup, err := newExtMockDownloader(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	runtime, err := d.NewExtensionEngine(newExtMockExtension(), map[string]any{})
	if err != nil {
		return err
	}
	defer runtime.Close()

	var value any
	if script != "" {
		value, err = runtime.Eval(script)
	} else {
		scriptPath, pathErr := filepath.Abs(positionalArgs[0])
		if pathErr != nil {
			return pathErr
		}
		value, err = runtime.EvalFile(scriptPath)
	}
	if err != nil {
		return err
	}
	return printJSON(value, pretty)
}

func splitResolveArgs(args []string) ([]string, []string, error) {
	var flagArgs []string
	var positionalArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-pretty":
			flagArgs = append(flagArgs, arg)
		case arg == "-webview-provider", arg == "-webview-network", arg == "-webview-address", arg == "-webview-token":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case arg == "-label":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case arg == "-setting":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case len(arg) > len("-label=") && arg[:len("-label=")] == "-label=":
			flagArgs = append(flagArgs, arg)
		case len(arg) > len("-setting=") && arg[:len("-setting=")] == "-setting=":
			flagArgs = append(flagArgs, arg)
		case hasFlagValue(arg, "-webview-provider"), hasFlagValue(arg, "-webview-network"),
			hasFlagValue(arg, "-webview-address"), hasFlagValue(arg, "-webview-token"):
			flagArgs = append(flagArgs, arg)
		case len(arg) > 0 && arg[0] == '-':
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs, nil
}

func splitDownloadArgs(args []string) ([]string, []string, error) {
	var flagArgs []string
	var positionalArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-pretty", arg == "-verb", arg == "-verbose":
			flagArgs = append(flagArgs, arg)
		case arg == "-webview-provider", arg == "-webview-network", arg == "-webview-address", arg == "-webview-token":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case arg == "-label":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case arg == "-setting":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case len(arg) > len("-label=") && arg[:len("-label=")] == "-label=":
			flagArgs = append(flagArgs, arg)
		case len(arg) > len("-setting=") && arg[:len("-setting=")] == "-setting=":
			flagArgs = append(flagArgs, arg)
		case hasFlagValue(arg, "-webview-provider"), hasFlagValue(arg, "-webview-network"),
			hasFlagValue(arg, "-webview-address"), hasFlagValue(arg, "-webview-token"):
			flagArgs = append(flagArgs, arg)
		case len(arg) > 0 && arg[0] == '-':
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs, nil
}

func splitEvalArgs(args []string) ([]string, []string, error) {
	var flagArgs []string
	var positionalArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-pretty":
			flagArgs = append(flagArgs, arg)
		case arg == "-webview-provider", arg == "-webview-network", arg == "-webview-address", arg == "-webview-token":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case arg == "-e", arg == "--eval":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag %q requires a value", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case hasFlagValue(arg, "-e"), hasFlagValue(arg, "--eval"),
			hasFlagValue(arg, "-webview-provider"), hasFlagValue(arg, "-webview-network"),
			hasFlagValue(arg, "-webview-address"), hasFlagValue(arg, "-webview-token"):
			flagArgs = append(flagArgs, arg)
		case len(arg) > 0 && arg[0] == '-':
			return nil, nil, fmt.Errorf("unknown flag %q", arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs, nil
}

func hasFlagValue(arg string, prefix string) bool {
	return len(arg) > len(prefix)+1 && arg[:len(prefix)+1] == prefix+"="
}

func toAnyMap(values kvFlag) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func newExtMockDownloader(cfg webViewConfig) (*download.Downloader, func(), error) {
	webViewProvider, err := newExtMockWebViewProvider(cfg)
	if err != nil {
		return nil, nil, err
	}
	d := download.NewDownloader(&download.DownloaderConfig{
		Storage:         download.NewMemStorage(),
		WebViewProvider: webViewProvider,
	})
	if err := d.Setup(); err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = d.Clear()
	}
	return d, cleanup, nil
}

func newExtMockExtension() *download.Extension {
	return &download.Extension{
		Name:    "extmock",
		Author:  "gopeed",
		Title:   "Gopeed ExtMock Runtime",
		Version: "0.0.0",
		DevMode: true,
	}
}

func newExtMockWebViewProvider(cfg webViewConfig) (enginewebview.Provider, error) {
	switch cfg.provider {
	case "", "go":
		return goprovider.New(), nil
	case "rpc":
		rpcCfg, err := resolveRPCConfig(cfg)
		if err != nil {
			return nil, err
		}
		return rpcprovider.New(rpcCfg), nil
	default:
		return nil, fmt.Errorf("unsupported webview provider %q", cfg.provider)
	}
}

func resolveRPCConfig(cfg webViewConfig) (enginewebview.RPCConfig, error) {
	network := cfg.network
	address := cfg.address
	if network == "" && address == "" {
		defaultAddress, err := defaultWebViewRPCUnixSocket()
		if err != nil {
			return enginewebview.RPCConfig{}, err
		}
		network = "unix"
		address = defaultAddress
	}
	if network == "" || address == "" {
		return enginewebview.RPCConfig{}, fmt.Errorf("webview rpc requires both network and address")
	}
	return enginewebview.RPCConfig{
		Network: network,
		Address: address,
		Token:   cfg.token,
	}, nil
}

func defaultWebViewRPCUnixSocket() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "gopeed_webview.sock"), nil
}

type taskTerminalWaiter struct {
	downloader *download.Downloader
}

func newTaskTerminalWaiter(d *download.Downloader) *taskTerminalWaiter {
	return &taskTerminalWaiter{downloader: d}
}

func (w *taskTerminalWaiter) Register(id string) {
}

func (w *taskTerminalWaiter) Wait(id string, timeout time.Duration) (*download.Task, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if task, ok := w.currentTerminalTask(id); ok {
			return task, w.taskError(task)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for task %s", id)
}

func (w *taskTerminalWaiter) currentTerminalTask(id string) (*download.Task, bool) {
	task := w.downloader.GetTask(id)
	if task == nil {
		return nil, false
	}
	if task.Status != base.DownloadStatusDone && task.Status != base.DownloadStatusError {
		return nil, false
	}
	return task, true
}

func (w *taskTerminalWaiter) taskError(task *download.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if task.Status == base.DownloadStatusError {
		return fmt.Errorf("task %s finished with status %s", task.ID, task.Status)
	}
	return nil
}

func startVerboseTaskListener(d *download.Downloader, files []*downloadedFileResult) func() {
	tracked := make(map[string]string, len(files))
	for _, file := range files {
		tracked[file.TaskID] = file.Name
	}
	d.Listener(func(event *download.Event) {
		if event == nil || event.Task == nil {
			return
		}
		name, ok := tracked[event.Task.ID]
		if !ok {
			return
		}
		switch event.Key {
		case download.EventKeyStart, download.EventKeyProgress, download.EventKeyDone, download.EventKeyError:
			printVerboseTaskEvent(event, name)
		}
	})
	return func() {}
}

func printVerboseTaskEvent(event *download.Event, name string) {
	task := event.Task
	ts := time.Now().Format("15:04:05.000")
	if task == nil {
		return
	}
	total := int64(0)
	if task.Meta != nil && task.Meta.Res != nil {
		total = task.Meta.Res.Size
	}
	downloaded := int64(0)
	speed := int64(0)
	if task.Progress != nil {
		downloaded = task.Progress.Downloaded
		speed = task.Progress.Speed
	}
	if event.Key == download.EventKeyError && event.Err != nil {
		fmt.Fprintf(
			os.Stderr,
			"[%s] event=%s task=%s status=%s downloaded=%d total=%d speed=%d name=%q err=%v\n",
			ts,
			event.Key,
			task.ID,
			task.Status,
			downloaded,
			total,
			speed,
			name,
			event.Err,
		)
		return
	}
	fmt.Fprintf(
		os.Stderr,
		"[%s] event=%s task=%s status=%s downloaded=%d total=%d speed=%d name=%q\n",
		ts,
		event.Key,
		task.ID,
		task.Status,
		downloaded,
		total,
		speed,
		name,
	)
}

func printJSON(v any, pretty bool) error {
	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
