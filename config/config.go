package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	AutoSyncEnabled    bool `json:"autoSyncEnabled"`
	MaxBackups         int  `json:"maxBackups"`
	ExperimentsEnabled bool `json:"experimentsEnabled"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() Config {
	return Config{
		AutoSyncEnabled:    false,
		MaxBackups:         10,
		ExperimentsEnabled: false,
	}
}

// configPath returns the path to the config file
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".smooth", "config.json"), nil
}

// Load reads the config from disk, returning defaults if not found
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if config doesn't exist
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// Ensure MaxBackups has a reasonable minimum
	if cfg.MaxBackups < 1 {
		cfg.MaxBackups = 1
	}

	return cfg, nil
}

// Save writes the config to disk
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Create .smooth directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
