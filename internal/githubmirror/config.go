// Package githubmirror rewrites GitHub URLs and executes mirrored operations.
package githubmirror

import (
	"fmt"
	"net/url"
	"strings"
)

type Mirror struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	IsBuiltIn bool   `json:"isBuiltIn"`
	IsDeleted bool   `json:"isDeleted"`
}
type Config struct {
	Enabled bool     `json:"enabled"`
	Mirrors []Mirror `json:"mirrors"`
}

func DefaultConfig() *Config {
	return &Config{Enabled: true, Mirrors: []Mirror{
		{Type: "jsdelivr", URL: "https://fastly.jsdelivr.net/gh", IsBuiltIn: true},
		{Type: "ghProxy", URL: "https://fastgit.cc", IsBuiltIn: true},
		{Type: "ghProxy", URL: "https://gh-proxy.com", IsBuiltIn: true},
		{Type: "ghProxy", URL: "https://ghfast.top", IsBuiltIn: true},
	}}
}
func (c *Config) Validate() error {
	if len(c.Mirrors) > 32 {
		return fmt.Errorf("too many GitHub mirrors")
	}
	for _, mirror := range c.Mirrors {
		u, err := url.Parse(mirror.URL)
		if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid GitHub mirror URL")
		}
		if mirror.Type != "jsdelivr" && mirror.Type != "ghProxy" {
			return fmt.Errorf("invalid GitHub mirror type: %s", mirror.Type)
		}
	}
	return nil
}

// Kind describes what a candidate endpoint must support.
type Kind uint8

const (
	Resource Kind = iota
	Repository
)

// Candidates preserves the official URL first. jsDelivr is only eligible for
// raw source resources; Git operations use mirrors supporting Git Smart HTTP.
// GitHub API URLs are never mirrored.
func Candidates(c *Config, raw string, kind Kind) []string {
	result := []string{raw}
	u, err := url.Parse(raw)
	if err != nil || c == nil || !c.Enabled || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return result
	}
	if u.Host != "github.com" && u.Host != "raw.githubusercontent.com" {
		return result
	}
	seen := map[string]bool{raw: true}
	for _, mirror := range c.Mirrors {
		if mirror.IsDeleted {
			continue
		}
		var candidate string
		switch mirror.Type {
		case "ghProxy":
			candidate = strings.TrimRight(mirror.URL, "/") + "/" + raw
		case "jsdelivr":
			if kind == Repository || u.Host != "raw.githubusercontent.com" {
				continue
			}
			parts := strings.SplitN(strings.TrimPrefix(u.EscapedPath(), "/"), "/", 4)
			if len(parts) != 4 {
				continue
			}
			candidate = strings.TrimRight(mirror.URL, "/") + "/" + parts[0] + "/" + parts[1] + "@" + parts[2] + "/" + parts[3]
		}
		if candidate != "" && !seen[candidate] {
			result = append(result, candidate)
			seen[candidate] = true
		}
	}
	return result
}
