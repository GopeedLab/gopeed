package model

import "github.com/monkeyWie/gopeed-core/pkg/base"

type CreateTask struct {
	Res  *base.Resource `json:"res"`
	Opts *base.Options  `json:"opts"`
}
