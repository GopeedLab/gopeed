package model

type Action struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Params any    `json:"params"`
}
