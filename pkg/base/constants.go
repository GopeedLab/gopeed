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

	HttpHeaderRange              = "Range"
	HttpHeaderContentLength      = "Content-Length"
	HttpHeaderContentRange       = "Content-Range"
	HttpHeaderContentDisposition = "Content-Disposition"
	HttpHeaderUserAgent          = "User-Agent"

	HttpHeaderRangeFormat = "bytes=%d-%d"
)

const (
	AgentName = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
)
