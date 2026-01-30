package model

// TestWebhookReq is the request body for testing a single webhook URL
type TestWebhookReq struct {
	URL string `json:"url"`
}

// TestScriptReq is the request body for testing a single script path
type TestScriptReq struct {
	Path string `json:"path"`
}
