package download

import (
	"errors"
	"github.com/GopeedLab/gopeed/internal/logger"
	"github.com/GopeedLab/gopeed/pkg/base"
	gojaerror "github.com/GopeedLab/gopeed/pkg/download/engine/inject/error"
	"github.com/dop251/goja"
	"os"
	"testing"
	"time"
)

func TestDownloader_InstallExtensionByFolder(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/basic", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByFolderDevMode(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/basic", true); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByGit(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByGit("https://github.com/GopeedLab/gopeed-extension-samples#github-release-sample"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByGitSimple(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByGit("github.com/GopeedLab/gopeed-extension-samples#github-release-sample"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_InstallExtensionByGitFull(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByGit("https://github.com/GopeedLab/gopeed-extension-samples.git#github-release-sample"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_UpgradeExtension(t *testing.T) {
	getSetting := func(settings []*Setting, name string) *Setting {
		for _, setting := range settings {
			if setting.Name == name {
				return setting
			}
		}
		return nil
	}

	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/update", false)
		if err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if len(extensions) == 0 {
			t.Fatal("extension not installed")
		}
		oldVersion := installedExt.Version
		// fetch new version from git
		newVersion, err := downloader.UpgradeCheckExtension(installedExt.Identity)
		if err != nil {
			t.Fatal(err)
		}
		if newVersion == "" {
			t.Fatal("new version not found")
		}
		// update extension
		if err = downloader.UpgradeExtension(installedExt.Identity); err != nil {
			t.Fatal(err)
		}
		upgradeExt := downloader.getExtension(installedExt.Identity)
		if upgradeExt.Version == oldVersion {
			t.Fatal("extension update fail")
		}

		// check setting update
		s1 := getSetting(upgradeExt.Settings, "s1")
		if s1.Title == "S1 old" {
			t.Fatal("setting update fail")
		}
		// check setting type update
		s2 := getSetting(upgradeExt.Settings, "s2")
		if s2.Type == "number" {
			t.Fatal("setting type update fail")
		}
		// check setting remove
		d1 := getSetting(upgradeExt.Settings, "d1")
		if d1 != nil {
			t.Fatal("setting remove fail")
		}
		// check setting add
		s3 := getSetting(upgradeExt.Settings, "s3")
		if s3 == nil {
			t.Fatal("setting add fail")
		}

		rr, err := downloader.Resolve(&base.Request{
			URL: "https://test.com",
		})
		if err != nil {
			t.Fatal(err)
		}
		if rr.Res.Name != "test" {
			t.Fatal("script update fail")
		}
	})
}

func TestDownloader_Extension_OnStart(t *testing.T) {
	downloadAndCheck := func(req *base.Request) {
		setupDownloader(func(downloader *Downloader) {
			if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/on_start", false); err != nil {
				t.Fatal(err)
			}
			errCh := make(chan error, 1)
			downloader.Listener(func(event *Event) {
				if event.Key == EventKeyFinally {
					errCh <- event.Err
				}
			})
			id, err := downloader.CreateDirect(req, nil)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case err = <-errCh:
				break
			case <-time.After(time.Second * 10):
				err = errors.New("timeout")
			}
			if err != nil {
				panic("extension on start download error: " + err.Error())
			}
			task := downloader.GetTask(id)
			if task.Meta.Req.URL != "https://github.com" {
				t.Fatalf("except url: https://github.com, actual: %s", task.Meta.Req.URL)
			}
			if task.Meta.Req.Labels["modified"] != "true" {
				t.Fatalf("except label: modified=true, actual: %s", task.Meta.Req.Labels["modified"])
			}
		})
	}

	// url match
	downloadAndCheck(&base.Request{
		URL: "https://github.com/gopeed/test/404",
	})

	// label match
	downloadAndCheck(&base.Request{
		URL: "https://test.com",
		Labels: map[string]string{
			"test": "true",
		},
	})
}

func TestDownloader_Extension_OnError(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/on_error", false); err != nil {
			t.Fatal(err)
		}
		errCh := make(chan error, 1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyFinally {
				errCh <- event.Err
			}
		})
		id, err := downloader.CreateDirect(&base.Request{
			URL: "https://github.com/gopeed/test/404",
			Labels: map[string]string{
				"test": "true",
			},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err = <-errCh:
			break
		case <-time.After(time.Second * 10):
			err = errors.New("timeout")
		}

		if err != nil {
			panic("extension on error download error: " + err.Error())
		}
		// extension on error modify url and continue download
		task := downloader.GetTask(id)
		if task.Status != base.DownloadStatusDone {
			t.Fatalf("except status is done, actual: %s", task.Status)
		}
	})
}

func TestDownloader_Extension_OnDone(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/on_done", false); err != nil {
			t.Fatal(err)
		}
		errCh := make(chan error, 1)
		downloader.Listener(func(event *Event) {
			if event.Key == EventKeyFinally {
				errCh <- event.Err
			}
		})
		id, err := downloader.CreateDirect(&base.Request{
			URL: "https://github.com",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err = <-errCh:
			break
		case <-time.After(time.Second * 10):
			err = errors.New("timeout")
		}
		// wait for script execution
		time.Sleep(time.Millisecond * 3000)

		if err != nil {
			panic("extension on done download error: " + err.Error())
		}
		// extension on error modify url and continue download
		task := downloader.GetTask(id)
		if task.Meta.Req.Labels["modified"] != "true" {
			t.Fatalf("except label: modified=true, actual: %s", task.Meta.Req.Labels["modified"])
		}
		if task.Status != base.DownloadStatusDone {
			t.Fatalf("except status is done, actual: %s", task.Status)
		}
	})
}

func TestDownloader_Extension_Errors(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/script_error", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 2 {
			t.Fatal("script error catch failed")
		}
	})

	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/function_error", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 2 {
			t.Fatal("function error catch failed")
		}
	})

	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/message_error", false); err != nil {
			t.Fatal(err)
		}
		_, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err == nil {
			t.Fatalf("except error, but got nil")
		}
		me, ok := err.(*gojaerror.MessageError)
		if !ok {
			t.Fatalf("except MessageError type, but got %s", err)
		}
		want := "test"
		if me.Error() != want {
			t.Fatalf("except MessageError message %s, but got %s", want, me.Message)
		}
	})
}

func TestDownloader_Extension_Settings(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_empty", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("settings parse error")
		}
	})

	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_all", false)
		if err != nil {
			t.Fatal(err)
		}
		downloader.UpdateExtensionSettings(installedExt.Identity, map[string]any{
			"stringValued":  "valued",
			"numberValued":  1.1,
			"booleanValued": true,
		})
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("settings parse error")
		}
	})
}

func TestDownloader_ExtensionStorage(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if _, err := downloader.InstallExtensionByFolder("./testdata/extensions/storage", false); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) == 1 {
			t.Fatal("resolve error")
		}
	})
}

func TestDownloader_SwitchExtension(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/basic", false)
		if err != nil {
			t.Fatal(err)
		}
		if installedExt.Disabled == true {
			t.Fatal("extension disabled")
		}
		if err = downloader.SwitchExtension(installedExt.Identity, false); err != nil {
			t.Fatal(err)
		}
		if installedExt.Disabled == false {
			t.Fatal("extension enabled")
		}
	})
}

func TestDownloader_DeleteExtension(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		installedExt, err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_all", false)
		if err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if err := downloader.DeleteExtension(installedExt.Identity); err != nil {
			t.Fatal(err)
		}
		extensions = downloader.GetExtensions()
		if len(extensions) != 0 {
			t.Fatal("extension delete fail")
		}
	})
}

func TestDownloader_Extension_Logger(t *testing.T) {
	logger := logger.NewLogger(false, "")
	il := newInstanceLogger(&Extension{
		Name: "test",
	}, logger)
	il.Debug(goja.NaN(), goja.Undefined())
	il.Info(goja.NaN(), goja.Undefined())
	il.Warn(goja.NaN(), goja.Undefined())
	il.Error(goja.NaN(), goja.Undefined())
}

func setupDownloader(fn func(downloader *Downloader)) {
	defaultDownloader.Setup()
	defaultDownloader.cfg.StorageDir = ".test_storage"
	defaultDownloader.cfg.DownloadDir = ".test_download"
	defer func() {
		defaultDownloader.Clear()
		os.RemoveAll(defaultDownloader.cfg.StorageDir)
		os.RemoveAll(defaultDownloader.cfg.DownloadDir)
	}()
	fn(defaultDownloader)
}
