package download

import (
	"fmt"
	"sync"

	"github.com/GopeedLab/gopeed/pkg/download/engine"
	"github.com/GopeedLab/gopeed/pkg/download/engine/inject/stream"
	"github.com/dop251/goja"
)

type engineSession struct {
	engine *engine.Engine

	mu     sync.Mutex
	refs   int
	closed bool
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
		if s.engine != nil {
			s.engine.Close()
		}
	}
}

func normalizeStreamChunk(data any) ([]byte, error) {
	switch v := data.(type) {
	case nil:
		return nil, nil
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case goja.ArrayBuffer:
		return v.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported stream chunk type: %T", data)
	}
}

func (d *Downloader) newExtensionEngine() (*engine.Engine, *engineSession) {
	session := newEngineSession(nil)
	engineCfg := &stream.Config{
		CreateBlobObjectURL: func(data []byte, contentType string) (string, error) {
			return d.gblob.CreateBlob(data, contentType)
		},
		CreateWritableStreamObjectURL: func() (string, error) {
			return d.gblob.CreateWritableStream(session)
		},
		WriteWritableStreamObjectURL: func(url string, data any) error {
			chunk, err := normalizeStreamChunk(data)
			if err != nil {
				return err
			}
			return d.gblob.Write(url, chunk)
		},
		CloseWritableStreamObjectURL: func(url string) error {
			return d.gblob.CloseSource(url)
		},
		AbortWritableStreamObjectURL: func(url string, reason string) error {
			return d.gblob.AbortSource(url, fmt.Errorf("%s", reason))
		},
		RevokeObjectURL: func(url string) error {
			return d.gblob.Revoke(url)
		},
	}
	e := engine.NewEngine(&engine.Config{
		ProxyConfig:  d.cfg.Proxy,
		StreamConfig: engineCfg,
	})
	session.SetEngine(e)
	return e, session
}
