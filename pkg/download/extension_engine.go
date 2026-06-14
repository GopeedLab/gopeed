package download

import (
	"fmt"
	"os"
)

type extensionEngineRunner interface {
	Close()
	RunString(script string) (any, error)
}

type ExtensionEngine struct {
	engine  extensionEngineRunner
	session *engineSession
}

func (d *Downloader) NewExtensionEngine(ext *Extension, settings map[string]any) (*ExtensionEngine, error) {
	if ext == nil {
		return nil, fmt.Errorf("extension is nil")
	}
	engine, session := d.newExtensionEngine()
	gopeed := &Instance{
		Events:   make(InstanceEvents),
		Info:     NewExtensionInfo(ext),
		Logger:   newInstanceLogger(ext, d.ExtensionLogger),
		Settings: settings,
		Storage: &ContextStorage{
			storage:  d.storage,
			identity: ext.buildIdentity(),
		},
		Runtime: &InstanceRuntime{
			WebView: d.newExtensionWebViewRuntime(session),
		},
	}
	if err := injectGopeed(engine.Runtime, gopeed); err != nil {
		session.CloseIfIdle()
		return nil, err
	}
	return &ExtensionEngine{
		engine:  engine,
		session: session,
	}, nil
}

func (r *ExtensionEngine) Eval(script string) (any, error) {
	if r == nil || r.engine == nil {
		return nil, fmt.Errorf("extension engine not initialized")
	}
	return r.engine.RunString(script)
}

func (r *ExtensionEngine) EvalFile(path string) (any, error) {
	if path == "" {
		return nil, fmt.Errorf("script path is empty")
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.Eval(string(buf))
}

func (r *ExtensionEngine) Close() {
	if r == nil {
		return
	}
	if r.session != nil {
		r.session.CloseIfIdle()
	} else if r.engine != nil {
		r.engine.Close()
	}
	r.engine = nil
	r.session = nil
}
