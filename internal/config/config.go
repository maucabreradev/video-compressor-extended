package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	ResourceProfileConservative = "conservative"
	ResourceProfileCustom       = "custom"

	ProcessPriorityNormal      = "normal"
	ProcessPriorityBelowNormal = "below_normal"
)

// ResourceSettings contains the effective per-stage limits and best-effort
// process settings selected by a configuration profile.
type ResourceSettings struct {
	EncodeWorkers   int
	SubtitleWorkers int
	WhisperThreads  int
	ProcessPriority string
	WhisperGPU      bool
}

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
	ResourceProfile  string `toml:"resource_profile"`
	ProcessPriority  string `toml:"process_priority"`
	WhisperGPU       bool   `toml:"whisper_gpu"`
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
		ResourceProfile: ResourceProfileConservative,
		ProcessPriority: ProcessPriorityBelowNormal,
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

	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	var keys map[string]any
	if err := toml.Unmarshal(contents, &keys); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	if _, hasProfile := keys["resource_profile"]; !hasProfile && hasResourceOverrides(keys) {
		cfg.ResourceProfile = ResourceProfileCustom
	}

	decoder := toml.NewDecoder(bytes.NewReader(contents)).DisallowUnknownFields()
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
	switch c.ResourceProfile {
	case ResourceProfileConservative, ResourceProfileCustom:
	default:
		return fmt.Errorf("resource_profile must be %q or %q, got %q", ResourceProfileConservative, ResourceProfileCustom, c.ResourceProfile)
	}
	switch c.ProcessPriority {
	case ProcessPriorityNormal, ProcessPriorityBelowNormal:
	default:
		return fmt.Errorf("process_priority must be %q or %q, got %q", ProcessPriorityNormal, ProcessPriorityBelowNormal, c.ProcessPriority)
	}
	if c.ResourceProfile == ResourceProfileConservative &&
		(c.EncodeWorkers != 1 || c.SubtitleWorkers != 1 || c.WhisperThreads != 1 ||
			c.ProcessPriority != ProcessPriorityBelowNormal || c.WhisperGPU) {
		return fmt.Errorf("resource_profile %q uses one worker/thread per stage, below-normal process priority, and CPU-only Whisper; select %q for custom settings", ResourceProfileConservative, ResourceProfileCustom)
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

// EffectiveResources returns the validated resource and acceleration settings.
func (c Config) EffectiveResources() ResourceSettings {
	return ResourceSettings{
		EncodeWorkers: c.EncodeWorkers, SubtitleWorkers: c.SubtitleWorkers,
		WhisperThreads: c.WhisperThreads, ProcessPriority: c.ProcessPriority,
		WhisperGPU: c.WhisperGPU,
	}
}

func hasResourceOverrides(keys map[string]any) bool {
	for _, key := range []string{
		"encode_workers", "subtitle_workers", "whisper_threads",
		"process_priority", "whisper_gpu",
	} {
		if _, ok := keys[key]; ok {
			return true
		}
	}
	return false
}
