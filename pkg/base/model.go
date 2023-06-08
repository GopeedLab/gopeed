package base

import (
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/util"
)

// Request download request
type Request struct {
	URL   string `json:"url"`
	Extra any    `json:"extra"`
}

// Resource download resource
type Resource struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	// is support range download
	Range   bool   `json:"range"`
	RootDir string `json:"rootDir"`
	// file list
	Files []*FileInfo `json:"files"`
	Hash  string      `json:"hash"`
}

func (r *Resource) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("invalid name")
	}
	if r.Size < 0 {
		return fmt.Errorf("invalid size")
	}
	if r.Files == nil || len(r.Files) == 0 {
		return fmt.Errorf("invalid files")
	}
	for _, file := range r.Files {
		if file.Name == "" {
			return fmt.Errorf("invalid file name")
		}
	}
	return nil

}

type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`

	Req *Request `json:"req"`
}

// Options for download
type Options struct {
	// Download file name
	Name string `json:"name"`
	// Download file path
	Path string `json:"path"`
	// Select file indexes to download
	SelectFiles []int `json:"selectFiles"`
	// Extra info for specific fetcher
	Extra any `json:"extra"`
}

func ParseReqExtra[E any](req *Request) error {
	if req.Extra == nil {
		return nil
	}
	if _, ok := req.Extra.(*E); ok {
		return nil
	}
	var t E
	if err := util.MapToStruct(req.Extra, &t); err != nil {
		return err
	}
	req.Extra = &t
	return nil
}

func ParseOptsExtra[E any](opts *Options) error {
	if opts.Extra == nil {
		return nil
	}
	if _, ok := opts.Extra.(*E); ok {
		return nil
	}
	var t E
	if err := util.MapToStruct(opts.Extra, &t); err != nil {
		return err
	}
	opts.Extra = &t
	return nil
}
