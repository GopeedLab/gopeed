package download

import (
	"github.com/GopeedLab/gopeed/pkg/base"
	"os"
	"testing"
)

func TestDownloader_InstallExtensionByFolder(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("./testdata/extension"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/test",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rr.Res.Files) != 2 {
			t.Fatal("resolve error")
		}
	})
}

func setupDownloader(fn func(downloader *Downloader)) {
	defaultDownloader.Setup()
	defaultDownloader.cfg.StorageDir = ".test_ext"
	defaultDownloader.cfg.DownloadDir = ".test_download"
	defer func() {
		defaultDownloader.Clear()
		os.RemoveAll(defaultDownloader.cfg.StorageDir)
		os.RemoveAll(defaultDownloader.cfg.DownloadDir)
	}()
	fn(defaultDownloader)
}
