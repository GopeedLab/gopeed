package cmd

import (
	"strings"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/rest/model"
)

func TestStartInvalidWebAPITokenReturnsError(t *testing.T) {
	tests := []struct {
		name    string
		webAuth *model.WebAuth
	}{
		{name: "no credentials"},
		{name: "missing username", webAuth: &model.WebAuth{Password: "secret"}},
		{name: "missing password", webAuth: &model.WebAuth{Username: "gopeed"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Start(&model.StartConfig{
				WebEnable: true,
				ApiToken:  "test-token",
				WebAuth:   tt.webAuth,
			})
			if err == nil {
				t.Fatal("Start() error = nil, want web authentication validation error")
			}
			if !strings.Contains(err.Error(), "web authentication username and password are required") {
				t.Fatalf("Start() error = %q, want missing web authentication credentials", err)
			}
		})
	}
}
