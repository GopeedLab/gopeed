package download

import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download/engine"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/dop251/goja"
	"github.com/go-git/go-git/v5"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type HookEvent string

const (
	HookEventOnResolve HookEvent = "onResolve"
	HookEventOnError   HookEvent = "onError"
	HookEventOnDone    HookEvent = "onDone"
)

func (d *Downloader) InstallExtensionByGit(url string) error {
	ext, err := d.fetchExtensionInfoByGit(url)
	if err != nil {
		return err
	}

	tempDir := filepath.Join(d.cfg.StorageDir, ".extensions", ext.Dir)
	if err := util.CopyDir(tempDir, filepath.Join(d.cfg.StorageDir, "extensions", ext.Manifest.Name), "node_modules"); err != nil {
		return err
	}
	// remove temp dir
	os.RemoveAll(tempDir)
	return nil
}

func (d *Downloader) InstallExtensionByFolder(path string) error {
	ext, err := d.fetchExtensionInfoByFolder(path)
	if err != nil {
		return err
	}

	if err := util.CopyDir(path, filepath.Join(d.cfg.StorageDir, "extensions", ext.Manifest.Name), "node_modules"); err != nil {
		return err
	}
	d.extensions = append(d.extensions, ext)
	return d.storage.Put(bucketExtension, ext.Manifest.Name, ext)
}

func (d *Downloader) triggerOnResolve(req *base.Request) (res *base.Resource) {
	// init extension global object
	gopeed := &Instance{
		Hooks: make(InstanceHooks),
	}
	ctx := &Context{
		Req: req,
	}
	var err error
	for _, ext := range d.extensions {
		for _, script := range ext.Manifest.Scripts {
			if script.match(HookEventOnResolve, req.URL) {
				scriptFilePath := filepath.Join(d.cfg.StorageDir, "extensions", ext.Manifest.Name, script.Entry)
				if _, err = os.Stat(scriptFilePath); os.IsNotExist(err) {
					continue
				}
				func() {
					var scriptFile *os.File
					scriptFile, err = os.Open(scriptFilePath)
					if err != nil {
						// TODO: log
						return
					}
					defer scriptFile.Close()
					var scriptBuf []byte
					scriptBuf, err = io.ReadAll(scriptFile)
					if err != nil {
						// TODO: log
						return
					}
					engine := engine.NewEngine()
					defer engine.Close()
					err = engine.Runtime.Set("gopeed", gopeed)
					if err != nil {
						// TODO: log
						return
					}
					_, err = engine.RunString(string(scriptBuf))
					if err != nil {
						// TODO: log
						return
					}
					if fn, ok := gopeed.Hooks[HookEventOnResolve]; ok {
						_, err = engine.CallFunction(fn, ctx)
						if err != nil {
							// TODO: log
							return
						}
						// calculate resource total size
						if ctx.Res != nil && len(ctx.Res.Files) > 0 {
							var size int64 = 0
							for _, f := range ctx.Res.Files {
								size += f.Size
							}
							ctx.Res.Size = size
						}
						res = ctx.Res
					}
				}()
			}
		}
	}
	return
}

func (d *Downloader) fetchExtensionInfoByGit(url string) (ext *Extension, err error) {
	extTempDir := filepath.Join(d.cfg.StorageDir, "extensions_temp")
	// check if temp dir not exist, create it
	if _, err = os.Stat(extTempDir); os.IsNotExist(err) {
		if err = os.Mkdir(extTempDir, os.ModePerm); err != nil {
			return
		}
	}
	_, err = git.PlainClone(extTempDir, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return
	}
	// cut project name
	_, projectDirName := filepath.Split(url)
	projectDirName = strings.TrimSuffix(projectDirName, ".git")
	ext, err = d.fetchExtensionInfoByFolder(filepath.Join(extTempDir, projectDirName))
	if err != nil {
		return
	}
	ext.URL = url
	return
}

func (d *Downloader) fetchExtensionInfoByFolder(extPath string) (ext *Extension, err error) {
	// resolve engine manifest
	manifestTempPath := filepath.Join(extPath, "manifest.json")
	if _, err = os.Stat(manifestTempPath); os.IsNotExist(err) {
		return
	}
	file, err := os.ReadFile(manifestTempPath)
	if err != nil {
		return
	}
	ext = &Extension{
		Dir:      path.Base(extPath),
		Manifest: &Manifest{},
	}
	if err = json.Unmarshal(file, ext.Manifest); err != nil {
		return
	}
	return
}

func parseRes(value any) (*base.Resource, error) {
	if value == nil {
		return nil, nil
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var res base.Resource
	if err := json.Unmarshal(buf, &res); err != nil {
		return nil, err
	}
	if err := res.Validate(); err != nil {
		return nil, err
	}
	return &res, nil
}

type Extension struct {
	// URL git repository url
	URL string `json:"url"`
	// Dir engine directory name
	Dir string `json:"dir"`

	// Manifest extension manifest info
	*Manifest `json:"manifest"`
}

type Manifest struct {
	// Name extension global unique name
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	// Version semantic version string
	Version string `json:"version"`
	// Homepage homepage url
	Homepage string `json:"homepage"`
	// Repository code repository url
	Repository string     `json:"repository"`
	Scripts    []*Script  `json:"scripts"`
	Settings   []*Setting `json:"settings"`
}

type Script struct {
	// Hooks hook event list
	Hooks []string `json:"hooks"`
	// Matches match url pattern list
	Matches []string `json:"matches"`
	// Entry js script file path
	Entry string `json:"entry"`
}

func (s *Script) match(event HookEvent, url string) bool {
	if len(s.Hooks) == 0 {
		return false
	}
	for _, hook := range s.Hooks {
		if hook != string(event) {
			return false
		}
	}
	if len(s.Matches) == 0 {
		return false
	}
	for _, match := range s.Matches {
		if util.Match(match, url) {
			return true
		}
	}
	return false
}

type SettingType string

const (
	SettingTypeString = "string"
	SettingTypeInt    = "int"
	SettingTypeFloat  = "float"
	SettingTypeBool   = "bool"
)

type Setting struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Required bool   `json:"required"`
	// setting type
	Type SettingType `json:"type"`
	// default value
	value    any       `json:"value"`
	Multiple bool      `json:"multiple"`
	Options  []*Option `json:"options"`
}

type Option struct {
	Title string `json:"title"`
	Value any    `json:"value"`
}

type Instance struct {
	Hooks InstanceHooks `json:"hooks"`
}

type InstanceHooks map[HookEvent]goja.Callable

func (h InstanceHooks) register(name HookEvent, fn goja.Callable) {
	h[name] = fn
}

func (h InstanceHooks) OnResolve(fn goja.Callable) {
	h.register(HookEventOnResolve, fn)
}

func (h InstanceHooks) OnError(fn goja.Callable) {
	h.register(HookEventOnError, fn)
}

func (h InstanceHooks) OnDone(fn goja.Callable) {
	h.register(HookEventOnDone, fn)
}

type Context struct {
	Req *base.Request  `json:"req"`
	Res *base.Resource `json:"res"`
}
