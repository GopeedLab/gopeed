package util

import (
	"reflect"
	"testing"
)

func TestDeepClone(t *testing.T) {
	type user struct {
		Name string `json:"name"`
		Age  int    `json:"age"`

		v int
	}

	type args[T any] struct {
		v *T
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want *T
	}
	tests := []testCase[user]{
		{
			name: "case 1",
			args: args[user]{
				v: &user{
					Name: "test",
					Age:  10,
				},
			},
			want: &user{
				Name: "test",
				Age:  10,
			},
		},
		{
			name: "case 2",
			args: args[user]{
				v: &user{
					Name: "test",
					Age:  10,
					v:    1,
				},
			},
			want: &user{
				Name: "test",
				Age:  10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeepClone(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeepClone() = %v, want %v", got, tt.want)
			}
		})
	}
}
