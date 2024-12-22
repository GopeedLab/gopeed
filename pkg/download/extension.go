package download

import (
	"encoding/json"
	"fmt"
	"github.com/GopeedLab/gopeed/internal/logger"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download/engine"
	gojaerror "github.com/GopeedLab/gopeed/pkg/download/engine/inject/error"
	gojautil "github.com/GopeedLab/gopeed/pkg/download/engine/util"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/dop251/goja"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
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

	ErrExtensionNoManifest = fmt.Errorf("manifest.json not found")
	ErrExtensionNotFound   = fmt.Errorf("extension not found")
)

type ActivationEvent string

const (
	EventOnResolve ActivationEvent = "onResolve"
	EventOnStart   ActivationEvent = "onStart"
	EventOnError   ActivationEvent = "onError"
	EventOnDone    ActivationEvent = "onDone"
)

func (d *Downloader) InstallExtensionByGit(url string) (*Extension, error) {
	return d.fetchExtensionByGit(url, d.InstallExtensionByFolder)
}

func (d *Downloader) InstallExtensionByFolder(path string, devMode bool) (*Extension, error) {
	ext, err := d.parseExtensionByPath(path)
	if err != nil {
		return nil, err
	}

	// if dev mode, don't copy to the extensions' directory
	if devMode {
		ext.DevMode = true
		ext.DevPath, _ = filepath.Abs(path)
	} else {
		if err = util.CopyDir(path, d.ExtensionPath(ext), extensionIgnoreDirs...); err != nil {
			return nil, err
		}
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
		return
	}
	installUrl := ext.buildInstallUrl()
	if installUrl == "" {
		return
	}
	_, err = d.fetchExtensionByGit(installUrl, func(tempExtPath string, devMode bool) (*Extension, error) {
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
	installUrl := ext.buildInstallUrl()
	if installUrl == "" {
		return nil
	}
	if _, err := d.InstallExtensionByGit(installUrl); err != nil {
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

func (d *Downloader) SwitchExtension(identity string, status bool) error {
	ext, err := d.GetExtension(identity)
	if err != nil {
		return err
	}
	ext.Disabled = !status
	return d.storage.Put(bucketExtension, ext.Identity, ext)
}

func (d *Downloader) DeleteExtension(identity string) error {
	ext, err := d.GetExtension(identity)
	if err != nil {
		return err
	}
	// remove from disk
	if !ext.DevMode {
		if err := os.RemoveAll(d.ExtensionPath(ext)); err != nil {
			return err
		}
	}
	// remove from extensions
	for i, e := range d.extensions {
		if e.Identity == identity {
			d.extensions = append(d.extensions[:i], d.extensions[i+1:]...)
			break
		}
	}
	if err = d.storage.Delete(bucketExtension, identity); err != nil {
		return err
	}
	if err = d.storage.Delete(bucketExtensionStorage, identity); err != nil {
		return err
	}
	return nil
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

func (d *Downloader) fetchExtensionByGit(url string, handler func(tempExtPath string, devMode bool) (*Extension, error)) (*Extension, error) {
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
	proxyOptions := transport.ProxyOptions{}
	proxyUrl := d.cfg.DownloaderStoreConfig.Proxy.ToUrl()
	if proxyUrl != nil {
		proxyOptions.URL = proxyUrl.Scheme + "://" + proxyUrl.Host
		proxyOptions.Username = proxyUrl.User.Username()
		proxyOptions.Password, _ = proxyUrl.User.Password()
	}
	// clone project to extension temp dir
	gitUrl := parentPath + projectPath + gitSuffix
	if _, err := git.PlainClone(tempExtDir, false, &git.CloneOptions{
		URL:          gitUrl,
		Depth:        1,
		ProxyOptions: proxyOptions,
	}); err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempExtDir)

	return handler(filepath.Join(tempExtDir, subPath), false)
}

func (d *Downloader) parseExtensionByPath(path string) (*Extension, error) {
	// resolve extension manifest
	manifestTempPath := filepath.Join(path, "manifest.json")
	if _, err := os.Stat(manifestTempPath); os.IsNotExist(err) {
		return nil, ErrExtensionNoManifest
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

func (d *Downloader) triggerOnResolve(req *base.Request) (res *base.Resource, err error) {
	err = doTrigger(d,
		EventOnResolve,
		req,
		&OnResolveContext{
			Req: req,
		},
		func(ext *Extension, gopeed *Instance, ctx *OnResolveContext) {
			// Validate resource structure
			if ctx.Res != nil && len(ctx.Res.Files) > 0 {
				if err := ctx.Res.Validate(); err != nil {
					gopeed.Logger.logger.Warn().Err(err).Msgf("[%s] resource invalid", ext.buildIdentity())
					return
				}
				ctx.Res.Name = util.ReplaceInvalidFilename(ctx.Res.Name)
				for _, file := range ctx.Res.Files {
					file.Name = util.ReplaceInvalidFilename(file.Name)
				}
				ctx.Res.CalcSize(nil)
			}
			res = ctx.Res
		},
	)
	return
}

func (d *Downloader) triggerOnStart(task *Task) {
	doTrigger(d,
		EventOnStart,
		task.Meta.Req,
		&OnStartContext{
			Task: NewExtensionTask(d, task),
		},
		func(ext *Extension, gopeed *Instance, ctx *OnStartContext) {
			// Validate request structure
			if ctx.Task.Meta.Req != nil {
				if err := ctx.Task.Meta.Req.Validate(); err != nil {
					gopeed.Logger.logger.Warn().Err(err).Msgf("[%s] request invalid", ext.buildIdentity())
					return
				}
			}
		},
	)
	return
}

func (d *Downloader) triggerOnError(task *Task, err error) {
	doTrigger(d,
		EventOnError,
		task.Meta.Req,
		&OnErrorContext{
			Task:  NewExtensionTask(d, task),
			Error: err,
		},
		nil,
	)
}

func (d *Downloader) triggerOnDone(task *Task) {
	doTrigger(d,
		EventOnDone,
		task.Meta.Req,
		&OnErrorContext{
			Task: NewExtensionTask(d, task),
		},
		nil,
	)
}

func doTrigger[T any](d *Downloader, event ActivationEvent, req *base.Request, ctx T, handler func(ext *Extension, gopeed *Instance, ctx T)) error {
	// init extension global object
	gopeed := &Instance{
		Events: make(InstanceEvents),
	}
	var err error
	for _, ext := range d.extensions {
		if ext.Disabled {
			continue
		}
		for _, script := range ext.Scripts {
			if script.match(event, req) {
				gopeed.Info = NewExtensionInfo(ext)
				gopeed.Logger = newInstanceLogger(ext, d.ExtensionLogger)
				gopeed.Settings = parseSettings(ext.Settings)
				gopeed.Storage = &ContextStorage{
					storage:  d.storage,
					identity: ext.buildIdentity(),
				}
				scriptFilePath := filepath.Join(d.ExtensionPath(ext), script.Entry)
				if _, err = os.Stat(scriptFilePath); os.IsNotExist(err) {
					gopeed.Logger.logger.Error().Err(err).Msgf("[%s] script file not exist", ext.buildIdentity())
					continue
				}
				func() {
					var scriptFile *os.File
					scriptFile, err = os.Open(scriptFilePath)
					if err != nil {
						gopeed.Logger.logger.Error().Err(err).Msgf("[%s] open script file failed", ext.buildIdentity())
						return
					}
					defer scriptFile.Close()
					var scriptBuf []byte
					scriptBuf, err = io.ReadAll(scriptFile)
					if err != nil {
						gopeed.Logger.logger.Error().Err(err).Msgf("[%s] read script file failed", ext.buildIdentity())
						return
					}
					// Init request labels
					if req.Labels == nil {
						req.Labels = make(map[string]string)
					}
					engine := engine.NewEngine(&engine.Config{
						ProxyConfig: d.cfg.Proxy,
					})
					defer engine.Close()
					err = engine.Runtime.Set("gopeed", gopeed)
					if err != nil {
						gopeed.Logger.logger.Error().Err(err).Msgf("[%s] engine inject failed", ext.buildIdentity())
						return
					}
					_, err = engine.RunString(string(scriptBuf))
					if err != nil {
						gopeed.Logger.logger.Error().Err(err).Msgf("[%s] run script failed", ext.buildIdentity())
						return
					}
					if fn, ok := gopeed.Events[event]; ok {
						_, err = engine.CallFunction(fn, ctx)
						if err != nil {
							gopeed.Logger.logger.Error().Err(err).Msgf("[%s] call function failed: %s", ext.buildIdentity(), event)
							return
						}
						if handler != nil {
							handler(ext, gopeed, ctx)
						}
					}
				}()
			}
		}
	}

	// Only return MessageError
	if me, ok := gojautil.AssertError[*gojaerror.MessageError](err); ok {
		return me
	}
	return nil
}

func (d *Downloader) ExtensionPath(ext *Extension) string {
	if ext.DevMode {
		return ext.DevPath
	}
	return filepath.Join(d.cfg.StorageDir, extensionsDir, ext.Identity)
}

type Extension struct {
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
	// Repository git repository info
	Repository *Repository `json:"repository"`
	Scripts    []*Script   `json:"scripts"`
	Settings   []*Setting  `json:"settings"`
	// Disabled if true, this extension will be ignored
	Disabled bool `json:"disabled"`

	DevMode bool `json:"devMode"`
	// DevPath is the local path of extension source code
	DevPath string `json:"devPath"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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

func (e *Extension) buildInstallUrl() string {
	if e.Repository == nil || e.Repository.Url == "" {
		return ""
	}
	repoUrl := e.Repository.Url
	if e.Repository.Directory != "" {
		if strings.HasSuffix(repoUrl, "/") {
			repoUrl = repoUrl[:len(repoUrl)-1]
		}
		dir := e.Repository.Directory
		if strings.HasPrefix(dir, "/") {
			dir = dir[1:]
		}
		repoUrl = repoUrl + "#" + dir
	}
	return repoUrl
}

func (e *Extension) update(newExt *Extension) error {
	e.Title = newExt.Title
	e.Description = newExt.Description
	e.Icon = newExt.Icon
	e.Version = newExt.Version
	e.Homepage = newExt.Homepage
	e.Repository = newExt.Repository
	e.Scripts = newExt.Scripts
	// merge settings
	// if new setting not exist in old settings, append it
	for _, newSetting := range newExt.Settings {
		var exist bool
		for _, setting := range e.Settings {
			if setting.Name == newSetting.Name {
				exist = true
				break
			}
		}
		if !exist {
			e.Settings = append(e.Settings, newSetting)
		}
	}
	// if old setting not exist in new settings, remove it
	for i := 0; i < len(e.Settings); i++ {
		var exist bool
		for _, setting := range newExt.Settings {
			if setting.Name == e.Settings[i].Name {
				exist = true
				break
			}
		}
		if !exist {
			e.Settings = append(e.Settings[:i], e.Settings[i+1:]...)
			i--
		}
	}
	// if new setting exist in old settings, update it
	for _, newSetting := range newExt.Settings {
		for _, setting := range e.Settings {
			if setting.Name == newSetting.Name {
				setting.Title = newSetting.Title
				setting.Description = newSetting.Description
				setting.Options = newSetting.Options
				// if type changed, reset value
				if setting.Type != newSetting.Type {
					setting.Type = newSetting.Type
					setting.Value = newSetting.Value
				}
				break
			}
		}
	}
	e.UpdatedAt = time.Now()
	return nil
}

type Repository struct {
	Url       string `json:"url"`
	Directory string `json:"directory"`
}

type Script struct {
	// Event active event name
	Event string `json:"event"`
	// Match rules
	Match *Match `json:"match"`
	// Entry js script file path
	Entry string `json:"entry"`
}

func (s *Script) match(event ActivationEvent, req *base.Request) bool {
	if s.Event == "" {
		return false
	}
	if s.Event != string(event) {
		return false
	}
	if s.Match == nil || (len(s.Match.Urls) == 0 && len(s.Match.Labels) == 0) {
		return false
	}

	// match url
	for _, url := range s.Match.Urls {
		if util.Match(url, req.URL) {
			return true
		}
	}

	// match label
	for _, label := range s.Match.Labels {
		if _, ok := req.Labels[label]; ok {
			return true
		}
	}
	return false
}

type Match struct {
	// Urls match expression, refer to https://developer.chrome.com/docs/extensions/mv3/match_patterns/
	Urls []string `json:"urls"`
	// Labels match request labels
	Labels []string `json:"labels"`
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
	// setting value
	Value any `json:"value"`
	//Multiple bool      `json:"multiple"`
	Options []*Option `json:"options"`
}

type Option struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// Instance inject to js context when extension script is activated
type Instance struct {
	Events   InstanceEvents  `json:"events"`
	Info     *ExtensionInfo  `json:"info"`
	Logger   *InstanceLogger `json:"logger"`
	Settings map[string]any  `json:"settings"`
	Storage  *ContextStorage `json:"storage"`
}

type InstanceEvents map[ActivationEvent]goja.Callable

func (h InstanceEvents) register(name ActivationEvent, fn goja.Callable) {
	h[name] = fn
}

func (h InstanceEvents) OnResolve(fn goja.Callable) {
	h.register(EventOnResolve, fn)
}

func (h InstanceEvents) OnStart(fn goja.Callable) {
	h.register(EventOnStart, fn)
}

func (h InstanceEvents) OnError(fn goja.Callable) {
	h.register(EventOnError, fn)
}

func (h InstanceEvents) OnDone(fn goja.Callable) {
	h.register(EventOnDone, fn)
}

type ExtensionInfo struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
	Author   string `json:"author"`
	Title    string `json:"title"`
	Version  string `json:"version"`
}

func NewExtensionInfo(ext *Extension) *ExtensionInfo {
	return &ExtensionInfo{
		Identity: ext.buildIdentity(),
		Name:     ext.Name,
		Author:   ext.Author,
		Title:    ext.Title,
		Version:  ext.Version,
	}
}

type InstanceLogger struct {
	identity string
	devMode  bool
	logger   *logger.Logger
}

func (l *InstanceLogger) Debug(msg ...goja.Value) {
	if l.devMode {
		l.logger.Debug().Msg(l.append(msg...))
	}
}

func (l *InstanceLogger) Info(msg ...goja.Value) {
	l.logger.Info().Msg(l.append(msg...))
}

func (l *InstanceLogger) Warn(msg ...goja.Value) {
	l.logger.Warn().Msg(l.append(msg...))
}

func (l *InstanceLogger) Error(msg ...goja.Value) {
	l.logger.Error().Msg(l.append(msg...))
}

func (l *InstanceLogger) append(msg ...goja.Value) string {
	strMsg := make([]string, len(msg))
	for i, m := range msg {
		strMsg[i] = m.String()
	}
	return fmt.Sprintf("[%s] %s", l.identity, strings.Join(strMsg, " "))
}

func newInstanceLogger(extension *Extension, logger *logger.Logger) *InstanceLogger {
	return &InstanceLogger{
		identity: extension.buildIdentity(),
		devMode:  extension.DevMode,
		logger:   logger,
	}
}

type OnResolveContext struct {
	Req *base.Request  `json:"req"`
	Res *base.Resource `json:"res"`
}

type OnStartContext struct {
	Task *ExtensionTask `json:"task"`
}

type OnErrorContext struct {
	Task  *ExtensionTask `json:"task"`
	Error error          `json:"error"`
}

type OnDoneContext struct {
	Task *Task `json:"task"`
}

// ExtensionTask is a wrapper of Task, it's used to interact with extension scripts.
// Avoid extension scripts modifying task directly, use ExtensionTask to encapsulate task,
// only some fields can be modified, such as request info.
type ExtensionTask struct {
	download *Downloader

	*Task
}

func NewExtensionTask(download *Downloader, task *Task) *ExtensionTask {
	// restricts extension scripts to only modify request info
	newTask := task.clone()
	newTask.Meta.Req = task.Meta.Req
	return &ExtensionTask{
		download: download,
		Task:     newTask,
	}
}

func (t *ExtensionTask) Continue() error {
	return t.download.Continue(&TaskFilter{
		IDs: []string{t.ID},
	})
}

func (t *ExtensionTask) Pause() error {
	return t.download.Pause(&TaskFilter{
		IDs: []string{t.ID},
	})
}

func parseSettings(settings []*Setting) map[string]any {
	m := make(map[string]any)
	for _, s := range settings {
		var val any
		if s.Value != nil {
			val = s.Value
		}
		m[s.Name] = tryParse(val, s.Type)
	}
	return m
}

func tryParse(val any, settingType SettingType) any {
	if val == nil {
		return nil
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

type ContextStorage struct {
	storage  Storage
	identity string
}

func (s *ContextStorage) Get(key string) any {
	raw := s.getRawData()
	if v, ok := raw[key]; ok {
		return v
	}
	return nil
}

func (s *ContextStorage) Set(key string, value string) {
	raw := s.getRawData()
	raw[key] = value
	s.storage.Put(bucketExtensionStorage, s.identity, raw)
}

func (s *ContextStorage) Remove(key string) {
	raw := s.getRawData()
	delete(raw, key)
	s.storage.Put(bucketExtensionStorage, s.identity, raw)
}

func (s *ContextStorage) Keys() []string {
	raw := s.getRawData()
	keys := make([]string, 0)
	for k := range raw {
		keys = append(keys, k)
	}
	return keys
}

func (s *ContextStorage) Clear() {
	s.storage.Delete(bucketExtensionStorage, s.identity)
}

func (s *ContextStorage) getRawData() map[string]string {
	var data map[string]string
	s.storage.Get(bucketExtensionStorage, s.identity, &data)
	if data == nil {
		data = make(map[string]string)
	}
	return data
}
