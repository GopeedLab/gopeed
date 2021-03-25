package base

// 下载请求
type Request struct {
	// 下载链接
	URL string
	// 附加信息
	Extra interface{}
}

// 资源信息
type Resource struct {
	Req *Request
	// 资源总大小
	TotalSize int64
	// 是否支持断点下载
	Range bool
	// 资源所包含的文件列表
	Files []*FileInfo
}

type FileInfo struct {
	Name string
	Path string
	Size int64
}

// 下载选项
type Options struct {
	// 保存文件名
	Name string
	// 保存目录
	Path string
	// 并发连接数
	Connections int
}
