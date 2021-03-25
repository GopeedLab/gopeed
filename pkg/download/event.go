package download

type EventKey string

const (
	EventKeyStart    = "start"
	EventKeyPause    = "pause"
	EventKeyContinue = "continue"
	EventKeyProgress = "progress"
	EventKeyError    = "error"
	EventKeyDone     = "done"
)

type Event struct {
	EventKey
	TaskInfo
}
