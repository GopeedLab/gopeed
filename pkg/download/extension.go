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

var (
	gitSuffix = ".git"

	tempExtensionsDir   = ".extensions"
	extensionsDir       = "extensions"
	extensionIgnoreDirs = []string{gitSuffix, "node_modules"}
)

type ActivationEvent string

const (
	HookEventOnResolve ActivationEvent = "onResolve"
	//HookEventOnError   HookEvent = "onError"
	//HookEventOnDone    HookEvent = "onDone"
)

func (d *Downloader) InstallExtensionByGit(url string) error {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	// resolve project path
	gitPath, projectPath := path.Split(url)
	projectPath = strings.TrimSuffix(projectPath, gitSuffix)
	// resolve project name and sub path
	pathArr := strings.SplitN(projectPath, "#", 2)
	projectPath = pathArr[0]
	subPath := ""
	if len(pathArr) > 1 {
		subPath = pathArr[1]
	}

	tempExtDir := filepath.Join(d.cfg.StorageDir, tempExtensionsDir, projectPath)
	if err := util.RmAndMkDirAll(tempExtDir); err != nil {
		return err
	}
	// clone project to extension temp dir
	gitUrl := gitPath + projectPath + gitSuffix
	if _, err := git.PlainClone(tempExtDir, false, &git.CloneOptions{
		URL:   gitUrl,
		Depth: 1,
	}); err != nil {
		return err
	}
	defer os.RemoveAll(tempExtDir)

	if err := d.InstallExtensionByFolder(filepath.Join(tempExtDir, subPath)); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) InstallExtensionByFolder(path string) error {
	// resolve engine manifest
	manifestTempPath := filepath.Join(path, "manifest.json")
	if _, err := os.Stat(manifestTempPath); os.IsNotExist(err) {
		return err
	}
	file, err := os.ReadFile(manifestTempPath)
	if err != nil {
		return err
	}
	ext := new(Extension)
	if err = json.Unmarshal(file, ext); err != nil {
		return err
	}

	if err := util.CopyDir(path, d.extensionPath(ext), extensionIgnoreDirs...); err != nil {
		return err
	}
	d.extensions = append(d.extensions, ext)
	return d.storage.Put(bucketExtension, ext.identity(), ext)
}

func (d *Downloader) triggerOnResolve(req *base.Request) (res *base.Resource) {
	// init extension global object
	gopeed := &Instance{
		Events: make(InstanceEvents),
	}
	ctx := &Context{
		Req: req,
	}
	var err error
	for _, ext := range d.extensions {
		for _, script := range ext.Scripts {
			if script.match(HookEventOnResolve, req.URL) {
				scriptFilePath := filepath.Join(d.extensionPath(ext), script.Entry)
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
					if fn, ok := gopeed.Events[HookEventOnResolve]; ok {
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

func (d *Downloader) extensionPath(ext *Extension) string {
	return filepath.Join(d.cfg.StorageDir, extensionsDir, ext.identity())
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
	Name        string `json:"name"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	// Version semantic version string, like: 1.0.0
	Version string `json:"version"`
	// Homepage homepage url
	Homepage string `json:"homepage"`
	// InstallUrl install url
	InstallUrl string `json:"installUrl"`
	// Repository git repository url
	Repository string     `json:"repository"`
	Scripts    []*Script  `json:"scripts"`
	Settings   []*Setting `json:"settings"`
}

func (e *Extension) identity() string {
	if e.Author == "" {
		return e.Name
	}
	return e.Author + "@" + e.Name
}

type Script struct {
	// Event active event name
	Event string `json:"event"`
	// Matches match request url pattern list
	Matches []string `json:"matches"`
	// Entry js script file path
	Entry string `json:"entry"`
}

func (s *Script) match(event ActivationEvent, url string) bool {
	if s.Event == "" {
		return false
	}
	if s.Event != string(event) {
		return false
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
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
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
	Events InstanceEvents `json:"events"`
}

type InstanceEvents map[ActivationEvent]goja.Callable

func (h InstanceEvents) register(name ActivationEvent, fn goja.Callable) {
	h[name] = fn
}

func (h InstanceEvents) OnResolve(fn goja.Callable) {
	h.register(HookEventOnResolve, fn)
}

//func (h InstanceEvents) OnError(fn goja.Callable) {
//	h.register(HookEventOnError, fn)
//}
//
//func (h InstanceEvents) OnDone(fn goja.Callable) {
//	h.register(HookEventOnDone, fn)
//}

type Context struct {
	Req *base.Request  `json:"req"`
	Res *base.Resource `json:"res"`
}
