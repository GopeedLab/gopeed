package model

import (
	common2 "github.com/monkeyWie/gopeed/download/common"
)

type Chunk struct {
	Status     common2.Status
	Begin      int64
	End        int64
	Downloaded int64
}

func NewChunk(begin int64, end int64) *Chunk {
	return &Chunk{
		Status: common2.DownloadStatusReady,
		Begin:  begin,
		End:    end,
	}
}
