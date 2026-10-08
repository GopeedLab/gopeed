package ftp

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"

	"github.com/GopeedLab/gopeed/pkg/base"
)

// Limits of a directory download. A tree past either one is refused with an
// error that names the limit; it is never cut short.
const maxWalkDepth = 8

// maxWalkFiles is a variable so tests can lower it.
var maxWalkFiles = 10000

// resolveTarget describes the URL's path: one file, or a directory walked
// into a multi-file resource whose paths are relative to it.
func (f *Fetcher) resolveTarget(s *session, t *target) (*base.Resource, error) {
	isDir, size, mtime, err := f.stat(s, t.path)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return &base.Resource{
			Size:  size,
			Range: true,
			Files: []*base.FileInfo{{
				Name:  path.Base(t.path),
				Size:  size,
				Ctime: mtime,
			}},
		}, nil
	}

	files, err := f.walk(s, t.path)
	if err != nil {
		return nil, err
	}
	name := path.Base(strings.TrimSuffix(t.path, "/"))
	if name == "" || name == "." || name == "/" {
		name = t.host
	}
	res := &base.Resource{Name: name, Range: true, Files: files}
	res.CalcSize(nil)
	return res, nil
}

// stat tells a file from a directory. MLST answers both at once. Without
// it, SIZE answers for a file, and a directory is one we can change into.
func (f *Fetcher) stat(s *session, p string) (isDir bool, size int64, mtime *time.Time, err error) {
	if p == "" || strings.HasSuffix(p, "/") {
		return true, 0, nil, nil
	}
	s.deadline()
	if entry, mlstErr := s.GetEntry(p); mlstErr == nil {
		switch entry.Type {
		case ftp.EntryTypeFolder:
			return true, 0, nil, nil
		case ftp.EntryTypeFile:
			if !entry.Time.IsZero() {
				t := entry.Time
				mtime = &t
			}
			return false, int64(entry.Size), mtime, nil
		}
		// A link: SIZE below follows it on the server.
	}
	// MLST is missing or refused the path: the older commands decide. A
	// broken connection fails them too, with its own error.

	s.deadline()
	size, sizeErr := s.FileSize(p)
	if sizeErr == nil {
		if s.IsGetTimeSupported() {
			s.deadline()
			if t, err := s.GetTime(p); err == nil {
				mtime = &t
			}
		}
		return false, size, mtime, nil
	}
	if replyCode(sizeErr) == 0 {
		return false, 0, nil, sizeErr
	}

	// SIZE failed: a directory, or a missing path.
	s.deadline()
	cwd, err := s.CurrentDir()
	if err != nil {
		return false, 0, nil, err
	}
	s.deadline()
	if err := s.ChangeDir(p); err != nil {
		return false, 0, nil, sizeErr
	}
	s.deadline()
	if err := s.ChangeDir(cwd); err != nil {
		return false, 0, nil, err
	}
	return true, 0, nil, nil
}

// safeEntryName reports whether a name from a listing can become a local
// path element. A hostile server could otherwise list "../x" and write
// outside the task directory.
func safeEntryName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\\x00")
}

// walk lists a directory tree, at most maxWalkDepth levels below root and
// maxWalkFiles files. Files come back sorted by path, then name.
func (f *Fetcher) walk(s *session, root string) ([]*base.FileInfo, error) {
	type dir struct {
		remote string // path on the server
		rel    string // path below root, "" for root itself
		depth  int
	}
	var files []*base.FileInfo
	stack := []dir{{remote: strings.TrimSuffix(root, "/"), rel: "", depth: 0}}
	for len(stack) > 0 {
		d := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		s.deadline()
		entries, err := s.List(d.remote)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.Name == "." || e.Name == ".." {
				continue
			}
			if !safeEntryName(e.Name) {
				f.warn("ftp: skipping a listed name that is not a safe file name: %q", e.Name)
				continue
			}
			remote := path.Join(d.remote, e.Name)
			rel := path.Join(d.rel, e.Name)
			if d.remote == "" {
				remote = e.Name
			}
			entryType, size := e.Type, int64(e.Size)
			if entryType == ftp.EntryTypeLink {
				// A link to a file has a size; a link to a directory is
				// not followed, so a loop cannot make the walk endless.
				s.deadline()
				linkSize, err := s.FileSize(remote)
				if err != nil {
					f.warn("ftp: skipping link %q: %v", rel, err)
					continue
				}
				entryType, size = ftp.EntryTypeFile, linkSize
			}
			switch entryType {
			case ftp.EntryTypeFolder:
				if d.depth+1 > maxWalkDepth {
					return nil, fmt.Errorf("ftp: the folder is deeper than %d levels, the limit for a folder download", maxWalkDepth)
				}
				stack = append(stack, dir{remote: remote, rel: rel, depth: d.depth + 1})
			case ftp.EntryTypeFile:
				if len(files) >= maxWalkFiles {
					return nil, fmt.Errorf("ftp: the folder has more than %d files, the limit for a folder download", maxWalkFiles)
				}
				fi := &base.FileInfo{Name: e.Name, Path: d.rel, Size: size}
				if !e.Time.IsZero() {
					t := e.Time
					fi.Ctime = &t
				}
				files = append(files, fi)
			}
		}
	}
	slices.SortFunc(files, func(a, b *base.FileInfo) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.Name, b.Name))
	})
	return files, nil
}
