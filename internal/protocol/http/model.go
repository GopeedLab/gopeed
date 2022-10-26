package http

import "github.com/monkeyWie/gopeed-core/pkg/base"

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

type extra struct {
	Method string
	Header map[string]string
	Body   string
}
