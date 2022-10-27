package model

import "github.com/monkeyWie/gopeed/pkg/base"

type CreateTask struct {
	Res  *base.Resource `json:"res"`
	Opts *base.Options  `json:"opts"`
}
