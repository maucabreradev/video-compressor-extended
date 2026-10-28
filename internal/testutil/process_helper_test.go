package testutil

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestProcessHelperEmitsStreamsAndUsesRequestedExitCode(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestProcessHelperEntryPoint$")
	command.Env = append(os.Environ(),
		"VCX_TEST_PROCESS_HELPER=1",
		"VCX_TEST_HELPER_STDOUT=child stdout",
		"VCX_TEST_HELPER_STDERR=child stderr",
		"VCX_TEST_HELPER_EXIT_CODE=17",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()

	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("expected child exit code 17, got error %v", err)
	}
	if exitError.ExitCode() != 17 {
		t.Fatalf("expected child exit code 17, got %d", exitError.ExitCode())
	}
	if stdout.String() != "child stdout\n" {
		t.Fatalf("unexpected child stdout: %q", stdout.String())
	}
	if stderr.String() != "child stderr\n" {
		t.Fatalf("unexpected child stderr: %q", stderr.String())
	}
}

func TestProcessHelperEntryPoint(t *testing.T) {
	if os.Getenv("VCX_TEST_PROCESS_HELPER") != "1" {
		return
	}

	_, _ = fmt.Fprintln(os.Stdout, os.Getenv("VCX_TEST_HELPER_STDOUT"))
	_, _ = fmt.Fprintln(os.Stderr, os.Getenv("VCX_TEST_HELPER_STDERR"))

	exitCode, err := strconv.Atoi(os.Getenv("VCX_TEST_HELPER_EXIT_CODE"))
	if err != nil {
		os.Exit(2)
	}
	os.Exit(exitCode)
}
