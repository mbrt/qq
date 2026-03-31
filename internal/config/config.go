// Package config handles loading and parsing the qq configuration file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the top-level application configuration.
type Config struct {
	Directories []Directory `yaml:"directories"`
	IndexPath   string      `yaml:"index_path"`
}

// Directory represents a configured directory to scan for documents.
type Directory struct {
	Path string `yaml:"path"`
}

// Load reads and parses the configuration from the given path.
// If path is empty, it defaults to ~/.config/qq/config.yaml.
func Load(path string) (Config, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("getting home directory: %w", err)
		}
		path = filepath.Join(home, ".config", "qq", "config.yaml")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}
	if len(cfg.Directories) == 0 {
		return Config{}, fmt.Errorf("no directories configured")
	}
	for i, d := range cfg.Directories {
		cfg.Directories[i].Path = expandHome(d.Path)
	}
	if cfg.IndexPath == "" {
		home, _ := os.UserHomeDir()
		cfg.IndexPath = filepath.Join(home, ".local", "share", "qq", "index")
	} else {
		cfg.IndexPath = expandHome(cfg.IndexPath)
	}
	return cfg, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
