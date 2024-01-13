package base

import (
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/util"
	"golang.org/x/exp/slices"
)

// Request download request
type Request struct {
	URL   string `json:"url"`
	Extra any    `json:"extra"`
	// Labels is used to mark the download task
	Labels map[string]string `json:"labels"`
}

func (r *Request) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("invalid request url")
	}
	return nil
}

// Resource download resource
type Resource struct {
	// if name is not empty, the resource is a folder and the name is the folder name
	Name string `json:"name"`
	Size int64  `json:"size"`
	// is support range download
	Range bool `json:"range"`
	// file list
	Files []*FileInfo `json:"files"`
	Hash  string      `json:"hash"`
	// health indicators of torrents, from large to small, ConnectedSeeders are also the key to the health of seed resources
	TotalPeers       int `json:"totalPeers"`
	ActivePeers      int `json:"activePeers"`
	ConnectedSeeders int `json:"connectedSeeders"`
}

func (r *Resource) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("invalid resource name")
	}
	if r.Files == nil || len(r.Files) == 0 {
		return fmt.Errorf("invalid resource files")
	}
	for _, file := range r.Files {
		if file.Name == "" {
			return fmt.Errorf("invalid resource file name")
		}
	}
	return nil
}

func (r *Resource) CalcSize(selectFiles []int) {
	var size int64
	for i, file := range r.Files {
		if len(selectFiles) == 0 || slices.Contains(selectFiles, i) {
			size += file.Size
		}
	}
	r.Size = size
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

func (o *Options) InitSelectFiles(fileSize int) {
	// if selectFiles is empty, select all files
	if len(o.SelectFiles) == 0 {
		o.SelectFiles = make([]int, fileSize)
		for i := 0; i < fileSize; i++ {
			o.SelectFiles[i] = i
		}
	}
}

func (o *Options) Clone() *Options {
	return &Options{
		Name:        o.Name,
		Path:        o.Path,
		SelectFiles: o.SelectFiles,
		Extra:       o.Extra,
	}
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
