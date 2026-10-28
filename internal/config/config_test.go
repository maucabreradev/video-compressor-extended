package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsProvidesConservativeSettings(t *testing.T) {
	got := Defaults()

	if got.InputRoot != "." {
		t.Errorf("expected default input root ., got %q", got.InputRoot)
	}
	if got.OutputDir != "vcx_processed" {
		t.Errorf("expected default output directory vcx_processed, got %q", got.OutputDir)
	}
	if got.EncodeWorkers != 1 {
		t.Errorf("expected one encode worker, got %d", got.EncodeWorkers)
	}
	if got.SubtitleWorkers != 1 {
		t.Errorf("expected one subtitle worker, got %d", got.SubtitleWorkers)
	}
	if got.WhisperThreads != 1 {
		t.Errorf("expected one Whisper thread, got %d", got.WhisperThreads)
	}
	if got.Language != "en" {
		t.Errorf("expected default language en, got %q", got.Language)
	}
	if got.VideoCodec != "h264" {
		t.Errorf("expected default video codec h264, got %q", got.VideoCodec)
	}
	if got.MaxWidth != 1280 || got.MaxHeight != 720 {
		t.Errorf("expected default maximum resolution 1280x720, got %dx%d", got.MaxWidth, got.MaxHeight)
	}
	if got.FrameRate != 0 {
		t.Errorf("expected default frame rate 0 (preserve source), got %d", got.FrameRate)
	}
	if got.WhisperModelPath != "" {
		t.Errorf("expected no default Whisper model path, got %q", got.WhisperModelPath)
	}
}

func TestLoadOverlaysPartialTOMLOnDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx.toml")
	contents := "output_dir = \"rendered\"\nencode_workers = 3\nlanguage = \"es\"\nvideo_codec = \"hevc\"\nwhisper_model_path = \"models/ggml-small.bin\"\nmax_width = 1920\nmax_height = 1080\nframe_rate = 30\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.OutputDir != "rendered" || got.EncodeWorkers != 3 || got.Language != "es" {
		t.Fatalf("TOML values were not applied: %+v", got)
	}
	if got.InputRoot != "." || got.SubtitleWorkers != 1 || got.WhisperThreads != 1 {
		t.Fatalf("omitted TOML values did not retain defaults: %+v", got)
	}
	if got.VideoCodec != "hevc" || got.WhisperModelPath != "models/ggml-small.bin" {
		t.Fatalf("diagnostic settings were not loaded: %+v", got)
	}
	if got.MaxWidth != 1920 || got.MaxHeight != 1080 || got.FrameRate != 30 {
		t.Fatalf("compression settings were not loaded: %+v", got)
	}
}

func TestLoadRejectsUnknownTOMLKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx.toml")
	if err := os.WriteFile(path, []byte("encdoe_workers = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for an unknown configuration key")
	}
}

func TestValidateRejectsInvalidLimits(t *testing.T) {
	tests := []struct {
		name   string
		update func(*Config)
	}{
		{name: "empty output directory", update: func(c *Config) { c.OutputDir = " " }},
		{name: "zero encode workers", update: func(c *Config) { c.EncodeWorkers = 0 }},
		{name: "negative subtitle workers", update: func(c *Config) { c.SubtitleWorkers = -1 }},
		{name: "zero Whisper threads", update: func(c *Config) { c.WhisperThreads = 0 }},
		{name: "empty language", update: func(c *Config) { c.Language = " " }},
		{name: "unsupported video codec", update: func(c *Config) { c.VideoCodec = "av1" }},
		{name: "too-small maximum width", update: func(c *Config) { c.MaxWidth = 1 }},
		{name: "too-small maximum height", update: func(c *Config) { c.MaxHeight = 1 }},
		{name: "unsupported frame rate", update: func(c *Config) { c.FrameRate = 241 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := Defaults()
			test.update(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}
