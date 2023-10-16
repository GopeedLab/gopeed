package util

import (
	"net/url"
	"regexp"
	"strings"
)

// Match url with pattern by chrome extension match pattern style
// https://developer.chrome.com/docs/extensions/mv3/match_patterns/
func Match(pattern string, u string) bool {
	scheme, host, path := parsePattern(pattern)
	url, err := url.Parse(u)
	if err != nil {
		return false
	}
	if scheme != "*" && scheme != url.Scheme {
		return false
	}
	if !matchHost(host, url.Hostname()) {
		return false
	}
	if !matchPath(path, url.Path) {
		return false
	}
	return true
}

func parsePattern(pattern string) (scheme string, host string, path string) {
	parts := strings.Split(pattern, "://")
	if len(parts) == 2 {
		scheme = parts[0]
		pattern = parts[1]
	} else {
		scheme = ""
	}
	parts = strings.SplitN(pattern, "/", 2)
	if len(parts) == 2 {
		host = parts[0]
		path = "/" + parts[1]
	} else {
		host = pattern
		path = "/"
	}
	return
}

func matchHost(pattern string, host string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		return strings.HasSuffix(host, pattern[1:])
	}
	return pattern == host
}

func matchPath(pattern string, path string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.HasSuffix(pattern, "*") && !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	if strings.Contains(pattern, "*") {
		pattern = strings.Replace(pattern, "*", ".*", -1)
		matched, _ := regexp.MatchString("^"+pattern+"$", path)
		return matched
	}
	return pattern == path
}
