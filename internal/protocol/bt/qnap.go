package bt

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
)

// Auto-detection of a local rtorrent on a QNAP NAS. QNAP lists its installed
// packages in /etc/config/qpkg.conf. When the Rtorrent 16 package
// (rtorrent16) is installed and enabled there, its rtorrent answers XML-RPC on
// a Unix socket in its install folder, var/run/rpc.sock, which is the address
// it gives other packages. With the hand-off switch never set by the user,
// torrents then go to that rtorrent. Nothing here runs on other systems.

// qpkgConfPath is QNAP's package list. Tests point it at a fake file.
var qpkgConfPath = "/etc/config/qpkg.conf"

// qnapRtorrentPackage is the section name of Rtorrent 16 in qpkg.conf.
const qnapRtorrentPackage = "rtorrent16"

// detectLocalRtorrent returns the endpoint of a local rtorrent that Gopeed
// can use without any setting, or "" when there is none.
func detectLocalRtorrent() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	section := readQpkgSection(qpkgConfPath, qnapRtorrentPackage)
	if !strings.EqualFold(section["Enable"], "TRUE") {
		return ""
	}
	dir := section["Install_Path"]
	if !filepath.IsAbs(dir) {
		return ""
	}
	return "scgi://" + filepath.Join(dir, "var", "run", "rpc.sock")
}

// readQpkgSection returns the key = value lines of one [section] of a
// qpkg.conf style file. A missing file or section gives an empty map.
func readQpkgSection(path, name string) map[string]string {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		return values
	}
	defer file.Close()
	in := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			in = strings.TrimSpace(line[1:len(line)-1]) == name
			continue
		}
		if !in {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}

// localRtorrentJob keeps state.localRtorrent current, so the settings page can
// say that a local rtorrent was found. The hand-off itself checks again when a
// torrent is added.
func localRtorrentJob(host fetcher.ProtocolHost) fetcher.CronJob {
	return fetcher.CronJob{
		Name:    "detect-local-rtorrent",
		Every:   time.Minute,
		Timeout: 10 * time.Second,
		Run: func(ctx context.Context, source fetcher.TriggerSource) error {
			found := detectLocalRtorrent()
			current, err := fetcher.ReadConfig[config](host)
			if err != nil {
				return err
			}
			if current.State.LocalRtorrent == found {
				return fetcher.ErrJobSkipped
			}
			return fetcher.UpdateState[config, state](host, func(_ *config, s *state) error {
				s.LocalRtorrent = found
				return nil
			})
		},
	}
}
