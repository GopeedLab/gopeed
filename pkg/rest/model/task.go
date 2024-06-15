package model

import "github.com/GopeedLab/gopeed/pkg/base"

type CreateTask struct {
	Rid string        `json:"rid"`
	Req *base.Request `json:"req"`
	Opt *base.Options `json:"opt"`
}

type CreateTaskBatch struct {
	Reqs []*base.Request `json:"reqs"`
	Opt  *base.Options   `json:"opt"`
}
