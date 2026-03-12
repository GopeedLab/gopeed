package torbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/GopeedLab/gopeed/internal/debrid/types"
)

const (
	api             = "https://api.torbox.app/v1/api"
	pollInterval    = 5 * time.Second
	defaultTimeout  = 90 * time.Second
)

type service struct {
	apiKey string
	client *http.Client
}

func New(apiKey string) types.Service {
	return &service{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *service) Name() types.ServiceName {
	return types.ServiceTorBox
}

func (s *service) Resolve(ctx context.Context, magnetOrTorrent string) ([]types.File, error) {
	// 1. Add magnet to TorBox
	torrentID, err := s.createTorrent(ctx, magnetOrTorrent)
	if err != nil {
		return nil, fmt.Errorf("torbox: add torrent: %w", err)
	}

	// 2. Poll until cached (respect caller context + 90s internal deadline)
	pollCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	torrent, err := s.waitUntilCached(pollCtx, torrentID)
	if err != nil {
		return nil, err
	}

	// 3. Build permanent redirect URLs (no extra API round-trip per file)
	files := torrent.Files
	if len(files) == 0 {
		// Single-file torrent with no file list — use torrent-level download
		u := s.redirectURL(torrentID, 0, false)
		return []types.File{{Name: torrent.Name, Size: torrent.Size, URL: u}}, nil
	}

	result := make([]types.File, len(files))
	for i, f := range files {
		result[i] = types.File{
			Name: f.Name,
			Size: f.Size,
			URL:  s.redirectURL(torrentID, f.ID, false),
		}
	}
	return result, nil
}

// redirectURL builds a permanent requestdl URL that TorBox redirects to the
// CDN at download time — no extra API call needed.
func (s *service) redirectURL(torrentID int, fileID int, zip bool) string {
	q := url.Values{}
	q.Set("token", s.apiKey)
	q.Set("torrent_id", fmt.Sprintf("%d", torrentID))
	q.Set("zip_link", fmt.Sprintf("%v", zip))
	q.Set("redirect", "true")
	if fileID > 0 {
		q.Set("file_id", fmt.Sprintf("%d", fileID))
	}
	return api + "/torrents/requestdl?" + q.Encode()
}

// ── API helpers ──────────────────────────────────────────────────────────────

type tbResponse struct {
	Success bool            `json:"success"`
	Detail  string          `json:"detail"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
}

type tbCreateData struct {
	TorrentID int `json:"torrent_id"`
	ID        int `json:"id"`
}

type tbTorrent struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	Cached        bool      `json:"cached"`
	Progress      float64   `json:"progress"`
	DownloadState string    `json:"download_state"`
	Files         []tbFile  `json:"files"`
}

type tbFile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (s *service) createTorrent(ctx context.Context, magnet string) (int, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("magnet", magnet)
	_ = mw.WriteField("seed", "1")
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		api+"/torrents/createtorrent", &buf)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	var resp tbResponse
	if err := s.do(req, &resp); err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("%s", s.errMsg(resp))
	}

	var data tbCreateData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return 0, fmt.Errorf("parse create response: %w", err)
	}
	id := data.TorrentID
	if id == 0 {
		id = data.ID
	}
	if id == 0 {
		return 0, fmt.Errorf("no torrent ID in response")
	}
	return id, nil
}

func (s *service) waitUntilCached(ctx context.Context, torrentID int) (*tbTorrent, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("torbox: timed out waiting for torrent to cache — try again in a minute")
		case <-time.After(pollInterval):
		}

		torrent, err := s.fetchTorrent(ctx, torrentID)
		if err != nil {
			// transient — keep polling
			continue
		}
		if torrent.Cached || torrent.DownloadState == "cached" || torrent.Progress >= 1.0 {
			return torrent, nil
		}
	}
}

func (s *service) fetchTorrent(ctx context.Context, torrentID int) (*tbTorrent, error) {
	u := fmt.Sprintf("%s/torrents/mylist?id=%d&bypass_cache=true", api, torrentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	var resp tbResponse
	if err := s.do(req, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("%s", s.errMsg(resp))
	}

	// data is either an object or a single-element array
	var torrent tbTorrent
	if err := json.Unmarshal(resp.Data, &torrent); err != nil {
		var arr []tbTorrent
		if err2 := json.Unmarshal(resp.Data, &arr); err2 != nil || len(arr) == 0 {
			return nil, fmt.Errorf("parse mylist response: %w", err)
		}
		torrent = arr[0]
	}
	return &torrent, nil
}

func (s *service) do(req *http.Request, out interface{}) error {
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (s *service) errMsg(r tbResponse) string {
	if r.Detail != "" {
		return r.Detail
	}
	if r.Error != "" {
		return r.Error
	}
	return "unknown error"
}
