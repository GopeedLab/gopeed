package githubmirror

import (
	"reflect"
	"testing"
)

func TestMirrorCandidates(t *testing.T) {
	cfg := &Config{Enabled: true, Mirrors: []Mirror{
		{Type: "jsdelivr", URL: "https://cdn.example/gh/"},
		{Type: "ghProxy", URL: "https://mirror.example/"},
		{Type: "ghProxy", URL: "https://deleted.example", IsDeleted: true},
	}}
	source := "https://raw.githubusercontent.com/owner/repo/main/a.txt"
	want := []string{source, "https://cdn.example/gh/owner/repo@main/a.txt", "https://mirror.example/" + source}
	if got := Candidates(cfg, source, Resource); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	repo := "https://github.com/owner/repo.git"
	if got := Candidates(cfg, repo, Repository); !reflect.DeepEqual(got, []string{repo, "https://mirror.example/" + repo}) {
		t.Fatalf("git candidates %v", got)
	}
	for _, raw := range []string{"https://github.com.evil.example/a", "https://example.com/a", "https://user:secret@github.com/a"} {
		if got := Candidates(cfg, raw, Resource); !reflect.DeepEqual(got, []string{raw}) {
			t.Fatalf("unexpected mirror: %v", got)
		}
	}
	apiURL := "https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=10"
	for _, kind := range []Kind{Resource, Repository} {
		if got := Candidates(cfg, apiURL, kind); !reflect.DeepEqual(got, []string{apiURL}) {
			t.Fatalf("GitHub API must never use mirrors: %v", got)
		}
	}
	cfg.Enabled = false
	if got := Candidates(cfg, source, Resource); !reflect.DeepEqual(got, []string{source}) {
		t.Fatalf("disabled mirror: %v", got)
	}
}
