package download

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

const (
	webhookTimeout = 10 * time.Second
)

// WebhookEvent represents the type of webhook event
type WebhookEvent string

const (
	WebhookEventDone  WebhookEvent = "done"
	WebhookEventError WebhookEvent = "error"
)

// WebhookPayload is the payload sent to webhook URLs
type WebhookPayload struct {
	Event WebhookEvent      `json:"event"`
	Task  *WebhookTask      `json:"task"`
	Error string            `json:"error,omitempty"`
	Time  int64             `json:"time"` // Unix timestamp in milliseconds
	Extra map[string]string `json:"extra,omitempty"`
}

// WebhookTask is a simplified task structure for webhook payload
type WebhookTask struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Protocol string `json:"protocol"`
}

func newWebhookTask(task *Task) *WebhookTask {
	wt := &WebhookTask{
		ID:       task.ID,
		Name:     task.Name(),
		Status:   string(task.Status),
		Protocol: task.Protocol,
	}
	if task.Meta != nil {
		if task.Meta.Req != nil {
			wt.URL = task.Meta.Req.URL
		}
		if task.Meta.Opts != nil {
			wt.Path = task.Meta.Opts.Path
		}
		if task.Meta.Res != nil {
			wt.Size = task.Meta.Res.Size
		}
	}
	return wt
}

// triggerWebhooks sends webhook notifications to all configured URLs
func (d *Downloader) triggerWebhooks(event WebhookEvent, task *Task, err error) {
	cfg := d.cfg.DownloaderStoreConfig
	if cfg == nil || cfg.Extra == nil {
		return
	}

	webhookUrls, ok := cfg.Extra["webhookUrls"]
	if !ok {
		return
	}

	// Convert interface to []string
	urls, ok := webhookUrls.([]interface{})
	if !ok {
		// Try direct string slice
		urlsStr, ok := webhookUrls.([]string)
		if !ok || len(urlsStr) == 0 {
			return
		}
		urls = make([]interface{}, len(urlsStr))
		for i, u := range urlsStr {
			urls[i] = u
		}
	}

	if len(urls) == 0 {
		return
	}

	payload := &WebhookPayload{
		Event: event,
		Task:  newWebhookTask(task),
		Time:  time.Now().UnixMilli(),
	}
	if err != nil {
		payload.Error = err.Error()
	}

	go d.sendWebhooks(urls, payload)
}

func (d *Downloader) sendWebhooks(urls []interface{}, payload *WebhookPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		d.Logger.Error().Err(err).Msg("webhook: failed to marshal payload")
		return
	}

	client := &http.Client{
		Timeout: webhookTimeout,
	}

	for _, urlInterface := range urls {
		url, ok := urlInterface.(string)
		if !ok || url == "" {
			continue
		}
		go func(webhookUrl string) {
			req, err := http.NewRequest(http.MethodPost, webhookUrl, bytes.NewBuffer(jsonData))
			if err != nil {
				d.Logger.Error().Err(err).Str("url", webhookUrl).Msg("webhook: failed to create request")
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "Gopeed-Webhook/1.0")

			resp, err := client.Do(req)
			if err != nil {
				d.Logger.Warn().Err(err).Str("url", webhookUrl).Msg("webhook: failed to send request")
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				d.Logger.Debug().Str("url", webhookUrl).Int("status", resp.StatusCode).Msg("webhook: sent successfully")
			} else {
				d.Logger.Warn().Str("url", webhookUrl).Int("status", resp.StatusCode).Msg("webhook: received non-success status")
			}
		}(url)
	}
}

// SendTestWebhook sends a test webhook with a simulated payload
func (d *Downloader) SendTestWebhook() error {
	cfg := d.cfg.DownloaderStoreConfig
	if cfg == nil || cfg.Extra == nil {
		return nil
	}

	webhookUrls, ok := cfg.Extra["webhookUrls"]
	if !ok {
		return nil
	}

	// Convert interface to []interface{}
	urls, ok := webhookUrls.([]interface{})
	if !ok {
		// Try direct string slice
		urlsStr, ok := webhookUrls.([]string)
		if !ok || len(urlsStr) == 0 {
			return nil
		}
		urls = make([]interface{}, len(urlsStr))
		for i, u := range urlsStr {
			urls[i] = u
		}
	}

	if len(urls) == 0 {
		return nil
	}

	// Create a simulated test payload
	testPayload := &WebhookPayload{
		Event: WebhookEventDone,
		Task: &WebhookTask{
			ID:       "test-task-id",
			Name:     "test-file.zip",
			Status:   "done",
			URL:      "https://example.com/test-file.zip",
			Path:     "/downloads",
			Size:     1024 * 1024 * 100, // 100MB
			Protocol: "http",
		},
		Time: time.Now().UnixMilli(),
		Extra: map[string]string{
			"test": "true",
		},
	}

	d.sendWebhooks(urls, testPayload)
	return nil
}
