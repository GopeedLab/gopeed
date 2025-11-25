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
	receivedPayload := make(chan *WebhookPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
			return
		}
		receivedPayload <- &payload
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
		downloader.triggerWebhooks(WebhookEventDone, task, nil)

		select {
		case payload := <-receivedPayload:
			if payload.Event != WebhookEventDone {
				t.Errorf("Expected event 'done', got '%s'", payload.Event)
			}
			if payload.Task.ID != task.ID {
				t.Errorf("Expected task ID '%s', got '%s'", task.ID, payload.Task.ID)
			}
			if payload.Error != "" {
				t.Errorf("Expected no error, got '%s'", payload.Error)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_TriggerOnError(t *testing.T) {
	receivedPayload := make(chan *WebhookPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
			return
		}
		receivedPayload <- &payload
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
		downloader.triggerWebhooks(WebhookEventError, task, testError)

		select {
		case payload := <-receivedPayload:
			if payload.Event != WebhookEventError {
				t.Errorf("Expected event 'error', got '%s'", payload.Event)
			}
			if payload.Error != testError.Error() {
				t.Errorf("Expected error '%s', got '%s'", testError.Error(), payload.Error)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_SendTestWebhook(t *testing.T) {
	receivedPayload := make(chan *WebhookPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
			return
		}
		receivedPayload <- &payload
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
		case payload := <-receivedPayload:
			if payload.Event != WebhookEventDone {
				t.Errorf("Expected event 'done', got '%s'", payload.Event)
			}
			if payload.Task.ID != "test-task-id" {
				t.Errorf("Expected task ID 'test-task-id', got '%s'", payload.Task.ID)
			}
			if payload.Extra == nil || payload.Extra["test"] != "true" {
				t.Error("Expected test=true in extra")
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
		downloader.triggerWebhooks(WebhookEventDone, task, nil)

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
		downloader.triggerWebhooks(WebhookEventDone, task, nil)

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
