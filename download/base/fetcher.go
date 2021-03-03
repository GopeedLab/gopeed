package base

// 对应协议的下载支持
type Fetcher interface {
	Init(ctl Controller)
	// 解析请求
	Resolve(req *Request) (*Resource, error)
	// 创建任务
	Create(res *Resource, opts *Options) error
	Start() error
	Pause() error
	Continue() error
	Delete() error
}

type DefaultFetcher struct {
	Ctl Controller
}

func (f *DefaultFetcher) Init(ctl Controller) {
	f.Ctl = ctl
}
