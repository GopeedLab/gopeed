package download

type EventKey string

const (
	EventKeyStart    = "start"
	EventKeyPause    = "pause"
	EventKeyProgress = "progress"
	EventKeyError    = "error"
	EventKeyDelete   = "delete"
	EventKeyDone     = "done"
	EventKeyFinally  = "finally"
)

type Event struct {
	Key  EventKey
	Task *Task
	Err  error
}
