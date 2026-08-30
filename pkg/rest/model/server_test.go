package model

import "testing"

func TestStartConfigValidateWebAPIToken(t *testing.T) {
	tests := []struct {
		name    string
		config  *StartConfig
		wantErr bool
	}{
		{
			name:    "web API token without web auth",
			config:  &StartConfig{WebEnable: true, ApiToken: "token"},
			wantErr: true,
		},
		{
			name: "web API token with web auth",
			config: &StartConfig{
				WebEnable: true,
				ApiToken:  "token",
				WebAuth: &WebAuth{
					Username: "admin",
					Password: "secret",
				},
			},
		},
		{
			name:   "web without authentication",
			config: &StartConfig{WebEnable: true},
		},
		{
			name:   "native REST API token",
			config: &StartConfig{NativeMode: true, ApiToken: "token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
