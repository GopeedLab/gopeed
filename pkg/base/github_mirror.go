package base

import "github.com/GopeedLab/gopeed/internal/githubmirror"

// Preserve downloader configuration type names and the existing JSON schema.
type GithubMirror = githubmirror.Mirror
type GithubMirrorConfig = githubmirror.Config

func DefaultGithubMirrorConfig() *GithubMirrorConfig {
	return githubmirror.DefaultConfig()
}
