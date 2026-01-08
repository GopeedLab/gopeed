package download

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

const (
	webhookTimeout = 10 * time.Second
)

// WebhookEvent represents the type of webhook event
type WebhookEvent string

const (
	WebhookEventDownloadDone  WebhookEvent = "DOWNLOAD_DONE"
	WebhookEventDownloadError WebhookEvent = "DOWNLOAD_ERROR"
)

// WebhookData is the data sent to webhook URLs
type WebhookData struct {
	Event   WebhookEvent    `json:"event"`
	Time    int64           `json:"time"` // Unix timestamp in milliseconds
	Payload *WebhookPayload `json:"payload"`
}

// WebhookPayload contains the task data
type WebhookPayload struct {
	Task *Task `json:"task"`
}

// getWebhookUrls extracts webhook URLs from config
// Supports both new webhook config format and legacy extra field for backward compatibility
func (d *Downloader) getWebhookUrls() []string {
	cfg := d.cfg.DownloaderStoreConfig
	if cfg == nil {
		return nil
	}

	// Try new webhook config first
	if cfg.Webhook != nil && cfg.Webhook.Enable && len(cfg.Webhook.URLs) > 0 {
		urls := make([]string, 0, len(cfg.Webhook.URLs))
		for _, url := range cfg.Webhook.URLs {
			if url != "" {
				urls = append(urls, url)
			}
		}
		if len(urls) > 0 {
			return urls
		}
	}

	// Fall back to legacy extra field for backward compatibility
	if cfg.Extra == nil {
		return nil
	}

	webhookUrls, ok := cfg.Extra["webhookUrls"]
	if !ok {
		return nil
	}

	// Try direct string slice first
	if urlsStr, ok := webhookUrls.([]string); ok {
		if len(urlsStr) == 0 {
			return nil
		}
		return urlsStr
	}

	// Convert []interface{} to []string
	if urlsInterface, ok := webhookUrls.([]any); ok {
		if len(urlsInterface) == 0 {
			return nil
		}
		urls := make([]string, 0, len(urlsInterface))
		for _, urlInterface := range urlsInterface {
			if url, ok := urlInterface.(string); ok && url != "" {
				urls = append(urls, url)
			}
		}
		if len(urls) == 0 {
			return nil
		}
		return urls
	}

	return nil
}

// sendWebhookToUrl sends webhook data to a single URL
// Returns the HTTP status code and any error that occurred
func (d *Downloader) sendWebhookToUrl(url string, data *WebhookData) (int, error) {
	if url == "" {
		return 0, fmt.Errorf("webhook URL is empty")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}

	client := &http.Client{
		Timeout: webhookTimeout,
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Gopeed-Webhook/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// triggerWebhooks sends webhook notifications to all configured URLs
func (d *Downloader) triggerWebhooks(event WebhookEvent, task *Task, err error) {
	urls := d.getWebhookUrls()
	if len(urls) == 0 {
		return
	}

	data := &WebhookData{
		Event: event,
		Time:  time.Now().UnixMilli(),
		Payload: &WebhookPayload{
			Task: task.clone(),
		},
	}

	go d.sendWebhooks(urls, data)
}

func (d *Downloader) sendWebhooks(urls []string, data *WebhookData) {
	for _, url := range urls {
		if url == "" {
			continue
		}
		go func(webhookUrl string) {
			statusCode, err := d.sendWebhookToUrl(webhookUrl, data)
			if err != nil {
				d.Logger.Warn().Err(err).Str("url", webhookUrl).Msg("webhook: failed to send request")
				return
			}
			if statusCode >= 200 && statusCode < 300 {
				d.Logger.Debug().Str("url", webhookUrl).Int("status", statusCode).Msg("webhook: sent successfully")
			} else {
				d.Logger.Warn().Str("url", webhookUrl).Int("status", statusCode).Msg("webhook: received non-success status")
			}
		}(url)
	}
}

// SendTestWebhook sends a test webhook with a simulated payload
// Returns error if any webhook URL does not respond with HTTP 200
func (d *Downloader) SendTestWebhook() error {
	urls := d.getWebhookUrls()
	if len(urls) == 0 {
		return nil
	}

	for _, url := range urls {
		if url == "" {
			continue
		}
		if err := d.TestWebhookUrl(url); err != nil {
			return err
		}
	}

	return nil
}

// TestWebhookUrl tests a single webhook URL with a simulated payload
// Returns error if the URL does not respond with HTTP 200
func (d *Downloader) TestWebhookUrl(url string) error {
	// Create a simulated test task with minimal required fields
	testTask := NewTask()
	testTask.Protocol = "http"
	testTask.Status = base.DownloadStatusDone
	testTask.Meta = &fetcher.FetcherMeta{
		Req: &base.Request{
			URL: "https://example.com/test-file.zip",
		},
		Opts: &base.Options{
			Name: "test-file.zip",
			Path: "/downloads",
		},
		Res: &base.Resource{
			Size: 1024 * 1024 * 100, // 100MB
			Files: []*base.FileInfo{
				{Name: "test-file.zip", Size: 1024 * 1024 * 100},
			},
		},
	}

	// Create test data
	testData := &WebhookData{
		Event: WebhookEventDownloadDone,
		Time:  time.Now().UnixMilli(),
		Payload: &WebhookPayload{
			Task: testTask,
		},
	}

	statusCode, err := d.sendWebhookToUrl(url, testData)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("webhook test failed: %s returned status %d", url, statusCode)
	}

	return nil
}
