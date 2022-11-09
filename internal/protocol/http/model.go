package http

import (
	"github.com/monkeyWie/gopeed/pkg/base"
)

type chunk struct {
	Status     base.Status
	Begin      int64
	End        int64
	Downloaded int64
}

func newChunk(begin int64, end int64) *chunk {
	return &chunk{
		Status: base.DownloadStatusReady,
		Begin:  begin,
		End:    end,
	}
}

type ReqExtra struct {
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type OptsExtra struct {
	Connections int `json:"connections"`
}

type config struct {
	Connections int `json:"connections"`
}
