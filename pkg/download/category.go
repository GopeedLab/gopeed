package download

import (
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
)

// routeDownloadPath returns the download directory for a task. When the task
// targets the global default download directory and the task file name matches
// a configured category by extension, the category directory replaces the
// current directory. The returned path may still contain placeholders, they
// are expanded later by initDownloadPath together with the white list check.
func (d *Downloader) routeDownloadPath(req *base.Request, opts *base.Options, cfg *base.DownloaderStoreConfig) string {
	current := strings.TrimSpace(opts.Path)
	if current == "" {
		current = cfg.DownloadDir
	}
	if current == "" || !cfg.AutoCategorize || len(cfg.Categories) == 0 {
		return current
	}
	// BT tasks bind their save path when the torrent is added during resolve,
	// re-routing the path afterwards would desynchronize the meta info from the
	// actual on-disk location. The guard only trips for urls the bt fetcher
	// claims first (magnet links, torrent payloads); a .torrent file over https
	// is claimed by the http fetcher instead and routes by name below.
	if req == nil || req.URL == "" {
		return current
	}
	if fm, err := d.parseFm(req.URL); err == nil && fm.Name() == "bt" {
		return current
	}
	name := categoryFileName(req, opts)
	if name == "" {
		return current
	}
	// Only tasks that keep the default download directory are routed, an
	// explicitly chosen directory always wins.
	if comparablePath(current) != comparablePath(cfg.DownloadDir) {
		return current
	}
	category := matchDownloadCategory(cfg, name)
	if category == nil {
		return current
	}
	return category.Path
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
		if category == nil || category.IsDeleted || strings.TrimSpace(category.Path) == "" {
			continue
		}
		for _, e := range normalizeCategoryExtensions(category.Extensions) {
			if strings.EqualFold(e, ext) {
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
			return r == ',' || unicode.IsSpace(r)
		}) {
			ext := strings.ToLower(strings.TrimLeft(strings.TrimSpace(part), "."))
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

// categoryFileName returns the best available file name of a task for category
// matching, which is the first candidate that carries a file extension. The
// candidate order must stay identical between resolve and create so protocols
// that bind the path early, e.g. ed2k and hls, never end up with different
// directories on both sides.
func categoryFileName(req *base.Request, opts *base.Options) string {
	if opts != nil && strings.TrimSpace(opts.Name) != "" {
		if name := strings.TrimSpace(opts.Name); fileExtension(name) != "" {
			return name
		}
	}
	if req == nil {
		return ""
	}
	name := fileNameFromURL(req.URL)
	if fileExtension(name) != "" {
		return name
	}
	return ""
}

// fileNameFromURL derives a file name from the task url, ignoring the query
// and fragment part.
func fileNameFromURL(rawURL string) string {
	cleaned := rawURL
	if i := strings.IndexAny(cleaned, "?#"); i >= 0 {
		cleaned = cleaned[:i]
	}
	// ed2k urls use literal pipes as separators, which are not valid url
	// characters, so handle them before parsing.
	if strings.HasPrefix(strings.ToLower(cleaned), "ed2k:") {
		if parts := strings.Split(cleaned, "|"); len(parts) >= 4 && strings.EqualFold(parts[1], "file") {
			return strings.TrimSpace(parts[2])
		}
		return ""
	}
	if u, err := url.Parse(cleaned); err == nil {
		switch u.Scheme {
		case "data", "blob", "magnet":
			return ""
		}
		if u.Path != "" {
			return lastPathSegment(u.Path)
		}
		if u.Scheme != "" && u.Host == "" && u.Opaque == "" {
			// A bare scheme like "magnet:" without any path carries no name.
			return ""
		}
	}
	return lastPathSegment(cleaned)
}

// lastPathSegment returns the last segment of a path that may use either
// separator, which keeps windows style urls working.
func lastPathSegment(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		p = p[i+1:]
	}
	return strings.TrimSpace(p)
}

// fileExtension returns the lower case extension of a file name without the
// leading dot, e.g. "setup.exe" -> "exe".
func fileExtension(name string) string {
	return strings.TrimPrefix(strings.ToLower(path.Ext(name)), ".")
}

// comparablePath normalizes a path for equality comparison: placeholders are
// expanded and separators unified so the same directory written in different
// but equivalent ways compares equal.
func comparablePath(p string) string {
	p = path.Clean(filepath.ToSlash(strings.TrimSpace(util.ReplacePathPlaceholders(p))))
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}
