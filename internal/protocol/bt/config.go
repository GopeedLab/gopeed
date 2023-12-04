package bt

type config struct {
	ListenPort int      `json:"listenPort"`
	Trackers   []string `json:"trackers"`
}
