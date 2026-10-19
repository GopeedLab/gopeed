package download

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/GopeedLab/gopeed/internal/githubmirror"
	"golang.org/x/mod/semver"
)

const githubReleasesURL = "https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=10"
const gopeedSiteURL = "https://gopeed.com"

type AppVersionInfo struct {
	Version    string `json:"version"`
	ChangeLog  string `json:"changeLog"`
	ReleaseURL string `json:"releaseUrl"`
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
func (d *Downloader) CheckAppUpdate(ctx context.Context, current string) (*AppVersionInfo, error) {
	ctx, finish, opErr := d.beginOperation(ctx)
	if opErr != nil {
		return nil, opErr
	}
	defer finish()
	if canonicalVersion(current) == "" {
		return nil, errors.New("invalid current application version")
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
	return selectAppRelease(releases, current)
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

// Store methods accept business parameters only, never arbitrary hosts/paths.
func (d *Downloader) StoreExtensions(ctx context.Context, query url.Values) (any, error) {
	ctx, finish, opErr := d.beginOperation(ctx)
	if opErr != nil {
		return nil, opErr
	}
	defer finish()
	values := url.Values{"view": {"summary"}}
	allowed := map[string]bool{"page": true, "limit": true, "sort": true, "order": true, "q": true}
	for key, list := range query {
		if !allowed[key] {
			return nil, fmt.Errorf("unsupported store parameter: %s", key)
		}
		if len(list) != 1 {
			return nil, errors.New("duplicate store parameter")
		}
		values.Set(key, list[0])
	}
	if value := values.Get("sort"); value != "" && value != "stars" && value != "installs" && value != "updated" {
		return nil, errors.New("invalid store sort")
	}
	if value := values.Get("order"); value != "" && value != "asc" && value != "desc" {
		return nil, errors.New("invalid store order")
	}
	for _, key := range []string{"page", "limit"} {
		if value := values.Get(key); value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || (key == "limit" && n > 100) {
				return nil, fmt.Errorf("invalid store %s", key)
			}
		}
	}
	if len(values.Get("q")) > 512 {
		return nil, errors.New("store query too long")
	}
	return d.siteJSON(ctx, "/api/extensions", values)
}
func (d *Downloader) StoreExtension(ctx context.Context, id, version string) (any, error) {
	ctx, finish, opErr := d.beginOperation(ctx)
	if opErr != nil {
		return nil, opErr
	}
	defer finish()
	if id == "" || len(id) > 256 || strings.ContainsAny(id, "/\\\\") || id == "." || id == ".." {
		return nil, errors.New("invalid store extension ID")
	}
	if len(version) > 128 {
		return nil, errors.New("invalid store extension version")
	}
	query := url.Values{}
	if version != "" {
		query.Set("version", version)
	}
	return d.siteJSON(ctx, "/api/extensions/"+url.PathEscape(id), query)
}
func (d *Downloader) siteJSON(ctx context.Context, path string, query url.Values) (any, error) {
	raw := gopeedSiteURL + path
	if len(query) > 0 {
		raw += "?" + query.Encode()
	}
	body, err := d.fetchResource(ctx, raw, 8<<20, false, func(body []byte) error {
		if !json.Valid(body) {
			return errors.New("invalid site JSON")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var result any
	err = json.Unmarshal(body, &result)
	return result, err
}
func (d *Downloader) ReportStoreInstall(ctx context.Context, id string) error {
	ctx, finish, opErr := d.beginOperation(ctx)
	if opErr != nil {
		return opErr
	}
	defer finish()
	if id == "" || len(id) > 256 {
		return errors.New("invalid store extension ID")
	}
	body, _ := json.Marshal(map[string]string{"id": id})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, gopeedSiteURL+"/api/extensions/install", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	client, err := d.resourceClient(false)
	if err != nil {
		return err
	}
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("install report HTTP %d", response.StatusCode)
	}
	return nil
}

type AppUpdateCandidates struct {
	URLs []string `json:"urls"`
}

var updateAssetPattern = regexp.MustCompile(`^Gopeed-v[0-9A-Za-z.+-]+-(windows|macos|linux|android)(-[0-9A-Za-z._-]+)?\.(zip|dmg|deb|apk)$`)

// GetAppUpdateCandidates generates fixed Gopeed release URLs. The client probes
// and downloads the package locally, preserving download progress and supporting
// clients connected to a remote Downloader.
func (d *Downloader) GetAppUpdateCandidates(version, asset string) (*AppUpdateCandidates, error) {
	version = canonicalVersion(version)
	if version == "" || !updateAssetPattern.MatchString(asset) || !strings.HasPrefix(asset, "Gopeed-"+version+"-") || filepath.Base(asset) != asset {
		return nil, errors.New("invalid application update asset")
	}
	raw := "https://github.com/GopeedLab/gopeed/releases/download/" + version + "/" + asset
	cfg, err := d.GetConfig()
	if err != nil {
		return nil, err
	}
	return &AppUpdateCandidates{URLs: githubmirror.Candidates(cfg.GithubMirror, raw, githubmirror.Resource)}, nil
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
