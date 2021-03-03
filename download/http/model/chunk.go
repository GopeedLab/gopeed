package model

import (
	"github.com/monkeyWie/gopeed-core/download/base"
)

type Chunk struct {
	Status     base.Status
	Begin      int64
	End        int64
	Downloaded int64
}

func NewChunk(begin int64, end int64) *Chunk {
	return &Chunk{
		Status: base.DownloadStatusReady,
		Begin:  begin,
		End:    end,
	}
}
