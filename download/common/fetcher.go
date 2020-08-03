package common

// 对应协议的下载支持
type Fetcher interface {
	BaseFetch
	// 支持的协议列表
	Protocols() []string
	// 解析请求
	Resolve(req *Request) (*Resource, error)
	// 创建任务
	Create(res *Resource, opts *Options) (Process, error)
}

type BaseFetch interface {
	InitCtl(ctl *Controller)
	GetCtl() *Controller
}

type BaseFetcher struct {
	ctl *Controller
}

func (b *BaseFetcher) InitCtl(ctl *Controller) {
	b.ctl = ctl
}

func (b *BaseFetcher) GetCtl() *Controller {
	return b.ctl
}
