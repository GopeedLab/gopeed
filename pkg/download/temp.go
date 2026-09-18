package download

import (
	"os"
	"path/filepath"

	"github.com/GopeedLab/gopeed/internal/tempfiles"
)

// TempDir may be shared with the OS or other applications. Initialization never
// removes its contents; shutdown only removes paths created by this downloader.
func (d *Downloader) setupTempDir() error {
	if d.cfg.TempDir == "" {
		d.cfg.TempDir = os.TempDir()
	}
	root, err := filepath.Abs(d.cfg.TempDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	d.cfg.TempDir = root
	d.tempFiles = &tempfiles.Scope{}
	return nil
}
