package util

import (
	"net/url"
	"regexp"
	"strings"
)

// Match url with pattern by chrome extension match pattern style.
// Magnet URI patterns such as magnet:?* are also supported, with * as a wildcard.
// https://developer.chrome.com/docs/extensions/mv3/match_patterns/
func Match(pattern string, u string) bool {
	url, err := url.Parse(u)
	if err != nil {
		return false
	}
	// Magnet URIs have no authority or path; match the raw scheme-specific
	// part instead, preserving query punctuation and percent-encoding.
	if strings.HasPrefix(pattern, "magnet:") && !strings.HasPrefix(pattern, "magnet://") {
		if url.Scheme != "magnet" {
			return false
		}
		_, target, _ := strings.Cut(u, ":")
		expression := regexp.QuoteMeta(strings.TrimPrefix(pattern, "magnet:"))
		expression = strings.ReplaceAll(expression, `\*`, ".*")
		matched, _ := regexp.MatchString("^"+expression+"$", target)
		return matched
	}
	scheme, host, path := parsePattern(pattern)
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
