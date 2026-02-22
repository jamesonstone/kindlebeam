package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents user configuration stored on disk.
type Config struct {
	KindleEmail         string `json:"kindle_email"`
	DefaultKindleEmail  string `json:"default_kindle_email,omitempty"`
	DefaultOutputFormat string `json:"default_output_format,omitempty"`
	DefaultInputFormat  string `json:"default_input_format,omitempty"`
	MailCommand         string `json:"mail_command,omitempty"`
	PandocPath          string `json:"pandoc_path,omitempty"`
}

// Load reads configuration from disk or returns a default config if the file does not exist.
// It also returns the resolved config file path.
func Load() (Config, string, error) {
	path, err := resolvePath()
	if err != nil {
		return Config{}, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			return cfg, path, nil
		}
		return Config{}, path, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, path, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)
	return cfg, path, nil
}

// Save writes configuration to disk, creating parent directories as needed.
func Save(cfg Config) error {
	path, err := resolvePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// EffectiveKindleEmail returns the effective Kindle email, considering an override flag
// and legacy default field.
func (c Config) EffectiveKindleEmail(override string) string {
	if override != "" {
		return override
	}
	if c.KindleEmail != "" {
		return c.KindleEmail
	}
	return c.DefaultKindleEmail
}

// DefaultInput returns the default input format.
func (c Config) DefaultInput() string {
	if c.DefaultInputFormat != "" {
		return c.DefaultInputFormat
	}
	return "markdown"
}

// DefaultOutput returns the default output format.
func (c Config) DefaultOutput() string {
	if c.DefaultOutputFormat != "" {
		return c.DefaultOutputFormat
	}
	return "pdf"
}

// EffectiveMailCommand returns the mail command to use.
func (c Config) EffectiveMailCommand() string {
	if c.MailCommand != "" {
		return c.MailCommand
	}
	return "mail"
}

// EffectivePandocPath returns the pandoc command/binary to use.
func (c Config) EffectivePandocPath() string {
	if c.PandocPath != "" {
		return c.PandocPath
	}
	return "pandoc"
}

func defaultConfig() Config {
	cfg := Config{}
	applyDefaults(&cfg)
	return cfg
}

func applyDefaults(c *Config) {
	if c.DefaultInputFormat == "" {
		c.DefaultInputFormat = "markdown"
	}
	if c.DefaultOutputFormat == "" {
		c.DefaultOutputFormat = "pdf"
	}
	if c.MailCommand == "" {
		c.MailCommand = "mail"
	}
	if c.PandocPath == "" {
		c.PandocPath = "pandoc"
	}
}

func resolvePath() (string, error) {
	if explicit := os.Getenv("KINDLEBEAM_CONFIG"); explicit != "" {
		return explicit, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}

	return filepath.Join(dir, "kindlebeam", "config.json"), nil
}
