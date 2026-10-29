package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/cli"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/diagnostics"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

func TestVersionFlagPrintsDevelopmentVersion(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, "--version")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if stdout != "VCX v2.0.0-dev\n" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
}

func TestHelpFlagPrintsUsageAndSupportedFlags(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, "--help")

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	for _, expected := range []string{"Usage:", "vcx [flags]", "--help", "--version"} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("help output does not contain %q: %s", expected, stdout)
		}
	}
}

func TestApplicationHandlersRegisterInteractiveTUI(t *testing.T) {
	application := app.New(process.Runner{}, io.Discard)
	if handlers := applicationHandlers(application); handlers.TUI == nil {
		t.Fatal("interactive TUI handler is not registered")
	}
}

func TestUnknownOptionReturnsUsageError(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "--unknown-option")

	if exitCode == 0 {
		t.Fatal("expected a non-zero exit code for an unknown option")
	}
	if !strings.Contains(stderr, "unknown") && !strings.Contains(stderr, "not defined") {
		t.Fatalf("expected an actionable unknown-option error, got: %s", stderr)
	}
}

func TestDoctorExitCodeMatchesWhisperModelReadiness(t *testing.T) {
	tests := []struct {
		name       string
		modelPath  string
		createFile bool
		wantExit   int
		wantStatus string
	}{
		{name: "unconfigured", wantExit: 1, wantStatus: "NOT CONFIGURED"},
		{name: "missing", modelPath: "missing-model.bin", wantExit: 1, wantStatus: "MISSING"},
		{name: "configured", modelPath: "model.bin", createFile: true, wantExit: 0, wantStatus: "OK"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			modelPath := test.modelPath
			if modelPath != "" {
				modelPath = filepath.Join(directory, modelPath)
			}
			if test.createFile {
				if err := os.WriteFile(modelPath, []byte("whisper model"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			configPath := filepath.Join(directory, "vcx.toml")
			contents := ""
			if modelPath != "" {
				contents = fmt.Sprintf("whisper_model_path = %q\n", modelPath)
			}
			if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			doctor := func(ctx context.Context, cfg config.Config) error {
				report := (diagnostics.Checker{
					Runner: doctorRunner{}, Platform: "windows", TempDir: directory,
				}).Diagnose(ctx, diagnostics.Options{
					VideoCodec: cfg.VideoCodec, WhisperModelPath: cfg.WhisperModelPath,
				})
				if err := report.WriteReport(&stdout); err != nil {
					return err
				}
				if report.HasFailures() {
					return errors.New("diagnostic readiness is incomplete")
				}
				return nil
			}
			gotExit := executeCLI([]string{"--config", configPath, "doctor"}, cli.Handlers{Doctor: doctor}, &stdout, &stderr)
			if gotExit != test.wantExit {
				t.Fatalf("doctor exit code = %d, want %d; stdout=%q stderr=%q", gotExit, test.wantExit, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), test.wantStatus) {
				t.Fatalf("doctor report does not include status %q: %q", test.wantStatus, stdout.String())
			}
		})
	}
}

type doctorRunner struct{}

func (doctorRunner) LookPath(name string) (string, error) { return name, nil }

func (doctorRunner) Run(_ context.Context, _ string, args ...string) ([]byte, []byte, error) {
	for _, arg := range args {
		if arg == "-encoders" {
			return []byte(" V..... h264_amf AMD H.264 Encoder\n"), nil, nil
		}
	}
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("missing output path")
	}
	if err := os.WriteFile(args[len(args)-1], []byte("temporary encoder probe"), 0o600); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	commandArgs := append([]string{"run", "."}, args...)
	command := exec.Command("go", commandArgs...)
	command.Stdout = new(bytes.Buffer)
	command.Stderr = new(bytes.Buffer)
	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			t.Fatalf("could not run VCX CLI: %v", err)
		}
	}

	stdout = command.Stdout.(*bytes.Buffer).String()
	stderr = command.Stderr.(*bytes.Buffer).String()
	return stdout, stderr, exitCode
}
