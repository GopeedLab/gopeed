package util

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func ParseSchema(url string) string {
	index := strings.Index(url, ":")
	if index == -1 || index == 1 {
		return ""
	}
	schema := url[:index]
	return strings.ToUpper(schema)
}

// ParseDataUri parses a data URI and returns the MIME type and decode data.
func ParseDataUri(uri string) (string, []byte) {
	re := regexp.MustCompile(`^data:(.*);base64,(.*)$`)
	matches := re.FindStringSubmatch(uri)
	if len(matches) != 3 {
		return "", nil
	}
	mime := matches[1]
	base64Data := matches[2]
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", nil
	}
	return mime, data
}

// BuildProxyUrl builds a proxy url with given host, username and password.
func BuildProxyUrl(scheme, host, usr, pwd string) *url.URL {
	var user *url.Userinfo
	if usr != "" && pwd != "" {
		user = url.UserPassword(usr, pwd)
	}
	return &url.URL{
		Scheme: scheme,
		User:   user,
		Host:   host,
	}
}

// ProxyUrlToHandler gets the proxy handler from the proxy url.
func ProxyUrlToHandler(proxyUrl *url.URL) func(*http.Request) (*url.URL, error) {
	if proxyUrl == nil {
		return nil
	}
	if proxyUrl.Scheme == "system" {
		return http.ProxyFromEnvironment
	}
	return http.ProxyURL(proxyUrl)
}

// TryUrlQueryUnescape tries to unescape a URL-encoded string.
//
// If unescaping fails, it returns the original string.
func TryUrlQueryUnescape(s string) string {
	if decoded, err := url.QueryUnescape(s); err == nil {
		return decoded
	}
	return s
}

// TryUrlPathUnescape tries to unescape a URL path-encoded string.
// Unlike QueryUnescape, PathUnescape does not treat '+' as a space.
// This is the correct function to use for decoding URL paths and filenames
// where %2B should decode to '+', not to a space.
//
// If unescaping fails, it returns the original string.
func TryUrlPathUnescape(s string) string {
	if decoded, err := url.PathUnescape(s); err == nil {
		return decoded
	}
	return s
}
