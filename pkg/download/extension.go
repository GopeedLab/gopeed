package download

import (
	"encoding/json"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download/engine"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/dop251/goja"
	"github.com/go-git/go-git/v5"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	gitSuffix = ".git"

	tempExtensionsDir   = ".extensions"
	extensionsDir       = "extensions"
	extensionIgnoreDirs = []string{gitSuffix, "node_modules"}

	ErrExtensionNotFound = fmt.Errorf("extension not found")
)

type ActivationEvent string

const (
	HookEventOnResolve ActivationEvent = "onResolve"
	//HookEventOnError   HookEvent = "onError"
	//HookEventOnDone    HookEvent = "onDone"
)

func (d *Downloader) InstallExtensionByGit(url string) (*Extension, error) {
	return d.fetchExtensionByGit(url, d.InstallExtensionByFolder)
}

func (d *Downloader) InstallExtensionByFolder(path string) (*Extension, error) {
	ext, err := d.parseExtensionByPath(path)
	if err != nil {
		return nil, err
	}

	if err = util.CopyDir(path, d.extensionPath(ext), extensionIgnoreDirs...); err != nil {
		return nil, err
	}

	// if extension is not installed, add it to the list, otherwise update it
	installedExt := d.getExtension(ext.Identity)
	if installedExt == nil {
		ext.CreatedAt = time.Now()
		ext.UpdatedAt = ext.CreatedAt
		installedExt = ext
		d.extensions = append(d.extensions, installedExt)
	} else {
		installedExt.update(ext)
	}
	if err = d.storage.Put(bucketExtension, installedExt.Identity, installedExt); err != nil {
		return nil, err
	}
	return installedExt, nil
}

// UpgradeCheckExtension Check if there is a new version for the extension.
func (d *Downloader) UpgradeCheckExtension(identity string) (newVersion string, err error) {
	ext, err := d.GetExtension(identity)
	if err != nil {
		return "", err
	}
	if ext.InstallUrl == "" {
		return
	}
	_, err = d.fetchExtensionByGit(ext.InstallUrl, func(tempExtPath string) (*Extension, error) {
		tempExt, err := d.parseExtensionByPath(tempExtPath)
		if err != nil {
			return nil, err
		}
		if tempExt.Version != ext.Version {
			newVersion = tempExt.Version
		}
		return tempExt, nil
	})
	return
}

func (d *Downloader) UpgradeExtension(identity string) error {
	ext, err := d.GetExtension(identity)
	if err != nil {
		return err
	}
	if ext.InstallUrl == "" {
		return nil
	}
	if _, err := d.InstallExtensionByGit(ext.InstallUrl); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) UpdateExtensionSettings(identity string, settings map[string]any) error {
	ext, err := d.GetExtension(identity)
	if err != nil {
		return err
	}
	for _, setting := range ext.Settings {
		if value, ok := settings[setting.Name]; ok {
			setting.Value = tryParse(value, setting.Type)
		}
	}
	return d.storage.Put(bucketExtension, ext.Identity, ext)
}

func (d *Downloader) DeleteExtension(identity string) error {
	ext, err := d.GetExtension(identity)
	if err != nil {
		return err
	}
	// remove from disk
	if err := os.RemoveAll(d.extensionPath(ext)); err != nil {
		return err
	}
	// remove from extensions
	for i, e := range d.extensions {
		if e.Identity == identity {
			d.extensions = append(d.extensions[:i], d.extensions[i+1:]...)
			break
		}
	}
	return d.storage.Delete(bucketExtension, identity)
}

func (d *Downloader) GetExtensions() []*Extension {
	return d.extensions
}

func (d *Downloader) GetExtension(identity string) (*Extension, error) {
	extension := d.getExtension(identity)
	if extension == nil {
		return nil, ErrExtensionNotFound
	}
	return extension, nil
}

func (d *Downloader) getExtension(identity string) *Extension {
	for _, ext := range d.extensions {
		if ext.Identity == identity {
			return ext
		}
	}
	return nil
}

func (d *Downloader) fetchExtensionByGit(url string, handler func(tempExtPath string) (*Extension, error)) (*Extension, error) {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	// resolve project path
	parentPath, projectPath := path.Split(url)
	// resolve project name and sub path
	pathArr := strings.SplitN(projectPath, "#", 2)
	projectPath = strings.TrimSuffix(pathArr[0], gitSuffix)
	subPath := ""
	if len(pathArr) > 1 {
		subPath = pathArr[1]
	}

	tempExtDir := filepath.Join(d.cfg.StorageDir, tempExtensionsDir, projectPath)
	if err := util.RmAndMkDirAll(tempExtDir); err != nil {
		return nil, err
	}
	// clone project to extension temp dir
	gitUrl := parentPath + projectPath + gitSuffix
	if _, err := git.PlainClone(tempExtDir, false, &git.CloneOptions{
		URL:   gitUrl,
		Depth: 1,
	}); err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempExtDir)

	return handler(filepath.Join(tempExtDir, subPath))
}

func (d *Downloader) parseExtensionByPath(path string) (*Extension, error) {
	// resolve extension manifest
	manifestTempPath := filepath.Join(path, "manifest.json")
	if _, err := os.Stat(manifestTempPath); os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.ReadFile(manifestTempPath)
	if err != nil {
		return nil, err
	}
	var ext Extension
	if err = json.Unmarshal(file, &ext); err != nil {
		return nil, err
	}
	if err = ext.validate(); err != nil {
		return nil, err
	}
	ext.Identity = ext.buildIdentity()
	return &ext, nil
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
					ctx.Settings = parseSettings(ext.Settings)
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
							if err = ctx.Res.Validate(); err != nil {
								// TODO: log
								return
							}
							ctx.Res.CalcSize()
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
	return filepath.Join(d.cfg.StorageDir, extensionsDir, ext.Identity)
}

type ExtensionBase struct {
	// Identity is global unique for an extension, it's a combination of author and name
	Identity    string `json:"identity"`
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
	Repository string `json:"repository"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Extension struct {
	ExtensionBase

	Scripts  []*Script  `json:"scripts"`
	Settings []*Setting `json:"settings"`
}

func (e *Extension) validate() error {
	if e.Name == "" {
		return fmt.Errorf("extension name is required")
	}
	if e.Title == "" {
		return fmt.Errorf("extension title is required")
	}
	if e.Version == "" {
		return fmt.Errorf("extension version is required")
	}
	return nil
}

func (e *Extension) buildIdentity() string {
	if e.Author == "" {
		return e.Name
	}
	return e.Author + "@" + e.Name
}

func (e *Extension) update(newExt *Extension) error {
	e.Title = newExt.Title
	e.Description = newExt.Description
	e.Icon = newExt.Icon
	e.Version = newExt.Version
	e.Homepage = newExt.Homepage
	e.InstallUrl = newExt.InstallUrl
	e.Repository = newExt.Repository
	e.Scripts = newExt.Scripts
	e.Settings = newExt.Settings
	e.UpdatedAt = time.Now()
	return nil
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
	SettingTypeString  SettingType = "string"
	SettingTypeNumber  SettingType = "number"
	SettingTypeBoolean SettingType = "boolean"
)

type Setting struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	// setting type
	Type SettingType `json:"type"`
	// default value
	Default any `json:"default"`
	// setting value
	Value    any       `json:"value"`
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
	Req      *base.Request  `json:"req"`
	Res      *base.Resource `json:"res"`
	Settings map[string]any `json:"settings"`
}

func parseSettings(settings []*Setting) map[string]any {
	m := make(map[string]any)
	for _, s := range settings {
		var val any
		if s.Value != nil {
			val = s.Value
		} else {
			val = s.Default
		}
		m[s.Name] = tryParse(val, s.Type)
	}
	return m
}

func tryParse(val any, settingType SettingType) any {
	if val == nil {
		return val
	}
	switch settingType {
	case SettingTypeString:
		return fmt.Sprint(val)
	case SettingTypeNumber:
		vv, err := strconv.ParseFloat(fmt.Sprint(val), 64)
		if err != nil {
			return nil
		}
		return vv
	case SettingTypeBoolean:
		vv, err := strconv.ParseBool(fmt.Sprint(val))
		if err != nil {
			return nil
		}
		return vv
	default:
		return nil
	}
}
