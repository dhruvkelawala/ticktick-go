package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Timezone       string `json:"timezone"`
	DefaultProject string `json:"default_project"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
}

var defaultConfig = Config{
	Timezone:       "Europe/London",
	DefaultProject: "inbox",
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ttg", "config.json")
}

func Load() *Config {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return &defaultConfig
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &defaultConfig
	}

	if cfg.Timezone == "" {
		cfg.Timezone = defaultConfig.Timezone
	}

	return &cfg
}

func EnsureConfigDir() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "ttg")
	return os.MkdirAll(dir, 0755)
}
