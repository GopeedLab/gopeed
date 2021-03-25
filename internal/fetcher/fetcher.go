package fetcher

import (
	"github.com/monkeyWie/gopeed-core/internal/controller"
	"github.com/monkeyWie/gopeed-core/pkg/base"
)

// 对应协议的下载支持
type Fetcher interface {
	Setup(ctl controller.Controller)
	// 解析请求
	Resolve(req *base.Request) (res *base.Resource, err error)
	// 创建任务
	Create(res *base.Resource, opts *base.Options) (err error)
	Start() (err error)
	Pause() (err error)
	Continue() (err error)

	// 获取任务各个文件下载进度
	Progress() Progress
	// 该方法会一直阻塞，直到任务下载结束
	Wait() (err error)
}

type DefaultFetcher struct {
	Ctl    controller.Controller
	DoneCh chan error
}

func (f *DefaultFetcher) Setup(ctl controller.Controller) {
	f.Ctl = ctl
	f.DoneCh = make(chan error, 1)
}

func (f *DefaultFetcher) Wait() (err error) {
	return <-f.DoneCh
}

// 获取任务中各个文件的已下载字节数
type Progress []int64

// TotalDownloaded 获取任务总下载字节数
func (p Progress) TotalDownloaded() int64 {
	total := int64(0)
	for _, d := range p {
		total += d
	}
	return total
}
