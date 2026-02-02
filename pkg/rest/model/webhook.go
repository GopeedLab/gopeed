package model

// TestWebhookReq is the request body for testing a single webhook URL
type TestWebhookReq struct {
	URL string `json:"url"`
}
