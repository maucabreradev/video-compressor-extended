package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config contains the user-configurable settings shared by the CLI and TUI.
type Config struct {
	InputRoot       string `toml:"input_root"`
	OutputDir       string `toml:"output_dir"`
	EncodeWorkers   int    `toml:"encode_workers"`
	SubtitleWorkers int    `toml:"subtitle_workers"`
	WhisperThreads  int    `toml:"whisper_threads"`
	Language        string `toml:"language"`
}

// Defaults returns the conservative settings used when no config file is
// supplied or when a TOML file omits a setting.
func Defaults() Config {
	return Config{
		InputRoot:       ".",
		OutputDir:       "vcx_processed",
		EncodeWorkers:   1,
		SubtitleWorkers: 1,
		WhisperThreads:  1,
		Language:        "en",
	}
}

// Load reads TOML settings over the project defaults. An empty path selects
// defaults without reading a file, then validates the resulting configuration.
func Load(path string) (Config, error) {
	cfg, err := Parse(path)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}
	return cfg, nil
}

// Parse reads TOML settings over the project defaults without semantic
// validation. Callers that apply command-line overrides should validate after
// applying those overrides.
func Parse(path string) (Config, error) {
	cfg := Defaults()
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer file.Close()

	decoder := toml.NewDecoder(file).DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	return cfg, nil
}

// Validate checks that settings are usable before any media processing starts.
func (c Config) Validate() error {
	if strings.TrimSpace(c.InputRoot) == "" {
		return fmt.Errorf("input_root must not be empty")
	}
	if strings.TrimSpace(c.OutputDir) == "" {
		return fmt.Errorf("output_dir must not be empty")
	}
	if c.EncodeWorkers < 1 {
		return fmt.Errorf("encode_workers must be at least 1, got %d", c.EncodeWorkers)
	}
	if c.SubtitleWorkers < 1 {
		return fmt.Errorf("subtitle_workers must be at least 1, got %d", c.SubtitleWorkers)
	}
	if c.WhisperThreads < 1 {
		return fmt.Errorf("whisper_threads must be at least 1, got %d", c.WhisperThreads)
	}
	if strings.TrimSpace(c.Language) == "" {
		return fmt.Errorf("language must not be empty")
	}
	return nil
}
