package http

import (
	"fmt"
	"github.com/monkeyWie/gopeed/download/common"
	"github.com/monkeyWie/gopeed/download/http/model"
	"io"
	"net/http"
	"path/filepath"
)

type Process struct {
	fetcher *Fetcher

	res     *common.Resource
	opts    *common.Options
	status  common.Status
	clients []*http.Response
	chunks  []*model.Chunk
}

func NewProcess(fetcher *Fetcher, res *common.Resource, opts *common.Options) *Process {
	return &Process{
		fetcher: fetcher,
		res:     res,
		opts:    opts,
		status:  common.DownloadStatusReady,
	}
}

func (p *Process) Start() error {
	ctl := p.fetcher.GetCtl()
	// 创建文件
	var filename = p.opts.Name
	if filename == "" {
		filename = p.res.Files[0].Name
	}
	name := filepath.Join(p.opts.Path, filename)
	err := ctl.Touch(name, p.res.Size)
	if err != nil {
		return err
	}
	if p.res.Range && p.res.Size > 0 {
		// 每个连接平均需要下载的分块大小
		var (
			chunkSize = p.res.Size / int64(p.opts.Connections)
			errCh     = make(chan error)
		)
		defer close(errCh)
		p.chunks = make([]*model.Chunk, p.opts.Connections)
		p.clients = make([]*http.Response, p.opts.Connections)
		for i := 0; i < p.opts.Connections; i++ {
			begin := chunkSize * int64(i)
			end := begin + chunkSize
			if i == p.opts.Connections-1 {
				// 最后一个连接下载的分块大小需要特殊计算，保证把文件下载完
				chunkSize = end + p.res.Size%int64(p.opts.Connections)
			}
			p.chunks[i] = model.NewChunk(begin, end)
			go func(i int) {
				errCh <- p.doFetch(i, name, begin, end)
			}(i)
		}
		var failErr error
		for i := 0; i < p.opts.Connections; i++ {
			err := <-errCh
			if failErr == nil {
				// 有一个连接下载失败，立即停止下载
				failErr = err
				if failErr != nil {
					p.Pause()
				}
			}
		}
		if failErr != nil {
			return failErr
		}
	}
	return nil
}

func (p *Process) Pause() error {
	if len(p.clients) > 0 {
		for _, client := range p.clients {
			client.Body.Close()
		}
	}
	return nil
}

func (p *Process) Continue() error {
	return nil
}

func (p *Process) Delete() error {
	panic("implement me")
}

func (p *Process) doFetch(index int, filename string, begin, end int64) (err error) {
	httpReq, err := BuildRequest(p.res.Req)
	if err != nil {
		return err
	}
	httpReq.Header.Set(common.HttpHeaderRange, fmt.Sprintf(common.HttpHeaderRangeFormat, begin, end))
	client := BuildClient()

	// 重试5次
	for i := 0; i < 5; i++ {
		var (
			resp  *http.Response
			retry bool
		)
		resp, err = client.Do(httpReq)
		if err != nil {
			// 连接失败，直接重试
			continue
		}
		p.chunks[index].Downloaded = 0
		p.clients[index] = resp
		retry, err = func() (bool, error) {
			defer resp.Body.Close()
			var (
				buf    = make([]byte, 8192)
				offset = begin
			)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					len, err := p.fetcher.GetCtl().Write(filename, offset, buf[:n])
					if err != nil {
						return false, err
					}
					offset += int64(len)
					p.chunks[index].Downloaded = p.chunks[i].Downloaded + int64(len)
				}
				if err != nil {
					if err != io.EOF {
						return true, err
					}
					break
				}
			}
			return false, nil
		}()
		if err == nil || !retry {
			// 下载成功，跳出重试
			break
		}
	}
	if err != nil {
		p.chunks[index].Status = common.DownloadStatusError
	} else {
		p.chunks[index].Status = common.DownloadStatusDone
	}
	return
}
