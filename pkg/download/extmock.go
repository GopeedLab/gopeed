package download

import (
	"fmt"
	"os"
)

func NewExtMockDownloader() (*Downloader, func(), error) {
	d := NewDownloader(&DownloaderConfig{
		Storage: NewMemStorage(),
	})
	if err := d.Setup(); err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = d.Clear()
	}
	return d, cleanup, nil
}

type ExtMockRuntime struct {
	engine  engineWrapper
	session *engineSession
}

type engineWrapper interface {
	Close()
	RunString(script string) (any, error)
}

func (d *Downloader) NewExtMockRuntime() (*ExtMockRuntime, error) {
	engine, session := d.newExtensionEngine()
	ext := &Extension{
		Name:    "extmock",
		Author:  "gopeed",
		Title:   "Gopeed ExtMock Runtime",
		Version: "0.0.0",
		DevMode: true,
	}
	gopeed := &Instance{
		Events:   make(InstanceEvents),
		Info:     NewExtensionInfo(ext),
		Logger:   newInstanceLogger(ext, d.ExtensionLogger),
		Settings: map[string]any{},
		Storage: &ContextStorage{
			storage:  d.storage,
			identity: ext.buildIdentity(),
		},
	}
	if err := engine.Runtime.Set("gopeed", gopeed); err != nil {
		engine.Close()
		return nil, err
	}
	return &ExtMockRuntime{
		engine:  engine,
		session: session,
	}, nil
}

func (r *ExtMockRuntime) Eval(script string) (any, error) {
	if r == nil || r.engine == nil {
		return nil, fmt.Errorf("extmock runtime not initialized")
	}
	return r.engine.RunString(script)
}

func (r *ExtMockRuntime) EvalFile(path string) (any, error) {
	if path == "" {
		return nil, fmt.Errorf("script path is empty")
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.Eval(string(buf))
}

func (r *ExtMockRuntime) Close() {
	if r == nil {
		return
	}
	if r.engine != nil {
		r.engine.Close()
	}
	r.engine = nil
	r.session = nil
}
