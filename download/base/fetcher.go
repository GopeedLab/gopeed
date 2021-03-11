package base

// 对应协议的下载支持
type Fetcher interface {
	Setup(ctl Controller) (err <-chan error)
	// 解析请求
	Resolve(req *Request) (res *Resource, err error)
	// 创建任务
	Create(res *Resource, opts *Options) (err error)
	Start() (err error)
	Pause() (err error)
	Continue() (err error)
}

type DefaultFetcher struct {
	Ctl    Controller
	DoneCh chan error
}

func (f *DefaultFetcher) Setup(ctl Controller) (doneCh <-chan error) {
	f.Ctl = ctl
	f.DoneCh = make(chan error, 1)
	return f.DoneCh
}
