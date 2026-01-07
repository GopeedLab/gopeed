package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/GopeedLab/gopeed/pkg/base"
	fhttp "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// ============================================================================
// HTTP Request/Client Building
// ============================================================================

func (f *Fetcher) buildRequest(ctx context.Context, req *base.Request) (httpReq *http.Request, err error) {
	var reqUrl string
	f.redirectLock.Lock()
	if f.redirectURL != "" {
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

func (f *Fetcher) buildClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
		Proxy: f.ctl.GetProxy(f.meta.Req.Proxy),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: f.meta.Req.SkipVerifyCert,
		},
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
func parseFilenameExtended(cd string) string {
	lower := strings.ToLower(cd)
	idx := strings.Index(lower, "filename*=")
	if idx == -1 {
		return ""
	}

	value := cd[idx+len("filename*="):]
	if endIdx := strings.Index(value, ";"); endIdx != -1 {
		value = value[:endIdx]
	}
	value = strings.TrimSpace(value)

	// Try charset''encoded format (e.g., UTF-8''%E4%B8%AD%E6%96%87.txt)
	parts := strings.SplitN(value, "''", 2)
	if len(parts) == 2 {
		decoded, err := url.QueryUnescape(parts[1])
		if err == nil {
			return decoded
		}
	}

	// Try charset'language'encoded format
	parts = strings.SplitN(value, "'", 3)
	if len(parts) >= 3 {
		decoded, err := url.QueryUnescape(parts[2])
		if err == nil {
			return decoded
		}
	}

	return ""
}

// decodeFilenameParam decodes filename parameter value
func decodeFilenameParam(filename string) string {
	// Handle RFC 2047 encoded word (=?charset?encoding?text?=)
	if strings.HasPrefix(filename, "=?") {
		decoder := new(mime.WordDecoder)
		normalizedFilename := strings.Replace(filename, "UTF8", "UTF-8", 1)
		if decoded, err := decoder.Decode(normalizedFilename); err == nil {
			return decoded
		}
	}

	// Try URL decoding
	decoded := util.TryUrlQueryUnescape(filename)

	// If not valid UTF-8, try GBK decoding (common for Chinese websites)
	if !utf8.ValidString(decoded) {
		if gbkDecoded := tryDecodeGBK(decoded); gbkDecoded != "" {
			return gbkDecoded
		}
	}

	return decoded
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
	if endIdx := strings.Index(value, ";"); endIdx != -1 {
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
