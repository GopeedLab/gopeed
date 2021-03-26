package download

type EventKey string

const (
	EventKeyStart    = "start"
	EventKeyPause    = "pause"
	EventKeyContinue = "continue"
	EventKeyProgress = "progress"
	EventKeyError    = "error"
	EventKeyDone     = "done"
	EventKeyFinally  = "finally"
)

type Event struct {
	Key  EventKey
	Task *TaskInfo
	Err  error
}
