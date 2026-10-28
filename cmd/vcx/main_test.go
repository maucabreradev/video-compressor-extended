package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
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

func TestUnknownOptionReturnsUsageError(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "--unknown-option")

	if exitCode == 0 {
		t.Fatal("expected a non-zero exit code for an unknown option")
	}
	if !strings.Contains(stderr, "unknown") && !strings.Contains(stderr, "not defined") {
		t.Fatalf("expected an actionable unknown-option error, got: %s", stderr)
	}
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
