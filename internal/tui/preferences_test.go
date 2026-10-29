package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
)

func TestPreferencesPersistOnlyNonSensitiveRunSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx", "preferences.json")
	cfg := config.Defaults()
	cfg.InputRoot = `C:\Users\Example\Private Media`
	cfg.OutputDir = `C:\Users\Example\Private Output`
	cfg.WhisperModelPath = `C:\Users\Example\Models\secret-model.bin`
	cfg.AllowSoftwareEncoding = true
	cfg.VideoCodec = "hevc"
	cfg.Language = "es"
	cfg.MaxWidth = 1920
	cfg.MaxHeight = 1080
	cfg.SubtitlesEnabled = false
	cfg.ResourceProfile = config.ResourceProfileCustom
	cfg.EncodeWorkers = 2

	if err := savePreferences(path, cfg); err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	if err := savePreferences(path, cfg); err != nil {
		t.Fatalf("replace saved preferences: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved preferences: %v", err)
	}
	for _, sensitive := range []string{cfg.InputRoot, cfg.OutputDir, cfg.WhisperModelPath, "allow_software_encoding"} {
		if strings.Contains(string(contents), sensitive) {
			t.Errorf("saved preferences contain sensitive or consent state %q: %s", sensitive, contents)
		}
	}

	loaded, err := loadPreferences(path)
	if err != nil {
		t.Fatalf("load preferences: %v", err)
	}
	got := applyPreferences(config.Defaults(), loaded)
	if got.VideoCodec != "hevc" || got.Language != "es" || got.MaxWidth != 1920 || got.MaxHeight != 1080 ||
		got.SubtitlesEnabled || got.ResourceProfile != config.ResourceProfileCustom || got.EncodeWorkers != 2 {
		t.Fatalf("saved settings were not restored: %+v", got)
	}
	if got.InputRoot != config.Defaults().InputRoot || got.OutputDir != config.Defaults().OutputDir ||
		got.WhisperModelPath != "" || got.AllowSoftwareEncoding {
		t.Fatalf("preferences restored paths or software-consent state: %+v", got)
	}
}

func TestPreferencesDoNotOverrideExplicitConfigurationAndCanBeReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	preferred := config.Defaults()
	preferred.VideoCodec = "hevc"
	if err := savePreferences(path, preferred); err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	loaded, err := loadPreferences(path)
	if err != nil {
		t.Fatalf("load preferences: %v", err)
	}

	explicit := config.Defaults()
	explicit.VideoCodec = "h264"
	explicit.MaxWidth = 640
	got := applyPreferences(explicit, loaded)
	if got.VideoCodec != "h264" || got.MaxWidth != 640 {
		t.Fatalf("preferences overrode explicit configuration: codec=%q width=%d", got.VideoCodec, got.MaxWidth)
	}
	if err := resetPreferences(path); err != nil {
		t.Fatalf("reset preferences: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("preferences still exist after reset: %v", err)
	}
}
