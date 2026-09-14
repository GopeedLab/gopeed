package webview

import "encoding/json"

const RPCEndpointPath = "/webview"

const (
	MethodProfileRemove     = "profile.remove"
	MethodIsAvailable       = "webview.isAvailable"
	MethodPageOpen          = "page.open"
	MethodPageAddInitScript = "page.addInitScript"
	MethodPageGoto          = "page.goto"
	MethodPageExecute       = "page.execute"
	MethodPageGetCookies    = "page.getCookies"
	MethodPageSetCookie     = "page.setCookie"
	MethodPageDeleteCookie  = "page.deleteCookie"
	MethodPageClearCookies  = "page.clearCookies"
	MethodPageClose         = "page.close"
)

const (
	ErrorCodeInvalidRequest   = "INVALID_REQUEST"
	ErrorCodeUnknownMethod    = "UNKNOWN_METHOD"
	ErrorCodeUnavailable      = "UNAVAILABLE"
	ErrorCodeBrowserNotFound  = "BROWSER_NOT_FOUND"
	ErrorCodePageNotFound     = "PAGE_NOT_FOUND"
	ErrorCodeNavigationFailed = "NAVIGATION_FAILED"
	ErrorCodeEvaluationFailed = "EVALUATION_FAILED"
	ErrorCodeTimeout          = "TIMEOUT"
	ErrorCodeInternal         = "INTERNAL_ERROR"
)

type RPCConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
	Token   string `json:"token,omitempty"`
}

func (c RPCConfig) Enabled() bool {
	return c.Network != "" && c.Address != ""
}

type RPCRequest struct {
	Method string `json:"method"`
	Params any    `json:"params"`
}

type RPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type IsAvailableParams struct{}

type IsAvailableResult struct {
	Available bool `json:"available"`
}

type PageOpenParams struct {
	ProfileID string `json:"profileId,omitempty"`
	DataPath  string `json:"dataPath,omitempty"`
	ProxyURL  string `json:"proxyUrl,omitempty"`
	Headless  bool   `json:"headless,omitempty"`
	Debug     bool   `json:"debug,omitempty"`
	Title     string `json:"title,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

func NewPageOpenParams(opts OpenOptions) PageOpenParams {
	return PageOpenParams{
		ProfileID: opts.ProfileID,
		DataPath:  opts.DataPath,
		ProxyURL:  opts.ProxyURL,
		Headless:  opts.Headless,
		Debug:     opts.Debug,
		Title:     opts.Title,
		Width:     opts.Width,
		Height:    opts.Height,
		UserAgent: opts.UserAgent,
	}
}

type PageOpenResult struct {
	PageID string `json:"pageId"`
}

type PageAddInitScriptParams struct {
	PageID string `json:"pageId"`
	Script string `json:"script"`
}

type PageGotoParams struct {
	PageID    string `json:"pageId"`
	URL       string `json:"url"`
	TimeoutMS int64  `json:"timeoutMs,omitempty"`
	WaitUntil string `json:"waitUntil,omitempty"`
}

func NewPageGotoParams(pageID string, url string, opts GotoOptions) PageGotoParams {
	return PageGotoParams{
		PageID:    pageID,
		URL:       url,
		TimeoutMS: opts.TimeoutMS,
		WaitUntil: opts.WaitUntil,
	}
}

type PageExecuteParams struct {
	PageID     string `json:"pageId"`
	Expression string `json:"expression"`
	Args       []any  `json:"args,omitempty"`
}

type PageGetCookiesParams struct {
	PageID string `json:"pageId"`
}

type PageSetCookieParams struct {
	PageID string `json:"pageId"`
	Cookie Cookie `json:"cookie"`
}

type PageDeleteCookieParams struct {
	PageID string `json:"pageId"`
	Cookie Cookie `json:"cookie"`
}

type PageClearCookiesParams struct {
	PageID string `json:"pageId"`
}

type PageCloseParams struct {
	PageID string `json:"pageId"`
}

type EmptyResult struct{}

// ProfileRemoveParams belongs to the private host RPC.
type ProfileRemoveParams struct {
	ProfileID string `json:"profileId"`
}
