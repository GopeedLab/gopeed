package ffmpeg

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/GopeedLab/gopeed/internal/httpclient"
	media "github.com/GopeedLab/gopeed/pkg/download/engine/ffmpeg"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

//go:embed ffmpeg.js
var script string

type Config struct {
	TempDir          string
	DefaultUserAgent *string
	ProxyHandler     func(*http.Request) (*url.URL, error)
	RegisterCleanup  func(func())
}
type options struct {
	Video  *media.HTTPSource `json:"video"`
	Audio  *media.HTTPSource `json:"audio"`
	Format string            `json:"format"`
	Args   []string          `json:"args"`
}
type job struct {
	ctx     context.Context
	cancel  context.CancelFunc
	output  *io.PipeReader
	streams [2]*media.StreamInput
}

func (j *job) stop() {
	j.cancel()
	j.output.CloseWithError(context.Canceled)
	for _, s := range j.streams {
		if s != nil {
			s.Close()
		}
	}
}

func Enable(vm *goja.Runtime, loop *eventloop.EventLoop, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	client, err := httpclient.NewClient(httpclient.Options{Transport: httpclient.TransportOptions{Proxy: cfg.ProxyHandler}})
	if err != nil {
		return err
	}
	var mu sync.Mutex
	jobs := map[string]*job{}
	var next atomic.Uint64
	get := func(id string) *job { mu.Lock(); defer mu.Unlock(); return jobs[id] }
	remove := func(id string) {
		mu.Lock()
		j := jobs[id]
		delete(jobs, id)
		mu.Unlock()
		if j != nil {
			j.stop()
		}
	}
	if cfg.RegisterCleanup != nil {
		cfg.RegisterCleanup(func() {
			done := make(chan struct{})
			if loop.RunOnLoop(func(vm *goja.Runtime) {
				defer close(done)
				if closeAll, ok := goja.AssertFunction(vm.Get("__gopeed_ffmpeg_close_all")); ok {
					_, _ = closeAll(goja.Undefined())
				}
			}) {
				<-done
			}
			mu.Lock()
			old := jobs
			jobs = map[string]*job{}
			mu.Unlock()
			for _, j := range old {
				j.stop()
			}
			client.CloseIdleConnections()
		})
	}
	if err := vm.Set("__gopeed_ffmpeg_create", func(call goja.FunctionCall) goja.Value {
		var opts options
		if err := vm.ExportTo(call.Argument(0), &opts); err != nil {
			panic(vm.NewGoError(err))
		}
		if _, err := media.Arguments(opts.Format, opts.Args); err != nil {
			panic(vm.NewGoError(err))
		}
		for _, source := range []*media.HTTPSource{opts.Video, opts.Audio} {
			if source == nil || cfg.DefaultUserAgent == nil {
				continue
			}
			specified := false
			for key := range source.Headers {
				if strings.EqualFold(key, "User-Agent") {
					specified = true
					break
				}
			}
			if !specified {
				if source.Headers == nil {
					source.Headers = make(map[string]string)
				}
				source.Headers["User-Agent"] = *cfg.DefaultUserAgent
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		r, w := io.Pipe()
		j := &job{ctx: ctx, cancel: cancel, output: r}
		for i, source := range []*media.HTTPSource{opts.Video, opts.Audio} {
			if source == nil {
				j.streams[i] = media.NewStreamInput(ctx, cfg.TempDir)
			}
		}
		id := fmt.Sprintf("ffmpeg-%d", next.Add(1))
		mu.Lock()
		jobs[id] = j
		mu.Unlock()
		go func() {
			var inputs [2]media.Input
			var runErr error
			defer func() {
				if v := recover(); v != nil {
					runErr = fmt.Errorf("ffmpeg panic: %v", v)
				}
				cancel()
				for _, in := range inputs {
					if in != nil {
						in.Close()
					}
				}
				// Closing only after Run returns prevents partial output from
				// being mistaken for a successful download on FFmpeg failure.
				w.CloseWithError(runErr)
			}()
			for i, source := range []*media.HTTPSource{opts.Video, opts.Audio} {
				if source == nil {
					inputs[i] = j.streams[i]
				} else {
					inputs[i], runErr = media.OpenHTTP(ctx, client, *source)
					if runErr != nil {
						return
					}
				}
			}
			runErr = media.Run(ctx, inputs[0], inputs[1], w, opts.Format, opts.Args)
		}()
		return vm.ToValue(id)
	}); err != nil {
		return err
	}
	if err := vm.Set("__gopeed_ffmpeg_read", func(id string) goja.Value {
		promise, resolve, reject := vm.NewPromise()
		go func() {
			var readErr error
			buf := make([]byte, 64*1024)
			n := 0
			j := get(id)
			if j == nil {
				readErr = io.ErrClosedPipe
			} else {
				n, readErr = j.output.Read(buf)
			}
			if !loop.RunOnLoop(func(vm *goja.Runtime) {
				if readErr != nil {
					remove(id)
					if readErr == io.EOF {
						resolve(goja.Null())
					} else {
						reject(vm.NewGoError(readErr))
					}
					return
				}
				resolve(vm.ToValue(vm.NewArrayBuffer(buf[:n])))
			}) {
				remove(id)
			}
		}()
		return vm.ToValue(promise)
	}); err != nil {
		return err
	}
	if err := vm.Set("__gopeed_ffmpeg_push", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		index := int(call.Argument(1).ToInteger())
		var chunk []byte
		switch b := call.Argument(2).Export().(type) {
		case goja.ArrayBuffer:
			chunk = append([]byte(nil), b.Bytes()...)
		case []byte:
			chunk = append([]byte(nil), b...)
		default:
			panic(vm.NewTypeError("ffmpeg input requires bytes"))
		}
		promise, resolve, reject := vm.NewPromise()
		go func() {
			var err error
			j := get(id)
			if j == nil || index < 0 || index > 1 || j.streams[index] == nil {
				err = io.ErrClosedPipe
			} else {
				_, err = j.streams[index].Write(chunk)
			}
			if err != nil && j != nil && index >= 0 && index < 2 && j.streams[index] != nil {
				j.streams[index].End(err)
			}
			loop.RunOnLoop(func(vm *goja.Runtime) {
				if j != nil && j.ctx.Err() != nil {
					resolve(false)
					return
				}
				if err != nil {
					reject(vm.NewGoError(err))
				} else {
					resolve(true)
				}
			})
		}()
		return vm.ToValue(promise)
	}); err != nil {
		return err
	}
	if err := vm.Set("__gopeed_ffmpeg_end", func(id string, index int, message string) {
		if j := get(id); j != nil && index >= 0 && index < 2 && j.streams[index] != nil {
			var err error
			if message != "" {
				err = errors.New(message)
			}
			j.streams[index].End(err)
		}
	}); err != nil {
		return err
	}
	if err := vm.Set("__gopeed_ffmpeg_cancel", remove); err != nil {
		return err
	}
	_, err = vm.RunString(script)
	return err
}
