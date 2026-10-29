//go:build windows

package subtitle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

func TestTranscribeUsesRelativeWhisperAndModelPathsForUnicodeCache(t *testing.T) {
	runtimeDirectory := filepath.Join(t.TempDir(), "Whisper runtime é")
	if err := os.MkdirAll(runtimeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	whisperPath := filepath.Join(runtimeDirectory, "whisper-cli.exe")
	modelPath := filepath.Join(runtimeDirectory, "ggml-base.bin")
	for _, path := range []string{whisperPath, modelPath} {
		if err := os.WriteFile(path, []byte(filepath.Base(path)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	inputPath := filepath.Join(t.TempDir(), "speech input é.wav")
	outputPrefix := filepath.Join(t.TempDir(), "transcript é")
	runner := &captureWhisperRunner{t: t, modelPath: modelPath}
	if err := transcribe(context.Background(), runner, Options{
		WhisperPath: whisperPath, ModelPath: modelPath, Language: "en", Threads: 1,
	}, inputPath, outputPrefix); err != nil {
		t.Fatalf("transcribe through Unicode cache path: %v", err)
	}

	command := runner.command
	if command.Dir != "" || hasNonASCIIPathCharacter(command.Executable) || !filepath.IsAbs(command.Executable) {
		t.Fatalf("Unicode Whisper executable path was not converted to an ASCII short path: executable=%q dir=%q", command.Executable, command.Dir)
	}
	originalExecutable, err := os.Stat(whisperPath)
	if err != nil {
		t.Fatal(err)
	}
	shortExecutable, err := os.Stat(command.Executable)
	if err != nil || !os.SameFile(originalExecutable, shortExecutable) {
		t.Fatalf("Whisper short path does not refer to the selected executable: %v", err)
	}
	modelArgument := argumentAfter(command.Args, "-m")
	if modelArgument == "" || hasNonASCIIPathCharacter(modelArgument) || modelArgument == modelPath {
		t.Fatalf("model argument must use a distinct ASCII short path, got %q", modelArgument)
	}
	if !strings.Contains(strings.Join(command.Args, " "), inputPath) || !strings.Contains(strings.Join(command.Args, " "), outputPrefix) {
		t.Fatalf("Unicode audio or output paths were not preserved: %#v", command.Args)
	}
}

type captureWhisperRunner struct {
	t         *testing.T
	modelPath string
	command   process.Command
}

func (runner *captureWhisperRunner) Execute(_ context.Context, command process.Command, _ func(process.OutputChunk)) (process.Result, error) {
	runner.command = command
	modelArgument := argumentAfter(command.Args, "-m")
	aliasedModel := filepath.Join(command.Dir, modelArgument)
	if _, err := os.Stat(aliasedModel); err != nil {
		runner.t.Errorf("model alias is unavailable to Whisper: %v", err)
	} else {
		original, originalErr := os.Stat(runner.modelPath)
		alias, aliasErr := os.Stat(aliasedModel)
		if originalErr != nil || aliasErr != nil || !os.SameFile(original, alias) {
			runner.t.Errorf("model alias does not refer to the selected model: original=%v alias=%v", originalErr, aliasErr)
		}
	}
	return process.Result{ExitCode: 0}, nil
}

func argumentAfter(arguments []string, option string) string {
	for index, argument := range arguments {
		if argument == option && index+1 < len(arguments) {
			return arguments[index+1]
		}
	}
	return ""
}

func hasNonASCIIPathCharacter(path string) bool {
	for _, character := range path {
		if character > 0x7f {
			return true
		}
	}
	return false
}
