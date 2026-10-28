package subtitle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

const validTranscript = "1\n00:00:00,000 --> 00:00:01,250\nHello from the original audio.\n"

func TestGenerateTranscribesOriginalAndPublishesValidatedSRT(t *testing.T) {
	plan, modelPath, sourceBytes := createSubtitleFixture(t)
	runner := &fakeRunner{transcript: validTranscript}

	err := Generate(context.Background(), runner, plan, Options{
		FFmpegPath: "ffmpeg", WhisperPath: "whisper-cli", ModelPath: modelPath,
		Language: "es", Threads: 3,
	})
	if err != nil {
		t.Fatalf("generate subtitles: %v", err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected FFmpeg extraction and one whisper invocation, got %#v", runner.commands)
	}
	if runner.commands[0].Executable != "ffmpeg" ||
		!containsPair(runner.commands[0].Args, "-i", plan.Source) ||
		!containsPair(runner.commands[0].Args, "-ac", "1") ||
		!containsPair(runner.commands[0].Args, "-ar", "16000") {
		t.Fatalf("audio extraction did not use the original media: %#v", runner.commands[0])
	}
	whisper := runner.commands[1]
	if whisper.Executable != "whisper-cli" || !containsPair(whisper.Args, "-m", modelPath) ||
		!containsPair(whisper.Args, "-l", "es") || !containsPair(whisper.Args, "-t", "3") ||
		!containsArg(whisper.Args, "-osrt") {
		t.Fatalf("unexpected whisper.cpp invocation: %#v", whisper)
	}
	if !containsPair(whisper.Args, "-f", runner.commands[0].Args[len(runner.commands[0].Args)-1]) {
		t.Fatalf("whisper did not receive the extracted audio: %#v", whisper.Args)
	}
	outputPrefix, ok := argValue(whisper.Args, "-of")
	if !ok || filepath.Dir(filepath.Dir(outputPrefix)) != filepath.Dir(plan.Subtitle) {
		t.Fatalf("temporary SRT was not created beside its final destination: %q", outputPrefix)
	}
	if contents, err := os.ReadFile(plan.Subtitle); err != nil || string(contents) != validTranscript {
		t.Fatalf("published subtitle mismatch: contents=%q err=%v", contents, err)
	}
	if contents, err := os.ReadFile(plan.Source); err != nil || string(contents) != sourceBytes {
		t.Fatalf("original input changed: contents=%q err=%v", contents, err)
	}
	entries, err := os.ReadDir(filepath.Dir(plan.Subtitle))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(plan.Subtitle) {
		t.Fatalf("temporary audio or SRT was not cleaned: %#v", entries)
	}
}

func TestValidateOutputRequiresARegularNonEmptyValidSRT(t *testing.T) {
	path := filepath.Join(t.TempDir(), "captions.srt")
	if err := os.WriteFile(path, []byte(validTranscript), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOutput(path); err != nil {
		t.Fatalf("validate completed SRT: %v", err)
	}
	if err := os.WriteFile(path, []byte("invalid SRT"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOutput(path); err == nil {
		t.Fatal("expected malformed SRT to be invalid")
	}
	if err := ValidateOutput(filepath.Join(t.TempDir(), "missing.srt")); err == nil {
		t.Fatal("expected missing SRT to be invalid")
	}
}

func TestGenerateRejectsMissingModelBeforeStartingProcesses(t *testing.T) {
	plan, _, _ := createSubtitleFixture(t)
	runner := &fakeRunner{transcript: validTranscript}
	options := subtitleOptions()
	options.ModelPath = filepath.Join(t.TempDir(), "missing-model.bin")

	err := Generate(context.Background(), runner, plan, options)
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("expected actionable missing-model error, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("external processes ran without a model: %#v", runner.commands)
	}
}

func TestGenerateRequiresWhisperExecutable(t *testing.T) {
	plan, modelPath, _ := createSubtitleFixture(t)
	runner := &fakeRunner{transcript: validTranscript}
	options := subtitleOptions()
	options.ModelPath = modelPath
	options.WhisperPath = " "

	err := Generate(context.Background(), runner, plan, options)
	if err == nil || !strings.Contains(err.Error(), "whisper") {
		t.Fatalf("expected actionable missing-executable error, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("external processes ran without whisper.cpp: %#v", runner.commands)
	}
}

func TestGenerateDoesNotPublishInvalidSRT(t *testing.T) {
	plan, modelPath, _ := createSubtitleFixture(t)
	runner := &fakeRunner{transcript: "not a subtitle file\n"}
	options := subtitleOptions()
	options.ModelPath = modelPath

	err := Generate(context.Background(), runner, plan, options)
	if err == nil || !strings.Contains(err.Error(), "validate") {
		t.Fatalf("expected invalid SRT validation error, got %v", err)
	}
	if _, err := os.Stat(plan.Subtitle); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid subtitle was published: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(plan.Subtitle))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("invalid SRT or extracted audio was not cleaned: %#v", entries)
	}
}

func TestGeneratePreservesExistingSubtitle(t *testing.T) {
	plan, modelPath, _ := createSubtitleFixture(t)
	if err := os.MkdirAll(filepath.Dir(plan.Subtitle), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.Subtitle, []byte("existing subtitles"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{transcript: validTranscript}
	options := subtitleOptions()
	options.ModelPath = modelPath

	if err := Generate(context.Background(), runner, plan, options); err == nil {
		t.Fatal("expected an existing subtitle destination to block generation")
	}
	if contents, err := os.ReadFile(plan.Subtitle); err != nil || string(contents) != "existing subtitles" {
		t.Fatalf("existing subtitle was changed: contents=%q err=%v", contents, err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("external processes ran despite an existing output: %#v", runner.commands)
	}
}

func TestGenerateReportsProcessFailuresAndCleansTemporaryFiles(t *testing.T) {
	tests := []struct {
		name       string
		ffmpegErr  error
		whisperErr error
		want       string
	}{
		{name: "audio extraction", ffmpegErr: errors.New("decoder failed"), want: "extract audio"},
		{name: "transcription", whisperErr: errors.New("model load failed"), want: "transcribe"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, modelPath, _ := createSubtitleFixture(t)
			runner := &fakeRunner{
				transcript: validTranscript, ffmpegErr: test.ffmpegErr, whisperErr: test.whisperErr,
			}
			options := subtitleOptions()
			options.ModelPath = modelPath

			err := Generate(context.Background(), runner, plan, options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q failure, got %v", test.want, err)
			}
			if _, err := os.Stat(plan.Subtitle); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed subtitle was published: %v", err)
			}
			entries, err := os.ReadDir(filepath.Dir(plan.Subtitle))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("temporary files were not cleaned after failure: %#v", entries)
			}
		})
	}
}

func TestGenerateCancellationCleansTemporaryAudioAndPreservesOriginal(t *testing.T) {
	plan, modelPath, sourceBytes := createSubtitleFixture(t)
	runner := &blockingAudioRunner{started: make(chan string, 1)}
	options := subtitleOptions()
	options.ModelPath = modelPath
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Generate(ctx, runner, plan, options) }()

	var audioPath string
	select {
	case audioPath = <-runner.started:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("audio extraction did not start")
	}
	cancel()
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("subtitle generation error = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(audioPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial extracted audio remains after cancellation: %v", err)
	}
	if _, err := os.Stat(plan.Subtitle); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled subtitle was published: %v", err)
	}
	if contents, err := os.ReadFile(plan.Source); err != nil || string(contents) != sourceBytes {
		t.Fatalf("original media changed after cancellation: contents=%q err=%v", contents, err)
	}
	entries, err := os.ReadDir(filepath.Dir(plan.Subtitle))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary subtitle directory remains after cancellation: %#v", entries)
	}
}

type fakeRunner struct {
	commands   []process.Command
	transcript string
	ffmpegErr  error
	whisperErr error
}

type blockingAudioRunner struct {
	started chan string
}

func (runner *blockingAudioRunner) Execute(ctx context.Context, command process.Command, _ func(process.OutputChunk)) (process.Result, error) {
	if command.Executable != "ffmpeg" {
		return process.Result{ExitCode: -1}, os.ErrNotExist
	}
	audioPath := command.Args[len(command.Args)-1]
	if err := os.WriteFile(audioPath, []byte("partial audio"), 0o600); err != nil {
		return process.Result{ExitCode: 1}, err
	}
	runner.started <- audioPath
	<-ctx.Done()
	return process.Result{ExitCode: -1}, ctx.Err()
}

func (runner *fakeRunner) Execute(_ context.Context, command process.Command, _ func(process.OutputChunk)) (process.Result, error) {
	runner.commands = append(runner.commands, command)
	switch command.Executable {
	case "ffmpeg":
		if runner.ffmpegErr != nil {
			return process.Result{ExitCode: 1, Stderr: []byte(runner.ffmpegErr.Error())}, runner.ffmpegErr
		}
		if err := os.WriteFile(command.Args[len(command.Args)-1], []byte("temporary wave audio"), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		return process.Result{ExitCode: 0}, nil
	case "whisper-cli":
		if runner.whisperErr != nil {
			return process.Result{ExitCode: 1, Stderr: []byte(runner.whisperErr.Error())}, runner.whisperErr
		}
		prefix, ok := argValue(command.Args, "-of")
		if !ok {
			return process.Result{ExitCode: 1}, errors.New("whisper output prefix is missing")
		}
		if err := os.WriteFile(prefix+".srt", []byte(runner.transcript), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		return process.Result{ExitCode: 0}, nil
	default:
		return process.Result{ExitCode: -1}, errors.New("unexpected executable")
	}
}

func createSubtitleFixture(t *testing.T) (outputplan.OutputPlan, string, string) {
	t.Helper()
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input café")
	source := filepath.Join(inputRoot, "clip source.mkv")
	sourceBytes := "original source video"
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte(sourceBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, "model.bin")
	if err := os.WriteFile(modelPath, []byte("whisper model"), 0o600); err != nil {
		t.Fatal(err)
	}
	plans, err := outputplan.PlanOutputs(inputRoot, filepath.Join(root, "output"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	return plans[0], modelPath, sourceBytes
}

func subtitleOptions() Options {
	return Options{
		FFmpegPath: "ffmpeg", WhisperPath: "whisper-cli", Language: "en", Threads: 1,
	}
}

func containsPair(args []string, key, value string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key && args[index+1] == value {
			return true
		}
	}
	return false
}

func containsArg(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func argValue(args []string, key string) (string, bool) {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key {
			return args[index+1], true
		}
	}
	return "", false
}
