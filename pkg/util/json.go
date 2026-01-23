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

func DeepClone[T any](v *T) *T {
	if v == nil {
		return nil
	}

	var t T
	b, err := json.Marshal(v)
	if err != nil {
		return &t
	}
	json.Unmarshal(b, &t)
	return &t
}

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// BoolPtr returns a pointer to a bool value.
func BoolPtr(v bool) *bool {
	return &v
}
