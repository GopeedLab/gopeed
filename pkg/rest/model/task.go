package model

import "github.com/GopeedLab/gopeed/pkg/base"

type CreateTask struct {
	Rid  string                `json:"rid"`
	Req  *base.ResolvedRequest `json:"req"`
	Opts *base.Options         `json:"opts"`
}

type CreateTaskBatch struct {
	Reqs []*base.ResolvedRequest `json:"reqs"`
	Opts *base.Options           `json:"opts"`
}
