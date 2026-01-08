package model

import "github.com/GopeedLab/gopeed/pkg/base"

type ResolveTask struct {
	Req  *base.Request `json:"req"`
	Opts *base.Options `json:"opts"`
}
type CreateTask struct {
	Rid string `json:"rid"`

	Req  *base.Request `json:"req"`
	Opts *base.Options `json:"opts"`
}
