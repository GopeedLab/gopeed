package realdebrid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GopeedLab/gopeed/internal/debrid/types"
)

const (
	api            = "https://api.real-debrid.com/rest/1.0"
	pollInterval   = 3 * time.Second
	defaultTimeout = 90 * time.Second
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
	return types.ServiceRealDebrid
}

// Resolve adds the magnet to Real-Debrid, waits for it to become available,
// then unrestricts each file link and returns the CDN URLs.
func (s *service) Resolve(ctx context.Context, magnetOrTorrent string) ([]types.File, error) {
	// 1. Add magnet
	torrentID, err := s.addMagnet(ctx, magnetOrTorrent)
	if err != nil {
		return nil, fmt.Errorf("realdebrid: add magnet: %w", err)
	}

	// 2. Select all files (required before RD starts downloading)
	if err := s.selectFiles(ctx, torrentID); err != nil {
		return nil, fmt.Errorf("realdebrid: select files: %w", err)
	}

	// 3. Poll until links are ready
	pollCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	info, err := s.waitUntilReady(pollCtx, torrentID)
	if err != nil {
		return nil, err
	}

	// 4. Unrestrict each link
	result := make([]types.File, 0, len(info.Links))
	for i, link := range info.Links {
		unrestricted, err := s.unrestrictLink(ctx, link)
		if err != nil {
			return nil, fmt.Errorf("realdebrid: unrestrict link %d: %w", i, err)
		}
		result = append(result, types.File{
			Name: unrestricted.Filename,
			Size: unrestricted.Filesize,
			URL:  unrestricted.Download,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("realdebrid: no download links returned")
	}
	return result, nil
}

// ── API types ────────────────────────────────────────────────────────────────

type rdAddMagnetResp struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

type rdTorrentInfo struct {
	ID       string   `json:"id"`
	Filename string   `json:"filename"`
	Bytes    int64    `json:"bytes"`
	Status   string   `json:"status"` // "downloaded" = ready
	Links    []string `json:"links"`
	Files    []struct {
		ID       int    `json:"id"`
		Path     string `json:"path"`
		Bytes    int64  `json:"bytes"`
		Selected int    `json:"selected"`
	} `json:"files"`
}

type rdUnrestrictResp struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
	Link     string `json:"link"`
	Download string `json:"download"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (s *service) addMagnet(ctx context.Context, magnet string) (string, error) {
	form := url.Values{"magnet": {magnet}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		api+"/torrents/addMagnet", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	var resp rdAddMagnetResp
	if err := s.do(req, &resp); err != nil {
		return "", err
	}
	if resp.ID == "" {
		return "", fmt.Errorf("no torrent ID returned")
	}
	return resp.ID, nil
}

func (s *service) selectFiles(ctx context.Context, torrentID string) error {
	form := url.Values{"files": {"all"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		api+"/torrents/selectFiles/"+torrentID, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (s *service) waitUntilReady(ctx context.Context, torrentID string) (*rdTorrentInfo, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("realdebrid: timed out waiting for torrent — try again in a minute")
		case <-time.After(pollInterval):
		}

		info, err := s.getTorrentInfo(ctx, torrentID)
		if err != nil {
			continue // transient
		}
		if info.Status == "downloaded" {
			return info, nil
		}
	}
}

func (s *service) getTorrentInfo(ctx context.Context, torrentID string) (*rdTorrentInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		api+"/torrents/info/"+torrentID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	var info rdTorrentInfo
	return &info, s.do(req, &info)
}

func (s *service) unrestrictLink(ctx context.Context, link string) (*rdUnrestrictResp, error) {
	form := url.Values{"link": {link}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		api+"/unrestrict/link", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	var resp rdUnrestrictResp
	return &resp, s.do(req, &resp)
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
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}
