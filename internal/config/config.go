package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Editor string   `yaml:"editor"`
	UI     UIConfig `yaml:"ui"`
}

type UIConfig struct {
	LineNumbers string `yaml:"line_numbers"`
	Theme       string `yaml:"theme"`
}

func Load() Config {
	cfg := Config{
		UI: UIConfig{
			LineNumbers: "hybrid",
			Theme:       "default",
		},
	}

	configPath := os.Getenv("DIFI_CONFIG")
	if configPath == "" {
		if configDir, err := os.UserConfigDir(); err == nil {
			configPath = filepath.Join(configDir, "difi", "config.yaml")
		}
	}

	if configPath != "" {
		//nolint:gosec // User-controlled config path is intentional.
		if data, err := os.ReadFile(configPath); err == nil {
			_ = yaml.Unmarshal(data, &cfg)
		}
	}

	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("DIFI_EDITOR")
	}
	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("EDITOR")
	}
	if cfg.Editor == "" {
		cfg.Editor = os.Getenv("VISUAL")
	}
	if cfg.Editor == "" {
		cfg.Editor = "vi"
	}

	return cfg
}
