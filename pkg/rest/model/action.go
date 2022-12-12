package model

type Command struct {
	Protocol string `json:"protocol"`
	Action   string `json:"action"`
	Params   any    `json:"params"`
}
