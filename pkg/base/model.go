package base

import "github.com/monkeyWie/gopeed/pkg/util"

// 下载请求
type Request struct {
	// 下载链接
	URL string `json:"url"`
	// 附加信息
	Extra any `json:"extra"`
}

// 资源信息
type Resource struct {
	Req *Request `json:"req"`
	// 资源名称
	Name string `json:"name"`
	// 资源总大小
	Size int64 `json:"size"`
	// 是否支持断点下载
	Range bool `json:"range"`
	// 资源所包含的文件列表
	Files []*FileInfo `json:"files"`
	// 资源hash值
	Hash string `json:"hash"`
}

type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// 下载选项
type Options struct {
	// 保存文件名
	Name string `json:"name"`
	// 保存目录
	Path string `json:"path"`
	// 选择下载的文件下标列表
	SelectFiles []int `json:"selectFiles"`
	// 附加信息
	Extra any `json:"extra"`
}

func ParseReqExtra[E any](req *Request) error {
	if req.Extra == nil {
		return nil
	}
	if _, ok := req.Extra.(*E); ok {
		return nil
	}
	var t E
	if err := util.MapToStruct(req.Extra, &t); err != nil {
		return err
	}
	req.Extra = &t
	return nil
}

func ParseOptsExtra[E any](opts *Options) error {
	if opts.Extra == nil {
		return nil
	}
	if _, ok := opts.Extra.(*E); ok {
		return nil
	}
	var t E
	if err := util.MapToStruct(opts.Extra, &t); err != nil {
		return err
	}
	opts.Extra = &t
	return nil
}
