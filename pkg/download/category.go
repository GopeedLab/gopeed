package download

import (
	"errors"
	"path"
	"strings"
	"unicode"

	"github.com/GopeedLab/gopeed/pkg/base"
)

// categoryForResource returns the category that receives the resolved
// resource, or nil when auto categorize does not apply. Only a single file
// resource is routed: a folder or a multi file resource keeps its directory
// layout, so its files are never matched one by one, not even when only a
// single one of them is selected.
func categoryForResource(cfg *base.DownloaderStoreConfig, res *base.Resource) *base.DownloadCategory {
	if !cfg.AutoCategorizeEnabled() || len(cfg.Categories) == 0 {
		return nil
	}
	if res == nil || res.Name != "" || len(res.Files) != 1 {
		return nil
	}
	return matchDownloadCategory(cfg, res.Files[0].Name)
}

// decideDownloadDir returns the download directory of a task. The base
// directory is the one the task asked for, or the configured download
// directory when the task left it empty. A task that did not skip auto
// categorize is routed to the category directory of its resolved resource
// when one matches. Path placeholders are expanded and the white list check
// runs on the result, so a category directory the white list rejects falls
// back to the base directory of this task instead of failing it.
func (d *Downloader) decideDownloadDir(opts *base.Options, res *base.Resource) (string, error) {
	if opts == nil {
		opts = &base.Options{}
	}
	cfg, err := d.GetConfig()
	if err != nil {
		return "", err
	}
	base := dartTrimSpace(opts.Path)
	if base == "" {
		base = cfg.DownloadDir
	}
	// Refuse to fall through to the process working directory when neither the
	// task nor the config names a directory.
	if base == "" {
		return "", errors.New("download directory is empty")
	}
	target := base
	if !opts.SkipAutoCategorize {
		if category := categoryForResource(cfg, res); category != nil {
			target = dartTrimSpace(category.Path)
		}
	}
	dir, err := d.initDownloadPath(target)
	if err != nil && target != base {
		d.Logger.Warn().Err(err).Msgf("category directory %q rejected by white list, falling back to %q", target, base)
		dir, err = d.initDownloadPath(base)
	}
	if err != nil {
		return "", err
	}
	return dir, nil
}

// suggestedDownloadDir returns the category directory a task would be routed
// to, or an empty string when auto categorize does not apply. It only reports
// the decision of decideDownloadDir, it expands no directory and writes
// nothing.
func (d *Downloader) suggestedDownloadDir(res *base.Resource) string {
	cfg, err := d.GetConfig()
	if err != nil {
		return ""
	}
	category := categoryForResource(cfg, res)
	if category == nil {
		return ""
	}
	dir, err := d.initDownloadPath(dartTrimSpace(category.Path))
	if err != nil {
		return ""
	}
	return dir
}

// asciiLower lowercases the ascii letters of s and leaves the rest alone. The
// ui lowercases the same way, because the unicode case tables of the two
// languages disagree for some code points and would let the two sides pick a
// different category for the same file.
func asciiLower(s string) string {
	hasUpper := false
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 'A' && c <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return s
	}
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// matchDownloadCategory returns the first category whose extension list
// contains the extension of the given file name, or nil when nothing matches.
// An empty extension list never matches.
func matchDownloadCategory(cfg *base.DownloaderStoreConfig, name string) *base.DownloadCategory {
	ext := fileExtension(name)
	if ext == "" {
		return nil
	}
	for _, category := range cfg.Categories {
		if category == nil || category.IsDeleted || dartTrimSpace(category.Path) == "" {
			continue
		}
		for _, e := range normalizeCategoryExtensions(category.Extensions) {
			if e == ext {
				return category
			}
		}
	}
	return nil
}

// normalizeCategoryExtensions mirrors the dart side: entries are split on
// commas and whitespace, trimmed, lowercased, stripped of leading dots and
// deduplicated while keeping the input order, so a stored value like
// "exe, msi" behaves the same in the backend as it does in the ui.
func normalizeCategoryExtensions(raw []string) []string {
	var result []string
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		for _, part := range strings.FieldsFunc(item, func(r rune) bool {
			return r == ',' || isECMAScriptSpace(r)
		}) {
			ext := asciiLower(strings.TrimLeft(dartTrimSpace(part), "."))
			if ext == "" {
				continue
			}
			if _, ok := seen[ext]; ok {
				continue
			}
			seen[ext] = struct{}{}
			result = append(result, ext)
		}
	}
	return result
}

// fileExtension returns the lower case extension of a file name without the
// leading dot, e.g. "setup.exe" -> "exe". Windows style separators are
// treated like the ui does, so a dot inside a directory segment never counts
// as an extension.
func fileExtension(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	return strings.TrimPrefix(asciiLower(path.Ext(name)), ".")
}

// dartTrimSpace trims the whitespace the dart String.trim() removes, which is
// the unicode white space set plus the byte order mark. The ui trims the
// category paths and the extension lists with it, so the backend uses the same
// set instead of the narrower unicode.IsSpace.
func dartTrimSpace(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == 0xFEFF
	})
}

// isECMAScriptSpace reports whether r belongs to the whitespace class of the
// ECMAScript regular expressions. The ui splits its extension lists with that
// class, so the backend uses the same one instead of the wider unicode.IsSpace.
func isECMAScriptSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}
