package stream

import (
	_ "embed"
	"fmt"

	"github.com/dop251/goja"
)

//go:embed stream.js
var script string

type Config struct {
	CreateBlobObjectURL           func(data []byte, contentType string) (string, error)
	CreateWritableStreamObjectURL func() (string, error)
	WriteWritableStreamObjectURL  func(url string, data any) error
	CloseWritableStreamObjectURL  func(url string) error
	AbortWritableStreamObjectURL  func(url string, reason string) error
	RevokeObjectURL               func(url string) error
}

func Enable(runtime *goja.Runtime, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	if err := runtime.Set("__gopeed_create_blob_object_url", func(data []byte, contentType string) string {
		if cfg.CreateBlobObjectURL == nil {
			panic(fmt.Errorf("gblob blob object url handler not configured"))
		}
		url, err := cfg.CreateBlobObjectURL(data, contentType)
		if err != nil {
			panic(err)
		}
		return url
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_create_writable_stream_object_url", func() string {
		if cfg.CreateWritableStreamObjectURL == nil {
			panic(fmt.Errorf("gblob writable stream object url handler not configured"))
		}
		url, err := cfg.CreateWritableStreamObjectURL()
		if err != nil {
			panic(err)
		}
		return url
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_write_writable_stream_object_url", func(url string, data any) {
		if cfg.WriteWritableStreamObjectURL == nil {
			panic(fmt.Errorf("gblob writable stream write handler not configured"))
		}
		if err := cfg.WriteWritableStreamObjectURL(url, data); err != nil {
			panic(err)
		}
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_close_writable_stream_object_url", func(url string) {
		if cfg.CloseWritableStreamObjectURL == nil {
			panic(fmt.Errorf("gblob writable stream close handler not configured"))
		}
		if err := cfg.CloseWritableStreamObjectURL(url); err != nil {
			panic(err)
		}
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_abort_writable_stream_object_url", func(url string, reason string) {
		if cfg.AbortWritableStreamObjectURL == nil {
			panic(fmt.Errorf("gblob writable stream abort handler not configured"))
		}
		if err := cfg.AbortWritableStreamObjectURL(url, reason); err != nil {
			panic(err)
		}
	}); err != nil {
		return err
	}
	if err := runtime.Set("__gopeed_revoke_object_url", func(url string) {
		if cfg.RevokeObjectURL == nil {
			panic(fmt.Errorf("gblob revoke handler not configured"))
		}
		if err := cfg.RevokeObjectURL(url); err != nil {
			panic(err)
		}
	}); err != nil {
		return err
	}
	_, err := runtime.RunString(script)
	return err
}
