package download

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
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
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL},
	}
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
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL},
	}
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
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL},
	}
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
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server1.URL, server2.URL},
	}
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
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL},
	}
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
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL},
	}
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

func TestWebhook_GetWebhookUrls_EmptyConfig(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil, got %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_NoExtraField(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = nil
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil, got %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_NoWebhookUrlsKey(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{Enable: true}  // No URLs set
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil, got %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_StringSlice(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{"http://example.com", "http://example2.com"},
	}
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if len(urls) != 2 {
			t.Errorf("Expected 2 URLs, got %d", len(urls))
		}
		if urls[0] != "http://example.com" || urls[1] != "http://example2.com" {
			t.Errorf("URLs don't match expected values: %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_InterfaceSlice(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{"http://example.com", "http://example2.com"},
	}
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if len(urls) != 2 {
			t.Errorf("Expected 2 URLs, got %d", len(urls))
		}
		if urls[0] != "http://example.com" || urls[1] != "http://example2.com" {
			t.Errorf("URLs don't match expected values: %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_EmptyStringSlice(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Extra = make(map[string]any)
		cfg.Webhook.URLs = []string{}
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil, got %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_InterfaceSliceWithEmptyStrings(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{"", "", ""},
	}
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil for all empty strings, got %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_InterfaceSliceMixedTypes(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
			Enable: true,
			URLs:   []string{"http://example.com", "", "http://example2.com", ""},
		}
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if len(urls) != 2 {
			t.Errorf("Expected 2 valid URLs (ignoring empty strings), got %d: %v", len(urls), urls)
		}
		if urls[0] != "http://example.com" || urls[1] != "http://example2.com" {
			t.Errorf("URLs don't match expected values: %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_InvalidType(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{Enable: false}  // Disabled webhook
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil for disabled webhook, got %v", urls)
		}
	})
}

func TestWebhook_GetWebhookUrls_DisabledWebhook(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
			Enable: false,
			URLs:   []string{"http://example.com"},
		}
		downloader.PutConfig(cfg)

		urls := downloader.getWebhookUrls()
		if urls != nil {
			t.Errorf("Expected nil for disabled webhook even with URLs, got %v", urls)
		}
	})
}

func TestWebhook_SendWebhookToUrl_Success(t *testing.T) {
	receivedData := make(chan *WebhookData, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("User-Agent") != "Gopeed-Webhook/1.0" {
			t.Errorf("Expected User-Agent Gopeed-Webhook/1.0, got %s", r.Header.Get("User-Agent"))
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
		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		data := &WebhookData{
			Event: WebhookEventDownloadDone,
			Time:  time.Now().UnixMilli(),
			Payload: &WebhookPayload{
				Task: task,
			},
		}

		statusCode, err := downloader.sendWebhookToUrl(server.URL, data)
		if err != nil {
			t.Errorf("sendWebhookToUrl failed: %v", err)
		}
		if statusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", statusCode)
		}

		select {
		case received := <-receivedData:
			if received.Event != WebhookEventDownloadDone {
				t.Errorf("Expected event DOWNLOAD_DONE, got %s", received.Event)
			}
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_SendWebhookToUrl_EmptyUrl(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		data := &WebhookData{
			Event: WebhookEventDownloadDone,
			Time:  time.Now().UnixMilli(),
		}

		_, err := downloader.sendWebhookToUrl("", data)
		if err == nil {
			t.Error("Expected error for empty URL")
		}
		if err.Error() != "webhook URL is empty" {
			t.Errorf("Expected 'webhook URL is empty' error, got: %v", err)
		}
	})
}

func TestWebhook_SendWebhookToUrl_InvalidUrl(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		data := &WebhookData{
			Event: WebhookEventDownloadDone,
			Time:  time.Now().UnixMilli(),
		}

		_, err := downloader.sendWebhookToUrl("://invalid-url", data)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})
}

func TestWebhook_SendWebhookToUrl_NonExistentHost(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		data := &WebhookData{
			Event: WebhookEventDownloadDone,
			Time:  time.Now().UnixMilli(),
		}

		_, err := downloader.sendWebhookToUrl("http://non-existent-host-12345.example", data)
		if err == nil {
			t.Error("Expected error for non-existent host")
		}
	})
}

func TestWebhook_SendWebhookToUrl_Timeout(t *testing.T) {
	// Create a server that delays response beyond webhook timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // Longer than webhookTimeout (10s)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		data := &WebhookData{
			Event: WebhookEventDownloadDone,
			Time:  time.Now().UnixMilli(),
		}

		_, err := downloader.sendWebhookToUrl(server.URL, data)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

func TestWebhook_SendWebhookToUrl_VariousStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"204 No Content", http.StatusNoContent},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			setupWebhookTest(t, func(downloader *Downloader) {
				data := &WebhookData{
					Event: WebhookEventDownloadDone,
					Time:  time.Now().UnixMilli(),
				}

				statusCode, err := downloader.sendWebhookToUrl(server.URL, data)
				if err != nil {
					t.Errorf("sendWebhookToUrl failed: %v", err)
				}
				if statusCode != tc.statusCode {
					t.Errorf("Expected status %d, got %d", tc.statusCode, statusCode)
				}
			})
		})
	}
}

func TestWebhook_TriggerWebhooks_EmptyUrlSkipped(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL, "", server.URL},
	}
		downloader.PutConfig(cfg)

		task := NewTask()
		task.Protocol = "http"
		task.Meta = &mockFetcherMeta

		downloader.triggerWebhooks(WebhookEventDownloadDone, task, nil)

		time.Sleep(500 * time.Millisecond)

		if requestCount != 2 {
			t.Errorf("Expected 2 requests (empty URL should be skipped), got %d", requestCount)
		}
	})
}

func TestWebhook_WebhookDataStructure(t *testing.T) {
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
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server.URL},
	}
		downloader.PutConfig(cfg)

		task := NewTask()
		task.Protocol = "http"
		task.Status = base.DownloadStatusDone
		task.Meta = &mockFetcherMeta

		downloader.triggerWebhooks(WebhookEventDownloadDone, task, nil)

		select {
		case data := <-receivedData:
			// Verify event
			if data.Event != WebhookEventDownloadDone {
				t.Errorf("Expected event DOWNLOAD_DONE, got %s", data.Event)
			}
			// Verify time is set
			if data.Time == 0 {
				t.Error("Expected time to be set")
			}
			// Verify payload
			if data.Payload == nil {
				t.Error("Expected payload to be present")
			}
			if data.Payload.Task == nil {
				t.Error("Expected task in payload to be present")
			}
			// Verify task is cloned (has same ID but different pointer)
			if data.Payload.Task.ID != task.ID {
				t.Errorf("Expected task ID %s, got %s", task.ID, data.Payload.Task.ID)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
		}
	})
}

func TestWebhook_SendTestWebhook_EmptyUrls(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Extra = make(map[string]any)
		cfg.Webhook.URLs = []string{}
		downloader.PutConfig(cfg)

		err := downloader.SendTestWebhook()
		if err != nil {
			t.Errorf("Expected no error for empty URLs, got: %v", err)
		}
	})
}

func TestWebhook_SendTestWebhook_MixedResults(t *testing.T) {
	// First server returns 200
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	// Second server returns 500
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	setupWebhookTest(t, func(downloader *Downloader) {
		cfg, _ := downloader.GetConfig()
		cfg.Webhook = &base.WebhookConfig{
		Enable: true,
		URLs: []string{server1.URL, server2.URL},
	}
		downloader.PutConfig(cfg)

		// Should fail because server2 returns 500
		err := downloader.SendTestWebhook()
		if err == nil {
			t.Error("Expected error when one server returns non-200 status")
		}
	})
}

func TestWebhook_TestWebhookUrl_InvalidUrl(t *testing.T) {
	setupWebhookTest(t, func(downloader *Downloader) {
		err := downloader.TestWebhookUrl("://invalid")
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})
}

func TestWebhook_TestWebhookUrl_VerifyTestPayload(t *testing.T) {
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
		err := downloader.TestWebhookUrl(server.URL)
		if err != nil {
			t.Errorf("TestWebhookUrl failed: %v", err)
		}

		select {
		case data := <-receivedData:
			// Verify it's a test webhook
			if data.Event != WebhookEventDownloadDone {
				t.Errorf("Expected event DOWNLOAD_DONE, got %s", data.Event)
			}
			if data.Payload == nil || data.Payload.Task == nil {
				t.Error("Expected payload with task")
			}
			// Verify test task properties
			task := data.Payload.Task
			if task.Protocol != "http" {
				t.Errorf("Expected protocol 'http', got '%s'", task.Protocol)
			}
			if task.Status != base.DownloadStatusDone {
				t.Errorf("Expected status Done, got %s", task.Status)
			}
			if task.Meta == nil || task.Meta.Req == nil {
				t.Error("Expected meta and request in test task")
			}
			if task.Meta.Req.URL != "https://example.com/test-file.zip" {
				t.Errorf("Expected test URL, got %s", task.Meta.Req.URL)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for webhook")
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
