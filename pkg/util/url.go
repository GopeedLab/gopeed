package util

import (
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"
)

const FileSchema = "FILE"

func ParseSchema(url string) string {
	index := strings.Index(url, ":")
	// if no schema or windows path like C:\a.txt, return FILE
	if index == -1 || index == 1 {
		return FileSchema
	}
	schema := url[:index]
	if schema == "data" {
		schema, _ = ParseDataUri(url)
	}
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
	// 解码Base64数据
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
