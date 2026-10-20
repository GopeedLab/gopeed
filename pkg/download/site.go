package download

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GopeedLab/gopeed/internal/githubmirror"
	"golang.org/x/mod/semver"
)

const githubReleasesURL = "https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=10"
const gopeedSiteURL = "https://gopeed.com"

type AppVersionInfo struct {
	Version      string   `json:"version"`
	ChangeLog    string   `json:"changeLog"`
	ReleaseURL   string   `json:"releaseUrl"`
	AssetName    string   `json:"assetName"`
	DownloadURLs []string `json:"downloadUrls"`
}
type AppUpdateRequest struct {
	CurrentVersion string `json:"currentVersion"`
	Channel        string `json:"channel"`
	Architecture   string `json:"architecture"`
}
type appRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Body       string `json:"body"`
	URL        string `json:"html_url"`
}

func canonicalVersion(raw string) string {
	value := "v" + strings.TrimPrefix(raw, "v")
	if !semver.IsValid(value) {
		return ""
	}
	return value
}
func selectAppRelease(releases []appRelease, current string) (*AppVersionInfo, error) {
	current = canonicalVersion(current)
	if current == "" {
		return nil, errors.New("invalid current application version")
	}
	var selected *appRelease
	best := ""
	for i := range releases {
		release := &releases[i]
		version := canonicalVersion(release.Tag)
		if release.Draft || version == "" || semver.Compare(version, current) <= 0 {
			continue
		}
		preview := release.Prerelease || semver.Prerelease(version) != ""
		if semver.Prerelease(current) == "" && preview {
			continue
		}
		if semver.Prerelease(current) != "" && preview && strings.SplitN(version, "-", 2)[0] != strings.SplitN(current, "-", 2)[0] {
			continue
		}
		if selected == nil || semver.Compare(version, best) > 0 {
			selected = release
			best = version
		}
	}
	if selected == nil {
		return nil, nil
	}
	releaseURL := selected.URL
	if releaseURL == "" {
		releaseURL = "https://github.com/GopeedLab/gopeed/releases/tag/" + selected.Tag
	}
	return &AppVersionInfo{Version: strings.TrimPrefix(best, "v"), ChangeLog: selected.Body, ReleaseURL: releaseURL}, nil
}
func (d *Downloader) CheckAppUpdate(ctx context.Context, request AppUpdateRequest) (*AppVersionInfo, error) {
	ctx, finish, opErr := d.beginOperation(ctx)
	if opErr != nil {
		return nil, opErr
	}
	defer finish()
	if canonicalVersion(request.CurrentVersion) == "" {
		return nil, errors.New("invalid current application version")
	}
	if _, err := appUpdateAssetName(request.CurrentVersion, request.Channel, request.Architecture); err != nil {
		return nil, err
	}
	var releases []appRelease
	validate := func(body []byte) error {
		var result []appRelease
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		if result == nil {
			return errors.New("invalid release list")
		}
		if len(result) > 0 {
			for _, release := range result {
				if canonicalVersion(release.Tag) != "" {
					return nil
				}
			}
			return errors.New("response contains no valid application releases")
		}
		return nil
	}
	body, err := fetchAppReleaseData(ctx, func(ctx context.Context, raw string) ([]byte, error) {
		return d.fetchResource(ctx, raw, 4<<20, false, validate)
	})
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}
	result, err := selectAppRelease(releases, request.CurrentVersion)
	if err != nil || result == nil {
		return result, err
	}
	result.AssetName, err = appUpdateAssetName(result.Version, request.Channel, request.Architecture)
	if err != nil {
		return nil, err
	}
	result.DownloadURLs = []string{}
	if result.AssetName != "" {
		cfg, err := d.GetConfig()
		if err != nil {
			return nil, err
		}
		raw := "https://github.com/GopeedLab/gopeed/releases/download/v" + result.Version + "/" + result.AssetName
		result.DownloadURLs = githubmirror.Candidates(cfg.GithubMirror, raw, githubmirror.Resource)
	}
	return result, nil
}

// Release providers are independent data sources. The GitHub provider races
// official and eligible mirror requests before falling back to the Gopeed API.
func fetchAppReleaseData(ctx context.Context, fetch func(context.Context, string) ([]byte, error)) ([]byte, error) {
	body, githubErr := fetch(ctx, githubReleasesURL)
	if githubErr == nil {
		return body, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body, siteErr := fetch(ctx, gopeedSiteURL+"/api/releases?per_page=10")
	if siteErr != nil {
		return nil, errors.Join(fmt.Errorf("GitHub releases: %w", githubErr), fmt.Errorf("Gopeed releases: %w", siteErr))
	}
	return body, nil
}

// Package selection uses the client's channel and architecture, including when
// the Downloader runs remotely. Channels without a package use the existing installer or release page.
func appUpdateAssetName(version, channel, architecture string) (string, error) {
	arch := map[string]string{"ia32": "386", "x64": "amd64", "arm": "arm", "arm64": "arm64", "riscv32": "riscv32", "riscv64": "riscv64"}[architecture]
	if architecture != "" && arch == "" {
		return "", errors.New("invalid application update architecture")
	}
	switch channel {
	case "", "linuxFlathub", "linuxSnap", "linuxAppImage", "linuxRpm", "iosIpa", "docker":
		return "", nil
	case "windowsInstaller", "windowsPortable", "macosDmg", "linuxDeb", "androidApk":
		if arch == "" {
			return "", errors.New("application update architecture is required")
		}
	default:
		return "", errors.New("invalid application update channel")
	}
	prefix := "Gopeed-" + canonicalVersion(version)
	switch channel {
	case "windowsInstaller":
		return prefix + "-windows-" + arch + ".zip", nil
	case "windowsPortable":
		return prefix + "-windows-" + arch + "-portable.zip", nil
	case "macosDmg":
		return prefix + "-macos-" + arch + ".dmg", nil
	case "linuxDeb":
		return prefix + "-linux-" + arch + ".deb", nil
	default: // androidApk
		abi := map[string]string{"arm": "armeabi-v7a", "arm64": "arm64-v8a", "x64": "x86_64"}[architecture]
		if abi == "" {
			return prefix + "-android.apk", nil
		}
		return prefix + "-android-" + abi + ".apk", nil
	}
}

// beginOperation binds API network work to runtime shutdown, including FFI
// calls that have no HTTP request context.
func (d *Downloader) beginOperation(parent context.Context) (context.Context, func(), error) {
	d.operationGate.Lock()
	defer d.operationGate.Unlock()
	if d.closed.Load() {
		return nil, nil, errors.New("downloader is closed")
	}
	d.operations.Add(1)
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(d.backgroundCtx, cancel)
	return ctx, func() { stop(); cancel(); d.operations.Done() }, nil
}
