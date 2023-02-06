package model

import "github.com/monkeyWie/gopeed/pkg/base"

type CreateTask struct {
	Rid  string        `json:"rid"`
	Req  *base.Request `json:"req"`
	Opts *base.Options `json:"opts"`
}
