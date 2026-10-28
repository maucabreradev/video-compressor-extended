package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerHelperProcess(t *testing.T) {
	if os.Getenv("VCX_PROCESS_HELPER") != "1" {
		return
	}

	if len(os.Args) >= 4 {
		_, _ = fmt.Fprintln(os.Stdout, os.Args[len(os.Args)-2])
		_, _ = fmt.Fprintln(os.Stdout, os.Args[len(os.Args)-1])
	}
	if byteCount, err := strconv.Atoi(os.Getenv("VCX_PROCESS_HELPER_BYTES")); err == nil && byteCount > 0 {
		payload := strings.Repeat("x", byteCount)
		_, _ = ioWriteString(os.Stdout, payload)
		_, _ = ioWriteString(os.Stderr, payload)
	}
	if delay, err := strconv.Atoi(os.Getenv("VCX_PROCESS_HELPER_SLEEP_MS")); err == nil && delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if exitCode, err := strconv.Atoi(os.Getenv("VCX_PROCESS_HELPER_EXIT_CODE")); err == nil {
		os.Exit(exitCode)
	}
	os.Exit(0)
}

func TestRunnerPassesArgumentsWithoutShellParsing(t *testing.T) {
	mediaPath := filepath.Join("media café", "video with spaces.mp4")
	command := helperCommand(mediaPath, "café")
	command.Dir = filepath.Join(t.TempDir(), "working directory café")
	if err := os.MkdirAll(command.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := (Runner{}).Execute(context.Background(), command, nil)
	if err != nil {
		t.Fatalf("run helper process: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if got, want := string(result.Stdout), mediaPath+"\ncafé\n"; got != want {
		t.Fatalf("arguments were not preserved: got %q, want %q", got, want)
	}
}

func TestRunnerReturnsNonZeroExitCodeAndDrainsBothStreams(t *testing.T) {
	command := helperCommand()
	command.Env = append(command.Env, "VCX_PROCESS_HELPER_BYTES=1048576", "VCX_PROCESS_HELPER_EXIT_CODE=23")
	runner := Runner{MaxCaptureBytes: 1024}
	var streamed atomic.Int64
	result, err := runner.Execute(context.Background(), command, func(chunk OutputChunk) {
		streamed.Add(int64(len(chunk.Data)))
	})
	if err == nil {
		t.Fatal("expected the child process error to be returned")
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 23 {
		t.Fatalf("expected child exit code 23, got result=%+v err=%v", result, err)
	}
	if result.ExitCode != 23 {
		t.Fatalf("expected result exit code 23, got %d", result.ExitCode)
	}
	if len(result.Stdout) > 1024 || len(result.Stderr) > 1024 || !result.StdoutTruncated || !result.StderrTruncated {
		t.Fatalf("captured logs were not bounded: stdout=%d stderr=%d result=%+v", len(result.Stdout), len(result.Stderr), result)
	}
	if streamed.Load() < 2*1024*1024 {
		t.Fatalf("expected all output to be streamed, got %d bytes", streamed.Load())
	}
}

func TestRunnerStopsChildWhenContextIsCancelled(t *testing.T) {
	command := helperCommand()
	command.Env = append(command.Env, "VCX_PROCESS_HELPER_SLEEP_MS=10000")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := (Runner{}).Execute(ctx, command, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline error, got %v", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("cancelled process took too long to stop: %s", elapsed)
	}
}

func helperCommand(args ...string) Command {
	commandArgs := append([]string{"-test.run=^TestRunnerHelperProcess$"}, args...)
	return Command{
		Executable: os.Args[0],
		Args:       commandArgs,
		Env: append(os.Environ(),
			"VCX_PROCESS_HELPER=1",
			"VCX_PROCESS_HELPER_EXIT_CODE=0",
		),
	}
}

func ioWriteString(writer *os.File, value string) (int, error) {
	return writer.WriteString(value)
}
