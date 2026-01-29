package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type RequestError struct {
	Code int
}

func NewRequestError(code int) *RequestError {
	return &RequestError{Code: code}
}

func (re *RequestError) Error() string {
	return fmt.Sprintf("http request fail, code:%d", re.Code)
}

func isFailureExemptHTTPCode(code int) bool {
	if code >= 500 && code <= 599 {
		return true
	}

	switch code {
	case 429, 408, 440, 499:
		return true
	default:
		return false
	}
}

func shouldCountHTTPFailure(err error) bool {
	var re *RequestError
	if !errors.As(err, &re) {
		return false
	}

	return !isFailureExemptHTTPCode(re.Code)
}

func extractRequestError(err error) *RequestError {
	var re *RequestError
	if errors.As(err, &re) {
		return re
	}

	return nil
}

// buildRequest creates an HTTP request using the redirect URL if available.
func (f *Fetcher) buildRequest(ctx context.Context, req *base.Request) (httpReq *http.Request, err error) {
	return f.buildRequestWithURL(ctx, req, true)
}

// buildRequestWithOriginalURL creates an HTTP request using the original URL.
// This is used for retrying when the redirect URL has expired.
func (f *Fetcher) buildRequestWithOriginalURL(ctx context.Context, req *base.Request) (httpReq *http.Request, err error) {
	return f.buildRequestWithURL(ctx, req, false)
}

// buildRequestWithURL creates an HTTP request.
// If useRedirect is true and a redirect URL exists, it will be used; otherwise the original URL is used.
func (f *Fetcher) buildRequestWithURL(ctx context.Context, req *base.Request, useRedirect bool) (httpReq *http.Request, err error) {
	var reqUrl string
	f.redirectLock.Lock()
	if useRedirect && f.redirectURL != "" {
		reqUrl = f.redirectURL
	} else {
		reqUrl = req.URL
	}
	f.redirectLock.Unlock()

	var (
		method string
		body   io.Reader
	)
	headers := http.Header{}
	if req.Extra == nil {
		method = http.MethodGet
	} else {
		extra := req.Extra.(*fhttp.ReqExtra)
		if extra.Method != "" {
			method = extra.Method
		} else {
			method = http.MethodGet
		}
		if len(extra.Header) > 0 {
			for k, v := range extra.Header {
				headers.Set(k, strings.TrimSpace(v))
			}
		}
		if extra.Body != "" {
			body = bytes.NewBufferString(extra.Body)
		}
	}
	if _, ok := headers[base.HttpHeaderUserAgent]; !ok {
		headers.Set(base.HttpHeaderUserAgent, strings.TrimSpace(f.config.UserAgent))
	}

	if ctx != nil {
		httpReq, err = http.NewRequestWithContext(ctx, method, reqUrl, body)
	} else {
		httpReq, err = http.NewRequest(method, reqUrl, body)
	}
	if err != nil {
		return
	}
	httpReq.Header = headers
	if host := headers.Get(base.HttpHeaderHost); host != "" {
		httpReq.Host = host
	}
	return httpReq, nil
}

// updateRedirectURL updates the redirect URL from the response.
// This is called when a request using the original URL succeeds after the redirect URL expired.
func (f *Fetcher) updateRedirectURL(resp *http.Response) {
	if resp != nil && resp.Request != nil {
		newRedirectURL := resp.Request.URL.String()
		f.redirectLock.Lock()
		f.redirectURL = newRedirectURL
		f.redirectLock.Unlock()
	}
}

// hasRedirectURL checks if a redirect URL exists and is different from the original URL.
func (f *Fetcher) hasRedirectURL() bool {
	f.redirectLock.Lock()
	defer f.redirectLock.Unlock()
	return f.redirectURL != "" && f.redirectURL != f.meta.Req.URL
}

// isRedirectExpiredError checks if the error indicates that the redirect URL may have expired.
// This includes 403 (Forbidden), 401 (Unauthorized), 410 (Gone), and network errors.
func isRedirectExpiredError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific HTTP error codes that might indicate URL expiration
	if re := extractRequestError(err); re != nil {
		switch re.Code {
		case 401, 403, 404, 410:
			return true
		}
	}

	return false
}

// tryFallbackToOriginalURL attempts to make a request using the original URL
// when the redirect URL has expired. Returns the response if successful.
func (f *Fetcher) tryFallbackToOriginalURL(ctx context.Context, client *http.Client, rangeStart, rangeEnd int64) (*http.Response, error) {
	httpReq, err := f.buildRequestWithOriginalURL(ctx, f.meta.Req)
	if err != nil {
		return nil, err
	}

	if f.meta.Res.Range && rangeEnd > 0 {
		httpReq.Header.Set(base.HttpHeaderRange,
			fmt.Sprintf(base.HttpHeaderRangeFormat, rangeStart, rangeEnd))
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != base.HttpCodeOK && resp.StatusCode != base.HttpCodePartialContent {
		resp.Body.Close()
		return nil, NewRequestError(resp.StatusCode)
	}

	return resp, nil
}

// buildClient creates an HTTP client with the default connection timeout.
// Used for resolve phase where we don't have connection time data yet.
func (f *Fetcher) buildClient() *http.Client {
	return f.buildClientWithTimeout(connectTimeout)
}

// buildFastFailClient creates an HTTP client with fast-fail timeout.
// Uses max(minFastFailTimeout, maxConnTime) for fast-fail retry during download phase.
func (f *Fetcher) buildFastFailClient() *http.Client {
	maxConn := f.maxConnTime.Load()
	if maxConn == 0 {
		// No successful connection yet, use default timeout
		return f.buildClientWithTimeout(connectTimeout)
	}

	timeout := maxConn
	if timeout >= minFastFailTimeout {
		// If greater than minFastFailTimeout, increase by 50% for safety margin
		timeout = int64(float64(timeout) * 1.5)
	} else {
		timeout = minFastFailTimeout
	}
	return f.buildClientWithTimeout(time.Duration(timeout))
}

// buildClientWithTimeout creates an HTTP client with the specified connection timeout.
func (f *Fetcher) buildClientWithTimeout(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
		Proxy: f.ctl.GetProxy(f.meta.Req.Proxy),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: f.meta.Req.SkipVerifyCert,
		},
		TLSHandshakeTimeout: timeout,
	}
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport: transport,
		Jar:       jar,
	}
}

// ============================================================================
// Filename Parsing
// ============================================================================

// parseFilename extracts filename from Content-Disposition header
func parseFilename(contentDisposition string) string {
	// Try RFC 5987 extended notation first (filename*=)
	if filename := parseFilenameExtended(contentDisposition); filename != "" {
		return filename
	}

	// Try standard MIME parsing
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err == nil {
		if filename := params["filename"]; filename != "" {
			return decodeFilenameParam(filename)
		}
	}

	// Fallback to manual parsing
	return parseFilenameFallback(contentDisposition)
}

// parseFilenameExtended handles RFC 5987 extended notation (filename*=)
// Format: filename*=charset'language'value (e.g., UTF-8”%E6%B5%8B%E8%AF%95.zip)
func parseFilenameExtended(cd string) string {
	lower := strings.ToLower(cd)
	idx := strings.Index(lower, "filename*=")
	if idx == -1 {
		return ""
	}

	value := cd[idx+len("filename*="):]

	// Find the end of the value using proper quote handling
	endIdx := findParamValueEnd(value)
	if endIdx != -1 {
		value = value[:endIdx]
	}
	value = strings.TrimSpace(value)

	// Try charset''encoded format (e.g., UTF-8''%E4%B8%AD%E6%96%87.txt)
	parts := strings.SplitN(value, "''", 2)
	if len(parts) == 2 {
		// Use PathUnescape to handle %2B correctly (should decode to +, not space)
		decoded, err := url.PathUnescape(parts[1])
		if err == nil {
			return decoded
		}
	}

	// Try charset'language'encoded format
	parts = strings.SplitN(value, "'", 3)
	if len(parts) >= 3 {
		// Use PathUnescape to handle %2B correctly (should decode to +, not space)
		decoded, err := url.PathUnescape(parts[2])
		if err == nil {
			return decoded
		}
	}

	return ""
}

// decodeFilenameParam decodes filename parameter value
// Handles HTML entities, MIME encoded-word, URL encoding, and GBK encoding fallback
func decodeFilenameParam(filename string) string {
	// First, unescape HTML entities (e.g., &amp; -> &, &lt; -> <, &gt; -> >)
	// This must be done before other decoding to handle cases where servers
	// HTML-encode special characters in filenames
	filename = unescapeHTMLEntities(filename)

	// Handle RFC 2047 encoded word (=?charset?encoding?text?=)
	if strings.HasPrefix(filename, "=?") {
		decoder := new(mime.WordDecoder)
		normalizedFilename := strings.Replace(filename, "UTF8", "UTF-8", 1)
		if decoded, err := decoder.Decode(normalizedFilename); err == nil {
			return decoded
		}
	}

	// Try URL decoding - use PathUnescape for filenames to handle %2B correctly
	decoded := util.TryUrlPathUnescape(filename)

	// If not valid UTF-8, try GBK decoding (common for Chinese websites)
	if !utf8.ValidString(decoded) {
		if gbkDecoded := tryDecodeGBK(decoded); gbkDecoded != "" {
			return gbkDecoded
		}
	}

	return decoded
}

// unescapeHTMLEntities unescapes common HTML entities in filenames
// This handles cases where servers HTML-encode special characters like & to &amp;
func unescapeHTMLEntities(s string) string {
	// Common HTML entities that might appear in filenames
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&apos;": "'",
	}

	result := s
	for entity, char := range replacements {
		result = strings.ReplaceAll(result, entity, char)
	}
	return result
}

// tryDecodeGBK attempts to decode string as GBK encoding
func tryDecodeGBK(s string) string {
	if len(s) == 0 {
		return ""
	}

	decoder := simplifiedchinese.GBK.NewDecoder()
	decoded, err := decoder.Bytes([]byte(s))
	if err != nil {
		return ""
	}
	result := string(decoded)
	if utf8.ValidString(result) {
		return result
	}
	return ""
}

// parseFilenameFallback is a fallback parser for non-standard Content-Disposition
func parseFilenameFallback(cd string) string {
	lower := strings.ToLower(cd)
	idx := strings.Index(lower, "filename=")
	if idx == -1 {
		return ""
	}

	value := cd[idx+len("filename="):]

	// Find the end of the value using proper quote handling
	endIdx := findParamValueEnd(value)
	if endIdx != -1 {
		value = value[:endIdx]
	}
	value = strings.TrimSpace(value)

	// Remove surrounding quotes
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return decodeFilenameParam(value)
}

// findParamValueEnd finds the end position of a parameter value in a Content-Disposition header.
// It correctly handles quoted values where semicolons inside quotes should not be treated as delimiters.
// It also handles HTML entities in unquoted values (e.g., &amp; should not be split at the semicolon).
// Returns the end index (exclusive) of the value, or -1 if it extends to the end of the string.
func findParamValueEnd(value string) int {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return 0
	}

	// If the value starts with a quote, find the matching closing quote
	if value[0] == '"' || value[0] == '\'' {
		quote := value[0]
		// Find the closing quote, handling escaped quotes
		for i := 1; i < len(value); i++ {
			if value[i] == quote {
				// Check if it's escaped
				if i > 0 && value[i-1] == '\\' {
					continue
				}
				// Found closing quote, now look for ; after it
				remaining := value[i+1:]
				if semiIdx := strings.Index(remaining, ";"); semiIdx != -1 {
					return i + 1 + semiIdx
				}
				return -1 // No semicolon after closing quote
			}
		}
		// No closing quote found, treat rest of string as value
		return -1
	}

	// Unquoted value - find the next semicolon that's not part of an HTML entity
	// HTML entities have the pattern &...;  (e.g., &amp; &lt; &gt; &quot; &#39;)
	for i := 0; i < len(value); i++ {
		if value[i] == ';' {
			// Check if this semicolon is part of an HTML entity
			// Look backwards for an & character
			isEntity := false
			if i > 0 {
				// Look for & before this semicolon (within reasonable distance, max 10 chars)
				for j := i - 1; j >= 0 && j >= i-10; j-- {
					if value[j] == '&' {
						// Found &, this semicolon might be part of an HTML entity
						// Check if there are only alphanumeric or # between & and ;
						entityChars := value[j+1 : i]
						if len(entityChars) > 0 && isValidHTMLEntityChars(entityChars) {
							isEntity = true
						}
						break
					}
					// If we hit whitespace or another special char, stop looking
					if value[j] == ' ' || value[j] == '"' || value[j] == '\'' {
						break
					}
				}
			}

			if !isEntity {
				return i
			}
		}
	}
	return -1 // No semicolon, extends to end
}

// isValidHTMLEntityChars checks if a string contains only valid HTML entity characters
// (alphanumeric and #, typically for entities like &amp; &lt; &#39; etc.)
func isValidHTMLEntityChars(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '#') {
			return false
		}
	}
	return true
}
