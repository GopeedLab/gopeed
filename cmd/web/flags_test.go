package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    *args
		expected *args
	}{
		{
			name:  "empty config should get defaults",
			input: &args{},
			expected: &args{
				Address:  stringPtr("0.0.0.0"),
				Port:     intPtr(9999),
				Username: stringPtr("gopeed"),
			},
		},
		{
			name: "partial config should only fill missing defaults",
			input: &args{
				Address: stringPtr("127.0.0.1"),
			},
			expected: &args{
				Address:  stringPtr("127.0.0.1"),
				Port:     intPtr(9999),
				Username: stringPtr("gopeed"),
			},
		},
		{
			name: "full config should remain unchanged",
			input: &args{
				Address:  stringPtr("192.168.1.1"),
				Port:     intPtr(8080),
				Username: stringPtr("admin"),
			},
			expected: &args{
				Address:  stringPtr("192.168.1.1"),
				Port:     intPtr(8080),
				Username: stringPtr("admin"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDefaults(tt.input)
			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("setDefaults() = %+v, want %+v", tt.input, tt.expected)
			}
		})
	}
}

func TestOverrideWithCliArgs(t *testing.T) {
	tests := []struct {
		name      string
		config    *args
		cliConfig *args
		expected  *args
	}{
		{
			name:   "empty cli should not override",
			config: &args{Address: stringPtr("192.168.1.1")},
			cliConfig: &args{
				Address: stringPtr(""),
				Port:    intPtr(0),
			},
			expected: &args{Address: stringPtr("192.168.1.1")},
		},
		{
			name:   "non-empty cli should override",
			config: &args{Address: stringPtr("192.168.1.1")},
			cliConfig: &args{
				Address: stringPtr("127.0.0.1"),
				Port:    intPtr(8080),
			},
			expected: &args{
				Address: stringPtr("127.0.0.1"),
				Port:    intPtr(8080),
			},
		},
		{
			name: "all fields override test",
			config: &args{
				Address:    stringPtr("old_address"),
				Port:       intPtr(1111),
				Username:   stringPtr("old_user"),
				Password:   stringPtr("old_pass"),
				ApiToken:   stringPtr("old_token"),
				StorageDir: stringPtr("old_dir"),
			},
			cliConfig: &args{
				Address:    stringPtr("new_address"),
				Port:       intPtr(2222),
				Username:   stringPtr("new_user"),
				Password:   stringPtr("new_pass"),
				ApiToken:   stringPtr("new_token"),
				StorageDir: stringPtr("new_dir"),
			},
			expected: &args{
				Address:    stringPtr("new_address"),
				Port:       intPtr(2222),
				Username:   stringPtr("new_user"),
				Password:   stringPtr("new_pass"),
				ApiToken:   stringPtr("new_token"),
				StorageDir: stringPtr("new_dir"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrideWithCliArgs(tt.config, tt.cliConfig)
			if !reflect.DeepEqual(tt.config, tt.expected) {
				t.Errorf("overrideWithCliArgs() = %+v, want %+v", tt.config, tt.expected)
			}
		})
	}
}

func TestLoadConfigFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "flags_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name       string
		configData string
		fileName   string
		expected   *args
	}{
		{
			name: "valid config file",
			configData: `{
				"address": "192.168.1.100",
				"port": 8080,
				"username": "testuser",
				"password": "testpass",
				"apiToken": "testtoken",
				"storageDir": "/test/storage",
				"downloadConfig": {
					"downloadDir": "/test/downloads",
					"maxRunning": 10
				}
			}`,
			fileName: "valid_config.json",
			expected: &args{
				Address:    stringPtr("192.168.1.100"),
				Port:       intPtr(8080),
				Username:   stringPtr("testuser"),
				Password:   stringPtr("testpass"),
				ApiToken:   stringPtr("testtoken"),
				StorageDir: stringPtr("/test/storage"),
				DownloadConfig: &base.DownloaderStoreConfig{
					DownloadDir: "/test/downloads",
					MaxRunning:  10,
				},
			},
		},
		{
			name: "partial config file",
			configData: `{
				"address": "10.0.0.1",
				"port": 3000
			}`,
			fileName: "partial_config.json",
			expected: &args{
				Address: stringPtr("10.0.0.1"),
				Port:    intPtr(3000),
			},
		},
		{
			name:       "invalid json should not panic",
			configData: `{invalid json}`,
			fileName:   "invalid_config.json",
			expected:   &args{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config file
			configPath := filepath.Join(tempDir, tt.fileName)
			err := os.WriteFile(configPath, []byte(tt.configData), 0644)
			if err != nil {
				t.Fatal(err)
			}

			cfg := &args{}
			loadConfigFile(cfg, configPath)

			if !reflect.DeepEqual(cfg, tt.expected) {
				t.Errorf("loadConfigFile() = %+v, want %+v", cfg, tt.expected)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		cfg := &args{}
		loadConfigFile(cfg, "/non/existent/file.json")
		expected := &args{}
		if !reflect.DeepEqual(cfg, expected) {
			t.Errorf("loadConfigFile() with non-existent file = %+v, want %+v", cfg, expected)
		}
	})
}

func TestLoadEnvVars(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envKeys := []string{
		"GOPEED_ADDRESS", "GOPEED_PORT", "GOPEED_USERNAME",
		"GOPEED_PASSWORD", "GOPEED_APITOKEN", "GOPEED_STORAGEDIR",
		"GOPEED_DOWNLOADCONFIG",
	}
	for _, key := range envKeys {
		originalEnv[key] = os.Getenv(key)
	}

	// Clean up function
	cleanup := func() {
		for _, key := range envKeys {
			if val, exists := originalEnv[key]; exists {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
	defer cleanup()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected *args
	}{
		{
			name: "all environment variables set",
			envVars: map[string]string{
				"GOPEED_ADDRESS":    "env.example.com",
				"GOPEED_PORT":       "7777",
				"GOPEED_USERNAME":   "envuser",
				"GOPEED_PASSWORD":   "envpass",
				"GOPEED_APITOKEN":   "envtoken",
				"GOPEED_STORAGEDIR": "/env/storage",
			},
			expected: &args{
				Address:    stringPtr("env.example.com"),
				Port:       intPtr(7777),
				Username:   stringPtr("envuser"),
				Password:   stringPtr("envpass"),
				ApiToken:   stringPtr("envtoken"),
				StorageDir: stringPtr("/env/storage"),
			},
		},
		{
			name: "partial environment variables",
			envVars: map[string]string{
				"GOPEED_ADDRESS": "partial.example.com",
				"GOPEED_PORT":    "5555",
			},
			expected: &args{
				Address: stringPtr("partial.example.com"),
				Port:    intPtr(5555),
			},
		},
		{
			name: "downloadConfig from environment",
			envVars: map[string]string{
				"GOPEED_DOWNLOADCONFIG": `{"downloadDir": "/env/downloads", "maxRunning": 15}`,
			},
			expected: &args{
				DownloadConfig: &base.DownloaderStoreConfig{
					DownloadDir: "/env/downloads",
					MaxRunning:  15,
				},
			},
		},
		{
			name: "invalid port should be ignored",
			envVars: map[string]string{
				"GOPEED_PORT": "invalid_port",
			},
			expected: &args{
				Port: intPtr(0), // Invalid port creates pointer with 0 value
			},
		},
		{
			name: "invalid json for downloadConfig should be ignored",
			envVars: map[string]string{
				"GOPEED_DOWNLOADCONFIG": `{invalid json}`,
			},
			expected: &args{
				DownloadConfig: &base.DownloaderStoreConfig{}, // Invalid JSON creates empty config
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables first
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg := &args{}
			loadEnvVars(cfg)

			if !reflect.DeepEqual(cfg, tt.expected) {
				t.Errorf("loadEnvVars() = %+v, want %+v", cfg, tt.expected)
			}
		})
	}
}

func TestConfigPriority(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "flags_test_priority")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config file
	configData := `{
		"address": "config.example.com",
		"port": 6666,
		"username": "configuser",
		"password": "configpass"
	}`
	configPath := filepath.Join(tempDir, "test_config.json")
	err = os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Save original environment
	originalEnv := make(map[string]string)
	envKeys := []string{"GOPEED_ADDRESS", "GOPEED_PORT", "GOPEED_USERNAME"}
	for _, key := range envKeys {
		originalEnv[key] = os.Getenv(key)
	}
	defer func() {
		for _, key := range envKeys {
			if val, exists := originalEnv[key]; exists {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Set environment variables
	os.Setenv("GOPEED_ADDRESS", "env.example.com")
	os.Setenv("GOPEED_PORT", "7777")

	// Test configuration priority: CLI > ENV > Config File > Defaults
	cfg := &args{}

	// Load config file
	loadConfigFile(cfg, configPath)

	// Load environment variables (should override config file)
	loadEnvVars(cfg)

	// Simulate CLI args (should override env vars)
	cliConfig := &args{
		Address: stringPtr("cli.example.com"),
		// Port not set in CLI, should remain from env
		// Username not set in CLI or env, should remain from config file
	}
	overrideWithCliArgs(cfg, cliConfig)

	// Set defaults for missing fields
	setDefaults(cfg)

	expected := &args{
		Address:  stringPtr("cli.example.com"), // From CLI (highest priority)
		Port:     intPtr(7777),                 // From ENV (overrides config file)
		Username: stringPtr("configuser"),      // From config file (env not set)
		Password: stringPtr("configpass"),      // From config file only
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("Configuration priority test failed.\nGot: %+v\nWant: %+v", cfg, expected)
	}
}

func TestCompleteConfigurationFlow(t *testing.T) {
	// This test simulates a complete configuration loading scenario
	// with all sources: defaults, config file, environment variables, and CLI args

	tempDir, err := os.MkdirTemp("", "flags_test_complete")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create comprehensive config file
	configData := `{
		"address": "config.host.com",
		"port": 5000,
		"username": "configuser",
		"password": "configpass",
		"apiToken": "configtoken",
		"storageDir": "/config/storage",
		"downloadConfig": {
			"downloadDir": "/config/downloads",
			"maxRunning": 8,
			"protocolConfig": {"http": {"maxConnections": 16}},
			"proxy": {
				"enable": true,
				"system": false,
				"scheme": "http",
				"host": "proxy.example.com:8080"
			}
		}
	}`
	configPath := filepath.Join(tempDir, "complete_config.json")
	err = os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Save and set environment variables
	originalEnv := make(map[string]string)
	envKeys := []string{"GOPEED_ADDRESS", "GOPEED_USERNAME", "GOPEED_APITOKEN"}
	for _, key := range envKeys {
		originalEnv[key] = os.Getenv(key)
	}
	defer func() {
		for _, key := range envKeys {
			if val, exists := originalEnv[key]; exists {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	os.Setenv("GOPEED_ADDRESS", "env.host.com")
	os.Setenv("GOPEED_USERNAME", "envuser")

	// Simulate complete configuration loading
	cfg := &args{}

	// Load from config file
	loadConfigFile(cfg, configPath)

	// Override with environment variables
	loadEnvVars(cfg)

	// Override with CLI arguments
	cliConfig := &args{
		Port:       intPtr(9090),
		StorageDir: stringPtr("/cli/storage"),
	}
	overrideWithCliArgs(cfg, cliConfig)

	// Set defaults
	setDefaults(cfg)

	// Verify the final configuration follows priority rules
	if *cfg.Address != "env.host.com" {
		t.Errorf("Address should be from environment, got %s", *cfg.Address)
	}
	if *cfg.Port != 9090 {
		t.Errorf("Port should be from CLI, got %d", *cfg.Port)
	}
	if *cfg.Username != "envuser" {
		t.Errorf("Username should be from environment, got %s", *cfg.Username)
	}
	if *cfg.Password != "configpass" {
		t.Errorf("Password should be from config file, got %s", *cfg.Password)
	}
	if *cfg.ApiToken != "configtoken" {
		t.Errorf("ApiToken should be from config file, got %s", *cfg.ApiToken)
	}
	if *cfg.StorageDir != "/cli/storage" {
		t.Errorf("StorageDir should be from CLI, got %s", *cfg.StorageDir)
	}
	if cfg.DownloadConfig == nil {
		t.Error("DownloadConfig should be loaded from config file")
	} else {
		if cfg.DownloadConfig.DownloadDir != "/config/downloads" {
			t.Errorf("DownloadConfig.DownloadDir should be from config file, got %s", cfg.DownloadConfig.DownloadDir)
		}
		if cfg.DownloadConfig.MaxRunning != 8 {
			t.Errorf("DownloadConfig.MaxRunning should be from config file, got %d", cfg.DownloadConfig.MaxRunning)
		}
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
