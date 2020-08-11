package model

import (
	"github.com/monkeyWie/gopeed-core/download/common"
)

type Chunk struct {
	Status     common.Status
	Begin      int64
	End        int64
	Downloaded int64
}

func NewChunk(begin int64, end int64) *Chunk {
	return &Chunk{
		Status: common.DownloadStatusReady,
		Begin:  begin,
		End:    end,
	}
}
