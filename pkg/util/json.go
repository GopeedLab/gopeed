package util

import "encoding/json"

func MapToStruct(s any, v any) error {
	if s == nil {
		return nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
