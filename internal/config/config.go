package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	Port         int `json:"port"`
	ReadTimeout  int `json:"read_timeout_seconds"`
	WriteTimeout int `json:"write_timeout_seconds"`
	IdleTimeout  int `json:"idle_timeout_seconds"`
}

type Config struct {
	Server ServerConfig `json:"server"`
}

// Load reads the configuration from a JSON file based on the environment name.
func Load(env string) (*Config, error) {
	// e.g. "config/config.dev.json"
	filePath := fmt.Sprintf("config/config.%s.json", env)

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}
