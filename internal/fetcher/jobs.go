package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type TriggerSource string

const (
	TriggerScheduled TriggerSource = "scheduled"
	TriggerManual    TriggerSource = "manual"
)

var ErrJobSkipped = errors.New("protocol job skipped")
var ErrConfigChanged = errors.New("protocol configuration changed during execution")

// ConfigState embeds the common backend-owned state section in a protocol config.
// Public config PUTs always retain this section from the latest stored config.
type ConfigState[T any] struct {
	State T `json:"state"`
}

// ProtocolHost is scoped to one manager. UpdateState runs against the latest
// config under Downloader's lock; callbacks must not perform network I/O.
type ProtocolHost struct {
	Read        func(any) error
	UpdateState func(func(config, state json.RawMessage) (any, error)) error
	Fetch       func(context.Context, string, int64, func([]byte) error) ([]byte, error)
}

func ReadConfig[T any](host ProtocolHost) (*T, error) {
	var cfg T
	if err := host.Read(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateState gives the callback a detached config snapshot and mutable state.
// Only the state is committed; edits to the config snapshot are discarded.
func UpdateState[C, S any](host ProtocolHost, update func(*C, *S) error) error {
	return host.UpdateState(func(rawConfig, rawState json.RawMessage) (any, error) {
		var cfg C
		var state S
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawState, &state); err != nil {
			return nil, err
		}
		if err := update(&cfg, &state); err != nil {
			return nil, err
		}
		return &state, nil
	})
}

type CronJob struct {
	Name    string
	Every   time.Duration
	Timeout time.Duration
	Run     func(context.Context, TriggerSource) error
}
type CronProvider interface{ CronJobs(ProtocolHost) []CronJob }

// ConfigNormalizer is optional. It owns protocol validation and derived fields.
type ConfigNormalizer interface {
	NormalizeConfig(json.RawMessage) (any, error)
}
