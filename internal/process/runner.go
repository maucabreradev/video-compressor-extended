package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

const (
	defaultCaptureLimit = 256 * 1024
	defaultReadSize     = 32 * 1024
)

// Stream identifies which child-process output stream produced a chunk.
type Stream string

const (
	Stdout Stream = "stdout"
	Stderr Stream = "stderr"
)

// Command describes a process invocation without shell parsing.
type Command struct {
	Executable string
	Args       []string
	Dir        string
	Env        []string
}

// OutputChunk is a copied chunk of process output. Callbacks are serialized
// across stdout and stderr and should return promptly.
type OutputChunk struct {
	Stream Stream
	Data   []byte
}

// Result contains the exit code and bounded tail captures of process output.
type Result struct {
	Stdout          []byte
	Stderr          []byte
	ExitCode        int
	StdoutTruncated bool
	StderrTruncated bool
}

// Runner launches external programs with argument arrays and drains both
// output streams concurrently. MaxCaptureBytes limits retained output per
// stream; zero selects a conservative default.
type Runner struct {
	MaxCaptureBytes int
	ReadBufferBytes int
}

// LookPath resolves an executable using the current process environment.
func (Runner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Run executes a command and returns bounded stdout and stderr captures.
func (runner Runner) Run(ctx context.Context, executable string, args ...string) ([]byte, []byte, error) {
	result, err := runner.Execute(ctx, Command{Executable: executable, Args: args}, nil)
	return result.Stdout, result.Stderr, err
}

// Execute starts a child process without invoking a shell. Output callbacks
// are serialized, while the underlying streams are always drained concurrently.
func (runner Runner) Execute(ctx context.Context, spec Command, onOutput func(OutputChunk)) (Result, error) {
	result := Result{ExitCode: -1}
	command := exec.CommandContext(ctx, spec.Executable, spec.Args...)
	command.Dir = spec.Dir
	if spec.Env != nil {
		command.Env = spec.Env
	}

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("open stdout pipe for %q: %w", spec.Executable, err)
	}
	stderrPipe, err := command.StderrPipe()
	if err != nil {
		return result, fmt.Errorf("open stderr pipe for %q: %w", spec.Executable, err)
	}
	if err := command.Start(); err != nil {
		return result, fmt.Errorf("start %q: %w", spec.Executable, err)
	}

	limit := runner.MaxCaptureBytes
	if limit <= 0 {
		limit = defaultCaptureLimit
	}
	readSize := runner.ReadBufferBytes
	if readSize <= 0 {
		readSize = defaultReadSize
	}
	stdoutCapture := tailBuffer{limit: limit}
	stderrCapture := tailBuffer{limit: limit}
	var callbackMu sync.Mutex
	var killOnce sync.Once
	var readers sync.WaitGroup
	readErrors := make(chan error, 2)
	readStream := func(stream Stream, source io.Reader, capture *tailBuffer) {
		defer readers.Done()
		buffer := make([]byte, readSize)
		for {
			count, readErr := source.Read(buffer)
			if count > 0 {
				chunk := append([]byte(nil), buffer[:count]...)
				capture.Write(chunk)
				if onOutput != nil {
					callbackMu.Lock()
					onOutput(OutputChunk{Stream: stream, Data: chunk})
					callbackMu.Unlock()
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					readErrors <- readErr
					killOnce.Do(func() { _ = command.Process.Kill() })
				}
				return
			}
		}
	}

	readers.Add(2)
	go readStream(Stdout, stdoutPipe, &stdoutCapture)
	go readStream(Stderr, stderrPipe, &stderrCapture)
	readers.Wait()
	waitErr := command.Wait()
	result.Stdout = stdoutCapture.data
	result.Stderr = stderrCapture.data
	result.StdoutTruncated = stdoutCapture.truncated
	result.StderrTruncated = stderrCapture.truncated
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}

	select {
	case readErr := <-readErrors:
		return result, fmt.Errorf("read output from %q: %w", spec.Executable, readErr)
	default:
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if waitErr != nil {
		return result, fmt.Errorf("run %q: %w", spec.Executable, waitErr)
	}
	return result, nil
}

type tailBuffer struct {
	limit     int
	data      []byte
	truncated bool
}

func (buffer *tailBuffer) Write(data []byte) {
	if len(data) >= buffer.limit {
		buffer.data = append(buffer.data[:0], data[len(data)-buffer.limit:]...)
		buffer.truncated = true
		return
	}
	if excess := len(buffer.data) + len(data) - buffer.limit; excess > 0 {
		copy(buffer.data, buffer.data[excess:])
		buffer.data = buffer.data[:len(buffer.data)-excess]
		buffer.truncated = true
	}
	buffer.data = append(buffer.data, data...)
}
