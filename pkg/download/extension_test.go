package download

import (
	"fmt"
	"github.com/GopeedLab/gopeed/internal/test"
	"github.com/GopeedLab/gopeed/pkg/base"
	"testing"
)

func TestDownloader_InstallExtensionByFolder(t *testing.T) {
	setupDownloader(func(downloader *Downloader) {
		if err := downloader.InstallExtensionByFolder("D:\\code\\study\\node\\gopeed-extension-test"); err != nil {
			t.Fatal(err)
		}
		rr, err := downloader.Resolve(&base.Request{
			URL: "https://github.com/GopeedLab/gopeed/releases",
		})
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(test.ToJson(rr))
	})
}

func setupDownloader(fn func(downloader *Downloader)) {
	defaultDownloader.Setup()
	defaultDownloader.cfg.StorageDir = "test_ext"
	defaultDownloader.cfg.DownloadDir = "test_download"
	defer defaultDownloader.Clear()

	fn(defaultDownloader)
}
