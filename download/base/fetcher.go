package base

// 对应协议的下载支持
type Fetcher interface {
	Init(ctl Controller)
	// 解析请求
	Resolve(req *Request) (res *Resource, err error)
	// 创建任务
	Create(res *Resource, opts *Options) (err error)
	Start() (err error)
	Pause() (err error)
	Continue() (err error)
}

type DefaultFetcher struct {
	Ctl Controller
}

func (f *DefaultFetcher) Init(ctl Controller) {
	f.Ctl = ctl
}
