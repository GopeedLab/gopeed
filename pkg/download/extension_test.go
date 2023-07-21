package download

import (
	"github.com/GopeedLab/gopeed/pkg/base"
	"os"
	"testing"
)

func TestDownloader_InstallExtensionByFolder(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("./testdata/extensions/basic"); err != nil {
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
		if err := downloader.InstallExtensionByGit("https://github.com/GopeedLab/gopeed-extension-samples#github-release-sample"); err != nil {
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
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("./testdata/extensions/update"); err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if len(extensions) == 0 {
			t.Fatal("extension not installed")
		}
		ext := extensions[0]
		if ext == nil {
			t.Fatal("extension not found")
		}
		oldVersion := ext.Version
		// fetch new version from git
		newVersion, err := downloader.UpgradeCheckExtension(ext.Identity)
		if err != nil {
			t.Fatal(err)
		}
		if newVersion == "" {
			t.Fatal("new version not found")
		}
		// update extension
		if err = downloader.UpgradeExtension(ext.Identity); err != nil {
			t.Fatal(err)
		}
		extensions = downloader.GetExtensions()
		if len(extensions) != 1 || extensions[0].Version == oldVersion {
			t.Fatal("extension update fail")
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

func TestDownloader_Extension_Settings(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_empty"); err != nil {
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
		if err := downloader.InstallExtensionByFolder("./testdata/extensions/settings_all"); err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		downloader.UpdateExtensionSettings(extensions[0].Identity, map[string]any{
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

func TestDownloader_DeleteExtension(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("./testdata/extensions/basic"); err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if err := downloader.DeleteExtension(extensions[0].Identity); err != nil {
			t.Fatal(err)
		}
		extensions = downloader.GetExtensions()
		if len(extensions) != 0 {
			t.Fatal("extension delete fail")
		}
	})
}

func TestDownloader_Extension_OnResolve(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("./testdata/extensions/basic"); err != nil {
			t.Fatal(err)
		}
		extensions := downloader.GetExtensions()
		if err := downloader.DeleteExtension(extensions[0].Identity); err != nil {
			t.Fatal(err)
		}
		extensions = downloader.GetExtensions()
		if len(extensions) != 0 {
			t.Fatal("extension delete fail")
		}
	})
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
