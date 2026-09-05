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
	Address           *string  `json:"address"`
	Port              *int     `json:"port"`
	Username          *string  `json:"username"`
	Password          *string  `json:"password"`
	ApiToken          *string  `json:"apiToken"`
	MCPEnable         bool     `json:"mcpEnable"`
	StorageDir        *string  `json:"storageDir"`
	WhiteDownloadDirs []string `json:"whiteDownloadDirs"`
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
	// set default values
	setDefaults(cfg, cliConfig)
	return cfg
}

// loadCliArgs parses command line arguments and returns initial config
func loadCliArgs() *args {
	cfg := &args{}
	cfg.Address = flag.String("A", "0.0.0.0", "Bind Address")
	flag.StringVar(cfg.Address, "address", "0.0.0.0", "Bind Address")
	cfg.Port = flag.Int("P", 9999, "Bind Port")
	flag.IntVar(cfg.Port, "port", 9999, "Bind Port")
	cfg.Username = flag.String("u", "gopeed", "Web Authentication Username")
	flag.StringVar(cfg.Username, "username", "gopeed", "Web Authentication Username")
	cfg.Password = flag.String("p", "", "Web Authentication Password, if no password is set, web authentication will not be enabled")
	flag.StringVar(cfg.Password, "password", "", "Web Authentication Password, if no password is set, web authentication will not be enabled")
	cfg.ApiToken = flag.String("T", "", "API token, it must be configured when using HTTP API in the case of enabling web authentication")
	flag.StringVar(cfg.ApiToken, "api-token", "", "API token, it must be configured when using HTTP API in the case of enabling web authentication")
	flag.BoolVar(&cfg.MCPEnable, "mcp-enable", false, "Enable MCP endpoint")
	cfg.StorageDir = flag.String("d", "", "Storage directory")
	flag.StringVar(cfg.StorageDir, "storage-dir", "", "Storage directory")
	whiteDownloadDirs := flag.String("white-download-dirs", "", "White download directories, comma-separated")
	cfg.configPath = flag.String("c", "./config.json", "Config file path")
	flag.StringVar(cfg.configPath, "config", "./config.json", "Config file path")
	flag.Parse()

	// Parse white download directories from comma-separated string
	if whiteDownloadDirs != nil && *whiteDownloadDirs != "" {
		dirs := strings.Split(*whiteDownloadDirs, ",")
		for i := range dirs {
			dirs[i] = strings.TrimSpace(dirs[i])
		}
		cfg.WhiteDownloadDirs = dirs
	}

	return cfg
}

// overrideWithCliArgs overrides config with non-empty command line arguments
func overrideWithCliArgs(cfg *args, cliConfig *args) {
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "A", "address":
			cfg.Address = cliConfig.Address
		case "P", "port":
			cfg.Port = cliConfig.Port
		case "u", "username":
			cfg.Username = cliConfig.Username
		case "p", "password":
			cfg.Password = cliConfig.Password
		case "T", "api-token":
			cfg.ApiToken = cliConfig.ApiToken
		case "mcp-enable":
			cfg.MCPEnable = cliConfig.MCPEnable
		case "d", "storage-dir":
			cfg.StorageDir = cliConfig.StorageDir
		case "white-download-dirs":
			cfg.WhiteDownloadDirs = cliConfig.WhiteDownloadDirs
		case "c", "config":
			cfg.configPath = cliConfig.configPath
		}
	})
}

// setDefaults sets default values for any unset configuration fields
func setDefaults(cfg *args, cliConfig *args) {
	if cfg.Address == nil {
		cfg.Address = cliConfig.Address
	}
	if cfg.Port == nil {
		cfg.Port = cliConfig.Port
	}
	if cfg.Username == nil {
		cfg.Username = cliConfig.Username
	}
	if cfg.Password == nil {
		cfg.Password = cliConfig.Password
	}
	if cfg.ApiToken == nil {
		cfg.ApiToken = cliConfig.ApiToken
	}
	if cfg.StorageDir == nil {
		cfg.StorageDir = cliConfig.StorageDir
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
		} else if field.Kind() == reflect.Slice {
			// Handle non-pointer slice types (like []string for WhiteDownloadDirs)
			if field.Type().Elem().Kind() == reflect.String {
				dirs := strings.Split(envValue, ",")
				for i := range dirs {
					dirs[i] = strings.TrimSpace(dirs[i])
				}
				field.Set(reflect.ValueOf(dirs))
			}
		} else if field.Kind() == reflect.Bool {
			if boolVal, err := strconv.ParseBool(envValue); err == nil {
				field.SetBool(boolVal)
			}
		}
	}
}
