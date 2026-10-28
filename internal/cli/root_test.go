package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
)

func TestRootCommandExposesVersionAndHelp(t *testing.T) {
	root := NewRootCommand("2.0.0-dev", Handlers{})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute version: %v; stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "2.0.0-dev") {
		t.Fatalf("version output does not include the version: %q", stdout.String())
	}

	stdout.Reset()
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	for _, expected := range []string{"vcx", "run", "tui", "doctor"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Errorf("help output does not contain %q: %s", expected, stdout.String())
		}
	}
}

func TestRunCommandLoadsConfigAndAppliesCLIOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx.toml")
	if err := os.WriteFile(path, []byte("input_root = \"input-from-file\"\noutput_dir = \"from-file\"\nencode_workers = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var got config.Config
	root := NewRootCommand("2.0.0-dev", Handlers{
		Run: func(_ context.Context, cfg config.Config) error {
			got = cfg
			return nil
		},
	})
	root.SetArgs([]string{
		"--config", path,
		"--input", "input-from-cli",
		"--output-dir", "from-cli",
		"--encode-workers", "4",
		"--max-width", "1920",
		"--max-height", "1080",
		"--frame-rate", "30",
		"run",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute run: %v", err)
	}
	if got.InputRoot != "input-from-cli" || got.OutputDir != "from-cli" || got.EncodeWorkers != 4 {
		t.Fatalf("CLI values did not override TOML: %+v", got)
	}
	if got.ResourceProfile != config.ResourceProfileCustom {
		t.Fatalf("explicit worker overrides should select custom profile, got %q", got.ResourceProfile)
	}
	if got.SubtitleWorkers != 1 || got.WhisperThreads != 1 {
		t.Fatalf("defaults were not preserved: %+v", got)
	}
	if got.MaxWidth != 1920 || got.MaxHeight != 1080 || got.FrameRate != 30 {
		t.Fatalf("CLI compression settings were not applied: %+v", got)
	}
}

func TestRunCommandRejectsInvalidConfigBeforeDispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx.toml")
	if err := os.WriteFile(path, []byte("encode_workers = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	dispatched := false
	root := NewRootCommand("2.0.0-dev", Handlers{
		Run: func(context.Context, config.Config) error {
			dispatched = true
			return nil
		},
	})
	root.SetArgs([]string{"--config", path, "run"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected invalid config to fail before dispatch")
	}
	if dispatched {
		t.Fatal("run handler was called with invalid configuration")
	}
}

func TestCLIOverrideIsAppliedBeforeConfigValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx.toml")
	if err := os.WriteFile(path, []byte("encode_workers = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var got config.Config
	root := NewRootCommand("2.0.0-dev", Handlers{
		Run: func(_ context.Context, cfg config.Config) error {
			got = cfg
			return nil
		},
	})
	root.SetArgs([]string{"--config", path, "--encode-workers", "4", "run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("CLI override should repair the invalid file value: %v", err)
	}
	if got.EncodeWorkers != 4 {
		t.Fatalf("expected CLI worker limit 4, got %d", got.EncodeWorkers)
	}
}

func TestRunCommandAppliesCustomResourceProfileAndAccelerationOptions(t *testing.T) {
	var got config.Config
	root := NewRootCommand("2.0.0-dev", Handlers{
		Run: func(_ context.Context, cfg config.Config) error {
			got = cfg
			return nil
		},
	})
	root.SetArgs([]string{
		"--resource-profile", "custom",
		"--encode-workers", "2",
		"--subtitle-workers", "3",
		"--whisper-threads", "4",
		"--process-priority", "normal",
		"--whisper-gpu",
		"run",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute run: %v", err)
	}
	resources := got.EffectiveResources()
	if resources.EncodeWorkers != 2 || resources.SubtitleWorkers != 3 || resources.WhisperThreads != 4 {
		t.Fatalf("CLI limits were not independently applied: %+v", resources)
	}
	if resources.ProcessPriority != config.ProcessPriorityNormal || !resources.WhisperGPU {
		t.Fatalf("CLI priority or Whisper GPU option was not applied: %+v", resources)
	}
}

func TestCommandsDispatchToTheirHandlers(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{name: "tui", command: "tui"},
		{name: "run", command: "run"},
		{name: "doctor", command: "doctor"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := ""
			handlers := Handlers{
				TUI: func(context.Context, config.Config) error { called = "tui"; return nil },
				Run: func(context.Context, config.Config) error { called = "run"; return nil },
				Doctor: func(context.Context, config.Config) error {
					called = "doctor"
					return nil
				},
			}
			root := NewRootCommand("2.0.0-dev", handlers)
			root.SetArgs([]string{test.command})
			if err := root.Execute(); err != nil {
				t.Fatalf("execute %s: %v", test.command, err)
			}
			if called != test.command {
				t.Fatalf("expected %s handler, got %q", test.command, called)
			}
		})
	}
}

func TestDoctorReceivesModelAndCodecOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vcx.toml")
	contents := "video_codec = \"h264\"\nwhisper_model_path = \"model-from-file.bin\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	var got config.Config
	root := NewRootCommand("2.0.0-dev", Handlers{
		Doctor: func(_ context.Context, cfg config.Config) error {
			got = cfg
			return nil
		},
	})
	root.SetArgs([]string{
		"--config", path,
		"--video-codec", "hevc",
		"--whisper-model-path", "model-from-cli.bin",
		"doctor",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
	if got.VideoCodec != "hevc" || got.WhisperModelPath != "model-from-cli.bin" {
		t.Fatalf("diagnostic CLI flags did not override TOML: %+v", got)
	}
}
