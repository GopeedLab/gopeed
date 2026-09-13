package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"

	ffembed "codeberg.org/gruf/go-ffmpreg/embed"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
	expsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/experimental/sysfs"
	"github.com/tetratelabs/wazero/sys"
)

var shared struct {
	once    sync.Once
	runtime wazero.Runtime
	module  wazero.CompiledModule
	err     error
}

// Limit active instances independently of the number of extension engines.
var slots = make(chan struct{}, 2)

func initialize() {
	shared.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				shared.err = fmt.Errorf("ffmpeg initialization: %v", r)
			}
		}()
		cfg := wazero.NewRuntimeConfig().WithCloseOnContextDone(true).WithMemoryLimitPages(8192)
		shared.runtime, shared.err = wasm.NewRuntime(context.Background(), cfg)
		if shared.err != nil {
			return
		}
		shared.module, shared.err = shared.runtime.CompileModule(context.Background(), ffembed.B())
		if shared.err != nil {
			shared.runtime.Close(context.Background())
			return
		}
		ffembed.Free()
	})
}

// Output arguments deliberately do not accept extra inputs, output paths or
// global filesystem/logging options. The host owns all I/O and cancellation.
var outputOptions = map[string]int{
	"-c": 1, "-codec": 1, "-c:v": 1, "-c:a": 1, "-codec:v": 1, "-codec:a": 1,
	"-map": 1, "-map_metadata": 1, "-map_chapters": 1, "-metadata": 1, "-metadata:s:a:0": 1, "-metadata:s:v:0": 1,
	"-movflags": 1, "-frag_duration": 1, "-frag_size": 1, "-min_frag_duration": 1,
	"-bsf:v": 1, "-bsf:a": 1, "-strict": 1, "-shortest": 0, "-t": 1, "-to": 1,
	"-avoid_negative_ts": 1, "-max_interleave_delta": 1, "-flush_packets": 1,
	"-disposition:a:0": 1, "-disposition:v:0": 1,
}

func Arguments(format string, extra []string) ([]string, error) {
	if format == "" {
		format = "mp4"
	}
	if format == "mkv" {
		format = "matroska"
	}
	switch format {
	case "mp4", "matroska", "webm", "mpegts":
	default:
		return nil, fmt.Errorf("ffmpeg: unsupported output format %q", format)
	}
	for i := 0; i < len(extra); {
		count, ok := outputOptions[extra[i]]
		if !ok {
			return nil, fmt.Errorf("ffmpeg: unsupported output option %q", extra[i])
		}
		if i+count >= len(extra) {
			return nil, fmt.Errorf("ffmpeg: missing value for %s", extra[i])
		}
		if strings.HasPrefix(extra[i], "-c") && extra[i+1] != "copy" {
			return nil, errors.New("ffmpeg: merge only supports stream copy")
		}
		i += count + 1
	}
	args := []string{"-hide_banner", "-nostdin", "-xerror", "-loglevel", "error", "-protocol_whitelist", "file,pipe", "-i", "/inputs/video", "-protocol_whitelist", "file,pipe", "-i", "/inputs/audio", "-map", "0:v:0", "-map", "1:a:0", "-c", "copy"}
	fragmentFlags := ""
	for i := 0; i < len(extra); {
		count := outputOptions[extra[i]]
		if format == "mp4" && extra[i] == "-movflags" {
			if strings.Contains(extra[i+1], "faststart") || strings.Contains(extra[i+1], "global_sidx") {
				return nil, errors.New("ffmpeg: these movflags require seekable output")
			}
			fragmentFlags = extra[i+1]
		} else {
			args = append(args, extra[i:i+count+1]...)
		}
		i += count + 1
	}
	if format == "mp4" {
		args = append(args, "-movflags", fragmentFlags+"+frag_keyframe+empty_moov+default_base_moof")
	}
	args = append(args, "-f", format, "pipe:1")
	return args, nil
}

// Run shares compilation, but gives every merge its own memory and filesystem.
// A clean stdout EOF is only delivered by the caller after this succeeds.
func Run(ctx context.Context, video, audio Input, out io.Writer, format string, extra []string) error {
	args, err := Arguments(format, extra)
	if err != nil {
		return err
	}
	select {
	case slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-slots }()
	ready := make(chan struct{})
	go func() { initialize(); close(ready) }()
	select {
	case <-ready:
	case <-ctx.Done():
		return ctx.Err()
	}
	if shared.err != nil {
		return shared.err
	}
	files := &inputFS{files: map[string]Input{"video": video, "audio": audio}}
	fsconfig := wazero.NewFSConfig().(sysfs.FSConfig).WithSysFSMount(files, "/inputs")
	logs := &tailLog{}
	rc, err := wasm.Run(ctx, shared.runtime, shared.module, wasm.Args{
		Name: "ffmpeg", Args: args, Stdout: out, Stderr: logs,
		Config: func(c wazero.ModuleConfig) wazero.ModuleConfig { return c.WithFSConfig(fsconfig) },
	})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if files.err != nil {
		return fmt.Errorf("ffmpeg input: %w (%s)", files.err, logs.String())
	}
	if err != nil {
		return fmt.Errorf("ffmpeg runtime: %w (%s)", err, logs.String())
	}
	if rc != 0 {
		return fmt.Errorf("ffmpeg exited with code %d: %s", rc, logs.String())
	}
	return nil
}

type tailLog struct{ data []byte }

func (l *tailLog) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) > 8192 {
		p = p[len(p)-8192:]
	}
	l.data = append(l.data, p...)
	if len(l.data) > 8192 {
		l.data = l.data[len(l.data)-8192:]
	}
	return n, nil
}
func (l *tailLog) String() string { return strings.TrimSpace(string(l.data)) }

type inputFS struct {
	expsys.UnimplementedFS
	files map[string]Input
	err   error
}

func (f *inputFS) OpenFile(path string, flag expsys.Oflag, perm fs.FileMode) (expsys.File, expsys.Errno) {
	if flag&(expsys.O_WRONLY|expsys.O_RDWR|expsys.O_CREAT|expsys.O_TRUNC) != 0 {
		return nil, expsys.EROFS
	}
	path = strings.TrimPrefix(path, "/")
	if path == "." || path == "" {
		return &inputFile{owner: f}, 0
	}
	in := f.files[path]
	if in == nil {
		return nil, expsys.ENOENT
	}
	return &inputFile{owner: f, in: in}, 0
}
func (f *inputFS) Stat(path string) (sys.Stat_t, expsys.Errno) {
	file, err := f.OpenFile(path, 0, 0)
	if err != 0 {
		return sys.Stat_t{}, err
	}
	return file.Stat()
}

type inputFile struct {
	expsys.UnimplementedFile
	owner *inputFS
	in    Input
}

func (f *inputFile) IsDir() (bool, expsys.Errno) { return f.in == nil, 0 }
func (f *inputFile) Stat() (sys.Stat_t, expsys.Errno) {
	if f.in == nil {
		return sys.Stat_t{Mode: fs.ModeDir | 0555}, 0
	}
	mode := fs.FileMode(0444)
	if !f.in.Seekable() {
		mode |= fs.ModeNamedPipe
	}
	return sys.Stat_t{Mode: mode, Size: max(0, f.in.Size()), Nlink: 1}, 0
}
func (f *inputFile) Read(p []byte) (int, expsys.Errno) {
	if f.in == nil {
		return 0, expsys.EISDIR
	}
	n, err := f.in.Read(p)
	if err != nil && err != io.EOF {
		f.owner.err = err
		return n, expsys.EIO
	}
	return n, 0
}
func (f *inputFile) Seek(off int64, whence int) (int64, expsys.Errno) {
	if f.in == nil {
		return 0, expsys.EISDIR
	}
	n, err := f.in.Seek(off, whence)
	// Capability probes may legitimately fail; a failed seek alone is not a
	// media error. FFmpeg decides whether it can continue sequentially.
	if err != nil {
		return 0, expsys.ENOSYS
	}
	return n, 0
}
