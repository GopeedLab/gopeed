package base

import "errors"

type Status int

const (
	DownloadStatusReady Status = iota
	DownloadStatusStart
	DownloadStatusPause
	DownloadStatusError
	DownloadStatusDone
)

const (
	HttpCodeOK             = 200
	HttpCodePartialContent = 206

	HttpHeaderRange              = "Range"
	HttpHeaderContentLength      = "Content-Length"
	HttpHeaderContentRange       = "Content-Range"
	HttpHeaderContentDisposition = "Content-Disposition"

	HttpHeaderRangeFormat = "bytes=%d-%d"
)

var (
	DeleteErr = errors.New("delete")
)
