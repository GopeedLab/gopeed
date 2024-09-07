package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/GopeedLab/gopeed/pkg/base"
	"os"
	"path/filepath"
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
	var cliArgs args
	cliArgs.Address = flag.String("A", "0.0.0.0", "Bind Address")
	cliArgs.Port = flag.Int("P", 9999, "Bind Port")
	cliArgs.Username = flag.String("u", "gopeed", "HTTP Basic Auth Username")
	cliArgs.Password = flag.String("p", "", "HTTP Basic Auth Pwd")
	cliArgs.ApiToken = flag.String("T", "", "API token, it must be configured when using HTTP API in the case of enabling basic authentication")
	cliArgs.StorageDir = flag.String("d", "", "Storage directory")
	cliArgs.configPath = flag.String("c", "./config.json", "Config file path")
	flag.Parse()

	// args priority: config file > cli args
	cfgArgs := loadConfig(*cliArgs.configPath)
	if cfgArgs.Address == nil {
		cfgArgs.Address = cliArgs.Address
	}
	if cfgArgs.Port == nil {
		cfgArgs.Port = cliArgs.Port
	}
	if cfgArgs.Username == nil {
		cfgArgs.Username = cliArgs.Username
	}
	if cfgArgs.Password == nil {
		cfgArgs.Password = cliArgs.Password
	}
	if cfgArgs.ApiToken == nil {
		cfgArgs.ApiToken = cliArgs.ApiToken
	}
	if cfgArgs.StorageDir == nil {
		cfgArgs.StorageDir = cliArgs.StorageDir
	}
	return cfgArgs
}

func loadConfig(path string) *args {
	var args args

	if !filepath.IsAbs(path) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("config dir get failed, reason:" + err.Error())
			return &args
		}
		path = filepath.Join(dir, path)
	}
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &args
		}
		fmt.Println("config file read failed, reason:" + err.Error())
		return &args
	}
	if err = json.Unmarshal(file, &args); err != nil {
		fmt.Println("config file parse failed, reason:" + err.Error())
		return &args
	}
	fmt.Printf("config file loaded: %s\n", path)
	return &args
}
