package download

import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var mockFetcherMeta = fetcher.FetcherMeta{
	Req: &base.Request{
		URL: "https://example.com/test.zip",
	},
	Opts: &base.Options{
		Path: "/downloads",
	},
	Res: &base.Resource{
		Size: 1024 * 1024,
		Files: []*base.FileInfo{
			{Name: "test.zip", Size: 1024 * 1024},
		},
	},
}

func TestWebhook_TriggerOnDone(t *testing.T) {
	receivedData := make(chan *WebhookData, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		var data WebhookData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Errorf("Failed to decode data: %v", err)
			return
		}
		receivedData <- &data
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Configure webhook URLs
		cfg, _ := downloader.GetConfig()
		if cfg.Extra == nil {
			cfg.Extra = make(map[string]any)
		}
		cfg.Extra["webhookUrls"] = []string{server.URL}
		downloader.PutConfig(cfg)

		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger webhook
		downloader.triggerWebhooks(WebhookEventDownloadDone, task, nil)

		select {
		case data := <-receivedData:
			if data.Event != WebhookEventDownloadDone {
				t.Errorf("Expected event 'DOWNLOAD_DONE', got '%s'", data.Event)
			}
			if data.Payload == nil || data.Payload.Task == nil {
				t.Error("Expected payload.task to be present")
			} else if data.Payload.Task.ID != task.ID {
				t.Errorf("Expected task ID '%s', got '%s'", task.ID, data.Payload.Task.ID)
			}
			if data.Time == 0 {
				t.Error("Expected time to be set")
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_TriggerOnError(t *testing.T) {
	receivedData := make(chan *WebhookData, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data WebhookData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Errorf("Failed to decode data: %v", err)
			return
		}
		receivedData <- &data
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Configure webhook URLs
		cfg, _ := downloader.GetConfig()
		if cfg.Extra == nil {
			cfg.Extra = make(map[string]any)
		}
		cfg.Extra["webhookUrls"] = []string{server.URL}
		downloader.PutConfig(cfg)

		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger webhook with error
		testError := http.ErrServerClosed
		downloader.triggerWebhooks(WebhookEventDownloadError, task, testError)

		select {
		case data := <-receivedData:
			if data.Event != WebhookEventDownloadError {
				t.Errorf("Expected event 'DOWNLOAD_ERROR', got '%s'", data.Event)
			}
			if data.Payload == nil || data.Payload.Task == nil {
				t.Error("Expected payload.task to be present")
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_SendTestWebhook(t *testing.T) {
	receivedData := make(chan *WebhookData, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data WebhookData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Errorf("Failed to decode data: %v", err)
			return
		}
		receivedData <- &data
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Configure webhook URLs
		cfg, _ := downloader.GetConfig()
		if cfg.Extra == nil {
			cfg.Extra = make(map[string]any)
		}
		cfg.Extra["webhookUrls"] = []string{server.URL}
		downloader.PutConfig(cfg)

		// Send test webhook
		err := downloader.SendTestWebhook()
		if err != nil {
			t.Errorf("SendTestWebhook failed: %v", err)
		}

		select {
		case data := <-receivedData:
			if data.Event != WebhookEventDownloadDone {
				t.Errorf("Expected event 'DOWNLOAD_DONE', got '%s'", data.Event)
			}
			if data.Payload == nil || data.Payload.Task == nil {
				t.Error("Expected payload.task to be present")
			}
			if data.Time == 0 {
				t.Error("Expected time to be set")
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_NoWebhookConfigured(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger webhook (should not panic with no webhooks configured)
		downloader.triggerWebhooks(WebhookEventDownloadDone, task, nil)

		// Send test webhook (should not panic)
		err := downloader.SendTestWebhook()
		if err != nil {
			t.Errorf("SendTestWebhook failed: %v", err)
		}
	})
}

func TestWebhook_MultipleUrls(t *testing.T) {
	count := 0
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Configure multiple webhook URLs
		cfg, _ := downloader.GetConfig()
		if cfg.Extra == nil {
			cfg.Extra = make(map[string]any)
		}
		cfg.Extra["webhookUrls"] = []string{server1.URL, server2.URL}
		downloader.PutConfig(cfg)

		// Create a mock task
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		// Trigger webhook
		downloader.triggerWebhooks(WebhookEventDownloadDone, task, nil)

		// Wait for webhooks
		time.Sleep(500 * time.Millisecond)

		if count != 2 {
			t.Errorf("Expected 2 webhook calls, got %d", count)
		}
	})
}

func TestWebhook_TestWebhookFailsOnNon200(t *testing.T) {
	// Test that SendTestWebhook returns error for non-200 status codes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // 500
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Configure webhook URLs
		cfg, _ := downloader.GetConfig()
		if cfg.Extra == nil {
			cfg.Extra = make(map[string]any)
		}
		cfg.Extra["webhookUrls"] = []string{server.URL}
		downloader.PutConfig(cfg)

		// Send test webhook - should fail with non-200 status
		err := downloader.SendTestWebhook()
		if err == nil {
			t.Error("Expected SendTestWebhook to return error for non-200 status")
		}
	})
}

func TestWebhook_TestWebhookFailsOn201(t *testing.T) {
	// Test that SendTestWebhook returns error for 201 (only 200 is success)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated) // 201
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Configure webhook URLs
		cfg, _ := downloader.GetConfig()
		if cfg.Extra == nil {
			cfg.Extra = make(map[string]any)
		}
		cfg.Extra["webhookUrls"] = []string{server.URL}
		downloader.PutConfig(cfg)

		// Send test webhook - should fail with 201 status (only 200 is success)
		err := downloader.SendTestWebhook()
		if err == nil {
			t.Error("Expected SendTestWebhook to return error for 201 status")
		}
	})
}

func TestWebhook_TestWebhookUrl(t *testing.T) {
	receivedData := make(chan *WebhookData, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data WebhookData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Errorf("Failed to decode data: %v", err)
			return
		}
		receivedData <- &data
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		// Test single URL
		err := downloader.TestWebhookUrl(server.URL)
		if err != nil {
			t.Errorf("TestWebhookUrl failed: %v", err)
		}

		select {
		case data := <-receivedData:
			if data.Event != WebhookEventDownloadDone {
				t.Errorf("Expected event 'DOWNLOAD_DONE', got '%s'", data.Event)
			}
			if data.Payload == nil || data.Payload.Task == nil {
				t.Error("Expected payload.task to be present")
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_TestWebhookUrlEmpty(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		// Test with empty URL - should return error
		err := downloader.TestWebhookUrl("")
		if err == nil {
			t.Error("Expected TestWebhookUrl to return error for empty URL")
		}
	})
}

func setupWebhookTest(t *testing.T, fn func(downloader *Downloader)) {
	defaultDownloader.Setup()
	defaultDownloader.cfg.StorageDir = ".test_storage"
	defaultDownloader.cfg.DownloadDir = ".test_download"
	defer func() {
		defaultDownloader.Clear()
		os.RemoveAll(defaultDownloader.cfg.StorageDir)
		os.RemoveAll(defaultDownloader.cfg.DownloadDir)
	}()
	fn(defaultDownloader)
}
