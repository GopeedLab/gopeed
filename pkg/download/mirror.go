package download

import (
	"errors"

	"github.com/GopeedLab/gopeed/pkg/base"
)

var (
	ErrMirrorProviderNotFound = errors.New("mirror provider not found")
	ErrInvalidMirrorURL       = errors.New("invalid mirror resource URL")
)

// Each provider owns URL validation and candidate generation using its own
// configuration. Providers only generate URLs; they must not fetch resources.
type mirrorCandidateProvider func(config *base.DownloaderStoreConfig, raw string) ([]string, error)

// Add providers here without changing the public API or its client transport.
var mirrorCandidateProviders = map[string]mirrorCandidateProvider{
	"github": func(config *base.DownloaderStoreConfig, raw string) ([]string, error) {
		return githubResourceCandidates(config.GithubMirror, raw)
	},
}

func (d *Downloader) GetMirrorCandidates(provider, raw string) ([]string, error) {
	candidates, ok := mirrorCandidateProviders[provider]
	if !ok {
		return nil, ErrMirrorProviderNotFound
	}
	config, err := d.GetConfig()
	if err != nil {
		return nil, err
	}
	return candidates(config, raw)
}
