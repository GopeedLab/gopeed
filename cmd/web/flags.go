package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/GopeedLab/gopeed/pkg/base"
)

type args struct {
	Address    *string `json:"address"`
	Port       *int    `json:"port"`
	Username   *string `json:"username"`
	Password   *string `json:"password"`
	ApiToken   *string `json:"apiToken"`
	StorageDir *string `json:"storageDir"`
	// DownloadConfig when the first time to start the server, it will be configured as initial value
	DownloadConfig *base.DownloaderStoreConfig `json:"downloadConfig"`

	configPath *string
}

func parse() *args {
	cfg := &args{}

	cliConfig := loadCliArgs()
	loadConfigFile(cfg, *cliConfig.configPath)
	loadEnvVars(cfg)
	// override with non-default command line arguments
	overrideWithCliArgs(cfg, cliConfig)
	// set default values for any unset fields
	setDefaults(cfg)
	return cfg
}

// loadCliArgs parses command line arguments and returns initial config
func loadCliArgs() *args {
	cfg := &args{}
	cfg.Address = flag.String("A", "", "Bind Address")
	cfg.Port = flag.Int("P", 0, "Bind Port")
	cfg.Username = flag.String("u", "", "Web Authentication Username")
	cfg.Password = flag.String("p", "", "Web Authentication Password, if no password is set, web authentication will not be enabled")
	cfg.ApiToken = flag.String("T", "", "API token, it must be configured when using HTTP API in the case of enabling web authentication")
	cfg.StorageDir = flag.String("d", "", "Storage directory")
	cfg.configPath = flag.String("c", "./config.json", "Config file path")
	flag.Parse()

	return cfg
}

// overrideWithCliArgs overrides config with non-empty command line arguments
func overrideWithCliArgs(cfg *args, cliConfig *args) {
	// Only override if the cli value is not empty/zero
	if cliConfig.Address != nil && *cliConfig.Address != "" {
		cfg.Address = cliConfig.Address
	}

	if cliConfig.Port != nil && *cliConfig.Port != 0 {
		cfg.Port = cliConfig.Port
	}

	if cliConfig.Username != nil && *cliConfig.Username != "" {
		cfg.Username = cliConfig.Username
	}

	if cliConfig.Password != nil && *cliConfig.Password != "" {
		cfg.Password = cliConfig.Password
	}

	if cliConfig.ApiToken != nil && *cliConfig.ApiToken != "" {
		cfg.ApiToken = cliConfig.ApiToken
	}

	if cliConfig.StorageDir != nil && *cliConfig.StorageDir != "" {
		cfg.StorageDir = cliConfig.StorageDir
	}
}

// setDefaults sets default values for any unset configuration fields
func setDefaults(cfg *args) {
	if cfg.Address == nil {
		address := "0.0.0.0"
		cfg.Address = &address
	}

	if cfg.Port == nil {
		port := 9999
		cfg.Port = &port
	}

	if cfg.Username == nil {
		username := "gopeed"
		cfg.Username = &username
	}
}

// loadConfigFile loads configuration from file
func loadConfigFile(cfg *args, configPath string) {
	if !filepath.IsAbs(configPath) {
		dir, err := os.Getwd()
		if err != nil {
			return
		}
		configPath = filepath.Join(dir, configPath)
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	if err = json.Unmarshal(file, cfg); err != nil {
		return
	}
}

// loadEnvVars loads configuration from environment variables with prefix GOPEED_
func loadEnvVars(cfg *args) {
	v := reflect.ValueOf(cfg).Elem()
	t := reflect.TypeOf(cfg).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Get json tag as environment variable suffix
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Remove options like omitempty
		if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
			jsonTag = jsonTag[:commaIdx]
		}

		// Convert to uppercase and add GOPEED_ prefix
		envKey := "GOPEED_" + strings.ToUpper(jsonTag)
		envValue := os.Getenv(envKey)

		if envValue == "" {
			continue
		}

		// Set value based on field type
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				// Create new pointer instance
				newVal := reflect.New(field.Type().Elem())
				field.Set(newVal)
			}

			switch field.Type().Elem().Kind() {
			case reflect.String:
				field.Elem().SetString(envValue)
			case reflect.Int:
				if intVal, err := strconv.Atoi(envValue); err == nil {
					field.Elem().SetInt(int64(intVal))
				}
			default:
				// For complex types like DownloadConfig, try JSON unmarshaling
				if field.Type().Elem() == reflect.TypeOf(base.DownloaderStoreConfig{}) {
					var config base.DownloaderStoreConfig
					if err := json.Unmarshal([]byte(envValue), &config); err == nil {
						field.Set(reflect.ValueOf(&config))
					}
				}
			}
		}
	}
}
