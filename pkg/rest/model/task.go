package model

import "github.com/GopeedLab/gopeed/pkg/base"

type ResolveTask struct {
	Req *base.Request `json:"req"`
	Opt *base.Options `json:"opt"`
}
type CreateTask struct {
	Rid string `json:"rid"`

	Req *base.Request `json:"req"`
	Opt *base.Options `json:"opt"`
}
