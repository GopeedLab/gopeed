package model

import "github.com/monkeyWie/gopeed-core/pkg/base"

type CreateTaskReq struct {
	Resource *base.Resource `json:"resource"`
	Options  *base.Options  `json:"options"`
}
