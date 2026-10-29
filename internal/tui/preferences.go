package tui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
)

const preferencesVersion = 1

// savedPreferences intentionally excludes paths, secrets, and software-encoder
// consent. The latter must be requested again for every run.
type savedPreferences struct {
	Version            int    `json:"version"`
	CompressionEnabled bool   `json:"compression_enabled"`
	SubtitlesEnabled   bool   `json:"subtitles_enabled"`
	VideoCodec         string `json:"video_codec"`
	MaxWidth           int    `json:"max_width"`
	MaxHeight          int    `json:"max_height"`
	FrameRate          int    `json:"frame_rate"`
	Language           string `json:"language"`
	ResourceProfile    string `json:"resource_profile"`
	EncodeWorkers      int    `json:"encode_workers"`
	SubtitleWorkers    int    `json:"subtitle_workers"`
	WhisperThreads     int    `json:"whisper_threads"`
	ProcessPriority    string `json:"process_priority"`
	WhisperGPU         bool   `json:"whisper_gpu"`
}

func userPreferencesPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find per-user configuration directory: %w", err)
	}
	return filepath.Join(root, "vcx", "preferences.json"), nil
}

func loadPreferences(path string) (savedPreferences, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return savedPreferences{}, nil
	}
	if err != nil {
		return savedPreferences{}, fmt.Errorf("read preferences %q: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var preferences savedPreferences
	if err := decoder.Decode(&preferences); err != nil {
		return savedPreferences{}, fmt.Errorf("decode preferences %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return savedPreferences{}, fmt.Errorf("preferences %q contain multiple JSON values", path)
		}
		return savedPreferences{}, fmt.Errorf("read preferences trailer %q: %w", path, err)
	}
	if err := preferences.validate(); err != nil {
		return savedPreferences{}, fmt.Errorf("validate preferences %q: %w", path, err)
	}
	return preferences, nil
}

func (preferences savedPreferences) validate() error {
	if preferences.Version != preferencesVersion {
		return fmt.Errorf("unsupported preferences version %d", preferences.Version)
	}
	cfg := config.Defaults()
	preferences.apply(&cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}
	return nil
}

func (preferences savedPreferences) apply(cfg *config.Config) {
	cfg.CompressionEnabled = preferences.CompressionEnabled
	cfg.SubtitlesEnabled = preferences.SubtitlesEnabled
	cfg.VideoCodec = preferences.VideoCodec
	cfg.MaxWidth = preferences.MaxWidth
	cfg.MaxHeight = preferences.MaxHeight
	cfg.FrameRate = preferences.FrameRate
	cfg.Language = preferences.Language
	cfg.ResourceProfile = preferences.ResourceProfile
	cfg.EncodeWorkers = preferences.EncodeWorkers
	cfg.SubtitleWorkers = preferences.SubtitleWorkers
	cfg.WhisperThreads = preferences.WhisperThreads
	cfg.ProcessPriority = preferences.ProcessPriority
	cfg.WhisperGPU = preferences.WhisperGPU
}

func preferencesFromConfig(cfg config.Config) savedPreferences {
	return savedPreferences{
		Version:            preferencesVersion,
		CompressionEnabled: cfg.CompressionEnabled, SubtitlesEnabled: cfg.SubtitlesEnabled,
		VideoCodec: cfg.VideoCodec, MaxWidth: cfg.MaxWidth, MaxHeight: cfg.MaxHeight,
		FrameRate: cfg.FrameRate, Language: cfg.Language, ResourceProfile: cfg.ResourceProfile,
		EncodeWorkers: cfg.EncodeWorkers, SubtitleWorkers: cfg.SubtitleWorkers,
		WhisperThreads: cfg.WhisperThreads, ProcessPriority: cfg.ProcessPriority,
		WhisperGPU: cfg.WhisperGPU,
	}
}

func applyPreferences(cfg config.Config, preferences savedPreferences) config.Config {
	if preferences.Version != preferencesVersion || !usesDefaultRunSettings(cfg) {
		return cfg
	}
	preferences.apply(&cfg)
	return cfg
}

func usesDefaultRunSettings(cfg config.Config) bool {
	defaults := config.Defaults()
	return cfg.EncodeWorkers == defaults.EncodeWorkers &&
		cfg.SubtitleWorkers == defaults.SubtitleWorkers && cfg.WhisperThreads == defaults.WhisperThreads &&
		cfg.Language == defaults.Language && cfg.VideoCodec == defaults.VideoCodec &&
		cfg.CompressionEnabled == defaults.CompressionEnabled && cfg.SubtitlesEnabled == defaults.SubtitlesEnabled &&
		cfg.MaxWidth == defaults.MaxWidth && cfg.MaxHeight == defaults.MaxHeight && cfg.FrameRate == defaults.FrameRate &&
		cfg.ResourceProfile == defaults.ResourceProfile && cfg.ProcessPriority == defaults.ProcessPriority &&
		cfg.WhisperGPU == defaults.WhisperGPU
}

func savePreferences(path string, cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate preferences before saving: %w", err)
	}
	contents, err := json.MarshalIndent(preferencesFromConfig(cfg), "", "  ")
	if err != nil {
		return fmt.Errorf("encode preferences: %w", err)
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create preferences directory: %w", err)
	}
	temporary, err := os.CreateTemp(parent, ".vcx-preferences-*")
	if err != nil {
		return fmt.Errorf("create temporary preferences file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect temporary preferences file: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary preferences: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary preferences: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary preferences: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish preferences: %w", err)
	}
	return nil
}

func resetPreferences(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove saved preferences: %w", err)
	}
	return nil
}
