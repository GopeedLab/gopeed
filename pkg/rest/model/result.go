package model

type Result[T any] struct {
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func NewResult[T any](msg string, data T) *Result[T] {
	return &Result[T]{
		Msg:  msg,
		Data: data,
	}
}

func NewResultWithMsg(msg string) *Result[any] {
	return NewResult[any](msg, nil)
}

func NewResultWithData[T any](data T) *Result[T] {
	return NewResult[T]("", data)
}
