package http

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/download/common"
	"github.com/monkeyWie/gopeed-core/download/http/model"
	"io"
	"net/http"
	"path/filepath"
	"sync"
)

type Process struct {
	fetcher *Fetcher

	res     *common.Resource
	opts    *common.Options
	status  common.Status
	clients []*http.Response
	chunks  []*model.Chunk

	pauseLock *sync.Mutex
}

func NewProcess(fetcher *Fetcher, res *common.Resource, opts *common.Options) *Process {
	return &Process{
		fetcher: fetcher,
		res:     res,
		opts:    opts,
		status:  common.DownloadStatusReady,

		pauseLock: &sync.Mutex{},
	}
}

func (p *Process) Start() error {
	ctl := p.fetcher.GetCtl()
	// 创建文件
	name := p.name()
	_, err := ctl.Touch(name, p.res.Size)
	if err != nil {
		return err
	}
	defer p.close()
	p.status = common.DownloadStatusStart
	var fetchErr error
	if p.res.Range {
		// 每个连接平均需要下载的分块大小
		var (
			chunkSize = p.res.Size / int64(p.opts.Connections)
			errCh     = make(chan error)
		)
		defer close(errCh)
		p.chunks = make([]*model.Chunk, p.opts.Connections)
		p.clients = make([]*http.Response, p.opts.Connections)
		for i := 0; i < p.opts.Connections; i++ {
			var (
				begin = chunkSize * int64(i)
				end   int64
			)
			if i == p.opts.Connections-1 {
				// 最后一个分块需要保证把文件下载完
				end = p.res.Size
			} else {
				end = begin + chunkSize
			}
			chunk := model.NewChunk(begin, end)
			p.chunks[i] = chunk
			go func(i int) {
				errCh <- p.fetchChunk(i, name, chunk)
			}(i)
		}
		fetchErr = p.wait(errCh)
	} else {
		// 只支持单连接下载
		p.chunks = make([]*model.Chunk, 1)
		p.clients = make([]*http.Response, 1)
		p.chunks[0] = model.NewChunk(0, 0)
		fetchErr = p.fetchChunk(0, name, p.chunks[0])
	}
	if p.status == common.DownloadStatusPause {
		return nil
	}
	return fetchErr
}

func (p *Process) Pause() error {
	p.pauseLock.Lock()
	defer p.pauseLock.Unlock()
	if common.DownloadStatusStart != p.status {
		return nil
	}
	p.status = common.DownloadStatusPause
	if len(p.clients) > 0 {
		for _, client := range p.clients {
			client.Body.Close()
		}
	}
	return nil
}

func (p *Process) Continue() error {
	if func() bool {
		p.pauseLock.Lock()
		defer p.pauseLock.Unlock()
		if common.DownloadStatusPause != p.status && common.DownloadStatusError != p.status {
			return true
		}
		p.status = common.DownloadStatusStart
		return false
	}() {
		return nil
	}

	var (
		ctl   = p.fetcher.GetCtl()
		name  = p.name()
		errCh = make(chan error)
	)
	_, err := ctl.Open(name)
	if err != nil {
		return err
	}
	defer ctl.Close(name)
	for i := 0; i < p.opts.Connections; i++ {
		go func(i int) {
			errCh <- p.fetchChunk(i, name, p.chunks[i])
		}(i)
	}
	return p.wait(errCh)
}

func (p *Process) Delete() error {
	panic("implement me")
}

func (p *Process) close() error {
	return p.fetcher.GetCtl().Close(p.name())
}

func (p *Process) name() string {
	// 创建文件
	var filename = p.opts.Name
	if filename == "" {
		filename = p.res.Files[0].Name
	}
	return filepath.Join(p.opts.Path, filename)
}

func (p *Process) wait(errCh <-chan error) error {
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
	return nil
}

type action int

const (
	actionDirect action = iota
	actionContinue
	actionReturn
)

func (p *Process) fetchChunk(index int, name string, chunk *model.Chunk) (err error) {
	httpReq, err := BuildRequest(p.res.Req)
	if err != nil {
		return err
	}
	var (
		client = BuildClient()
		buf    = make([]byte, 8192)
	)
	// 重试5次
	for i := 0; i < 5; i++ {
		var (
			resp  *http.Response
			retry bool
		)
		if p.res.Range {
			httpReq.Header.Set(common.HttpHeaderRange,
				fmt.Sprintf(common.HttpHeaderRangeFormat, chunk.Begin+chunk.Downloaded, chunk.End))
		} else {
			chunk.Downloaded = 0
		}
		var act action
		act, err = func() (action, error) {
			p.pauseLock.Lock()
			defer p.pauseLock.Unlock()
			if p.status == common.DownloadStatusPause {
				return actionReturn, nil
			}
			resp, err = client.Do(httpReq)
			if err != nil {
				// 连接失败，直接重试
				return actionContinue, err
			}
			if resp.StatusCode != common.HttpCodeOK && resp.StatusCode != common.HttpCodePartialContent {
				err = NewRequestError(resp.StatusCode, resp.Status)
				return actionContinue, err
			}
			p.clients[index] = resp
			return actionDirect, nil
		}()
		fmt.Printf("action:%d\n", act)
		if act == actionContinue {
			continue
		} else if act == actionReturn {
			return nil
		}
		retry, err = func() (bool, error) {
			defer resp.Body.Close()
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					_, err := p.fetcher.GetCtl().Write(name, chunk.Begin+chunk.Downloaded, buf[:n])
					if err != nil {
						return false, err
					}
					chunk.Downloaded += int64(n)
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
