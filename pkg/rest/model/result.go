package model

type Result[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func NewOk[T any]() *Result[T] {
	return &Result[T]{
		Code: 0,
		Msg:  "ok",
	}
}

func NewOkWithData[T any](data T) *Result[T] {
	result := NewOk[T]()
	result.Data = data
	return result
}

func NewError(code int, msg string) *Result[any] {
	return &Result[any]{
		Code: code,
		Msg:  msg,
	}
}

func NewErrorWithData(code int, msg string, data any) *Result[any] {
	result := NewError(code, msg)
	result.Data = data
	return result
}
