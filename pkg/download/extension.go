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
	"strings"
)

func (d *Downloader) InstallExtensionByGit(url string) error {
	ext, err := d.fetchExtensionInfoByGit(url)
	if err != nil {
		return err
	}

	tempDir := filepath.Join(d.cfg.StorageDir, "extensions_temp", ext.Dir)
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
	return nil
}

func (d *Downloader) triggerBeforeResolve(req *base.Request) (res *base.Resource) {
	// init extension global object
	gopeed := &Instance{
		Hooks: &InstanceHooks{
			hooks: make(map[string]goja.Callable),
		},
	}
	ctx := &Context{
		Req: req,
	}
	var err error
	for _, ext := range d.extensions {
		for _, script := range ext.Manifest.Scripts {
			var match bool
			for _, exp := range script.Matches {
				if match, err = path.Match(exp, req.URL); err != nil {
					// TODO: log
					match = false
				} else if match {
					break
				}
			}
			if match {
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
					if fn, ok := gopeed.Hooks.hooks["beforeResolve"]; ok {
						_, err = engine.CallFunction(fn, ctx)
						if err != nil {
							// TODO: log
							return
						}
						fmt.Println(ctx)
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

	// Manifest engine manifest info
	*Manifest `json:"manifest"`
}

type Manifest struct {
	// Name engine global unique name
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	// Version semantic version string
	Version string `json:"version"`
	// Homepage homepage url
	Homepage string     `json:"homepage"`
	Scripts  []*Script  `json:"scripts"`
	Settings []*Setting `json:"settings"`
}

type Script struct {
	// Matches match url pattern list
	Matches []string `json:"matches"`
	// Entry js script file path
	Entry string `json:"entry"`
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
	Hooks *InstanceHooks `json:"hooks"`
}

type InstanceHooks struct {
	hooks map[string]goja.Callable
}

func (h *InstanceHooks) Register(name string, fn goja.Callable) {
	h.hooks[name] = fn
}

type Context struct {
	Req *base.Request  `json:"req"`
	Res *base.Resource `json:"res"`
}
