package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config contains the user-configurable settings shared by the CLI and TUI.
type Config struct {
	InputRoot        string `toml:"input_root"`
	OutputDir        string `toml:"output_dir"`
	EncodeWorkers    int    `toml:"encode_workers"`
	SubtitleWorkers  int    `toml:"subtitle_workers"`
	WhisperThreads   int    `toml:"whisper_threads"`
	Language         string `toml:"language"`
	VideoCodec       string `toml:"video_codec"`
	WhisperModelPath string `toml:"whisper_model_path"`
	MaxWidth         int    `toml:"max_width"`
	MaxHeight        int    `toml:"max_height"`
	FrameRate        int    `toml:"frame_rate"`
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
		VideoCodec:      "h264",
		MaxWidth:        1280,
		MaxHeight:       720,
		FrameRate:       0,
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
	if c.MaxWidth < 2 || c.MaxWidth > 16384 {
		return fmt.Errorf("max_width must be between 2 and 16384, got %d", c.MaxWidth)
	}
	if c.MaxHeight < 2 || c.MaxHeight > 16384 {
		return fmt.Errorf("max_height must be between 2 and 16384, got %d", c.MaxHeight)
	}
	if c.FrameRate < 0 || c.FrameRate > 240 {
		return fmt.Errorf("frame_rate must be 0 (preserve source) or between 1 and 240, got %d", c.FrameRate)
	}
	switch strings.ToLower(strings.TrimSpace(c.VideoCodec)) {
	case "h264", "hevc":
	default:
		return fmt.Errorf("video_codec must be h264 or hevc, got %q", c.VideoCodec)
	}
	return nil
}
