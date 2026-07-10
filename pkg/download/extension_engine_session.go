package download

import (
	"context"
	"io"
	"sync"

	internalblob "github.com/GopeedLab/gopeed/internal/blob"
	"github.com/GopeedLab/gopeed/pkg/download/engine"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/stream"
	enginewebview "github.com/GopeedLab/gopeed/pkg/download/engine/webview"
)

type engineSession struct {
	engine *engine.Engine

	mu     sync.Mutex
	refs   int
	closed bool
	close  []func()
}

func newEngineSession(e *engine.Engine) *engineSession {
	return &engineSession{engine: e}
}

func (s *engineSession) SetEngine(e *engine.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.engine = e
}

func (s *engineSession) Retain() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.refs++
}

func (s *engineSession) Release() {
	s.mu.Lock()
	if s.refs > 0 {
		s.refs--
	}
	shouldClose := s.refs == 0 && !s.closed
	if shouldClose {
		s.closed = true
	}
	s.mu.Unlock()
	if shouldClose {
		s.runClosers()
		if s.engine != nil {
			go s.engine.Close()
		}
	}
}

func (s *engineSession) CloseIfIdle() {
	s.mu.Lock()
	shouldClose := s.refs == 0 && !s.closed
	if shouldClose {
		s.closed = true
	}
	s.mu.Unlock()
	if shouldClose {
		s.runClosers()
		if s.engine != nil {
			s.engine.Close()
		}
	}
}

func (s *engineSession) OnClose(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		fn()
		return
	}
	s.close = append(s.close, fn)
	s.mu.Unlock()
}

func (s *engineSession) runClosers() {
	s.mu.Lock()
	closeFns := s.close
	s.close = nil
	s.mu.Unlock()
	for _, fn := range closeFns {
		fn()
	}
}

func (d *Downloader) newExtensionEngine() (*engine.Engine, *engineSession) {
	session := newEngineSession(nil)
	engineCfg := &stream.Config{
		CreateObjectURL: func(opts *stream.ObjectURLOptions, open stream.ObjectURLOpener) (string, error) {
			createOpts := &internalblob.CreateOptions{
				Session: session,
			}
			if opts != nil {
				createOpts.ContentType = opts.ContentType
				createOpts.Size = opts.Size
				createOpts.Range = opts.Range
			}
			return d.blob.CreateOpener(func(ctx context.Context, req internalblob.OpenRequest) (io.ReadCloser, error) {
				return open(ctx, stream.ObjectURLOpenRequest{
					Offset: req.Offset,
					End:    req.End,
				})
			}, createOpts)
		},
		RevokeObjectURL: func(url string) error {
			return d.blob.Revoke(url)
		},
		ProxyHandler:    d.cfg.Proxy.ToHandler(),
		RegisterCleanup: session.OnClose,
	}
	e := engine.NewEngine(&engine.Config{
		ProxyConfig:  d.cfg.Proxy,
		StreamConfig: engineCfg,
	})
	session.SetEngine(e)
	return e, session
}

func (d *Downloader) newExtensionWebViewRuntime(session *engineSession) *enginewebview.Runtime {
	var (
		opener    enginewebview.Opener
		available bool
	)
	if provider := d.cfg.WebViewProvider; provider != nil && provider.IsAvailable() {
		opener = provider
		available = true
	}
	runtime := enginewebview.NewRuntime(opener, available)
	session.OnClose(func() {
		_ = runtime.Close()
	})
	return runtime
}
