package base

type Status string

const (
	DownloadStatusReady   Status = "ready" // task create but not start
	DownloadStatusRunning Status = "running"
	DownloadStatusPause   Status = "pause"
	DownloadStatusWait    Status = "wait" // task is wait for running
	DownloadStatusError   Status = "error"
	DownloadStatusDone    Status = "done"
)

const (
	HttpCodeOK             = 200
	HttpCodePartialContent = 206

	HttpHeaderHost               = "Host"
	HttpHeaderRange              = "Range"
	HttpHeaderAcceptRanges       = "Accept-Ranges"
	HttpHeaderContentLength      = "Content-Length"
	HttpHeaderContentRange       = "Content-Range"
	HttpHeaderContentDisposition = "Content-Disposition"
	HttpHeaderUserAgent          = "User-Agent"
	HttpHeaderLastModified       = "Last-Modified"

	HttpHeaderBytes       = "bytes"
	HttpHeaderRangeFormat = "bytes=%d-%d"
)
