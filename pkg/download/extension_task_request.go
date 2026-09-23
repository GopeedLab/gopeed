package download

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/GopeedLab/gopeed/pkg/base"
	btProtocol "github.com/GopeedLab/gopeed/pkg/protocol/bt"
	httpProtocol "github.com/GopeedLab/gopeed/pkg/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/dop251/goja"
)

// ExtensionTaskRequest exposes request operations without exposing the backing task.
// It always reads the current request so a retained handle sees subsequent updates.
type ExtensionTaskRequest struct {
	*base.Request
	task *Task
}

// SetUrl replaces the task request URL.
func (r *ExtensionTaskRequest) SetUrl(url string) { r.task.Meta.Req.URL = url }

// SetLabels replaces all labels on the request.
func (r *ExtensionTaskRequest) SetLabels(labels map[string]string) {
	r.Labels = labels
}

// PutLabel sets a label on the request.
func (r *ExtensionTaskRequest) PutLabel(key, value string) {
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	r.Labels[key] = value
}

// DelLabel deletes a label from the request.
func (r *ExtensionTaskRequest) DelLabel(key string) {
	delete(r.Labels, key)
}

// mutateTaskExtra converts into an independent protocol value before mutation.
// Other protocols are a no-op. Failed conversion or mutation must not change the live request.
func mutateTaskExtra[E any](task *Task, protocol string, mutate func(*E) error) error {
	if task.Protocol != protocol {
		return nil
	}
	if task.Meta == nil || task.Meta.Req == nil {
		return fmt.Errorf("task request is missing")
	}
	var extra E
	if err := util.MapToStruct(task.Meta.Req.Extra, &extra); err != nil {
		return err
	}
	if err := mutate(&extra); err != nil {
		return err
	}
	task.Meta.Req.Extra = &extra
	return nil
}

func (t *ExtensionTaskRequest) SetMethod(method string) error {
	return mutateTaskExtra(t.task, "http", func(extra *httpProtocol.ReqExtra) error { extra.Method = method; return nil })
}

func (t *ExtensionTaskRequest) SetBody(body string) error {
	return mutateTaskExtra(t.task, "http", func(extra *httpProtocol.ReqExtra) error { extra.Body = body; return nil })
}

// decodeHeaders omits undefined properties and rejects non-string values.
func decodeHeaders(value *goja.Object) (map[string]string, error) {
	if value == nil {
		return nil, fmt.Errorf("headers must be an object")
	}
	data, err := value.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var input map[string]*string
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("headers must be an object")
	}
	headers := make(map[string]string, len(input))
	seen := make(map[string]bool, len(input))
	for name, value := range input {
		if value == nil {
			return nil, fmt.Errorf("header %q must be a string", name)
		}
		lower := strings.ToLower(name)
		if seen[lower] {
			return nil, fmt.Errorf("duplicate header name %q", name)
		}
		seen[lower] = true
		headers[http.CanonicalHeaderKey(name)] = *value
	}
	return headers, nil
}

func (t *ExtensionTaskRequest) SetHeaders(headers *goja.Object) error {
	return mutateTaskExtra(t.task, "http", func(extra *httpProtocol.ReqExtra) error {
		values, err := decodeHeaders(headers)
		if err != nil {
			return err
		}
		extra.Header = values
		return nil
	})
}

func (t *ExtensionTaskRequest) PutHeader(name, value string) error {
	return mutateTaskExtra(t.task, "http", func(extra *httpProtocol.ReqExtra) error {
		if extra.Header == nil {
			extra.Header = make(map[string]string)
		}
		deleteHeader(extra.Header, name)
		extra.Header[http.CanonicalHeaderKey(name)] = value
		return nil
	})
}

func (t *ExtensionTaskRequest) DelHeader(name string) error {
	return mutateTaskExtra(t.task, "http", func(extra *httpProtocol.ReqExtra) error { deleteHeader(extra.Header, name); return nil })
}

func deleteHeader(headers map[string]string, name string) {
	for key := range headers {
		if strings.EqualFold(key, name) {
			delete(headers, key)
		}
	}
}

func (t *ExtensionTaskRequest) SetTrackers(trackers []string) error {
	return mutateTaskExtra(t.task, "bt", func(extra *btProtocol.ReqExtra) error { extra.Trackers = append([]string{}, trackers...); return nil })
}
