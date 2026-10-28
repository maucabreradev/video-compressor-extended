package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

func TestRunCompressesDiscoveredInputThroughVerifiedAMF(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "nested", "clip.mkv")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	delete(runner.paths, "whisper-cli")
	application := New(runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = "processed"

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression: %v", err)
	}
	encoded := filepath.Join(inputRoot, "processed", "nested", "clip.mp4")
	if contents, err := os.ReadFile(encoded); err != nil || string(contents) != "encoded output" {
		t.Fatalf("expected validated output at %q, contents=%q err=%v", encoded, contents, err)
	}
	if !runner.usedEncoder("h264_amf") {
		t.Fatalf("run did not use the verified AMF encoder: %#v", runner.commands)
	}
}

func TestRunEmitsStructuredLifecycleAndProgressEvents(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.progressEvents = 2
	var output bytes.Buffer
	application := New(runner, &output)
	application.platform = "windows"
	var events []RunEvent
	application.SetRunEventHandler(func(event RunEvent) error {
		events = append(events, event)
		return nil
	})
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("expected start, three progress, and completion events; got %#v", events)
	}
	if events[0].Kind != RunStarted || events[0].Source != source {
		t.Fatalf("unexpected start event: %+v", events[0])
	}
	if events[1].Kind != RunProgress || events[1].Source != source || !events[1].DurationKnown ||
		events[1].Progress.Frame != 1 || events[1].Progress.OutTime != time.Second || events[1].Progress.Percent != 10 {
		t.Fatalf("unexpected first progress event: %+v", events[1])
	}
	lastProgress := events[3]
	if lastProgress.Kind != RunProgress || !lastProgress.Progress.Done || lastProgress.Progress.Percent != 100 {
		t.Fatalf("unexpected final progress event: %+v", lastProgress)
	}
	if events[4].Kind != RunCompleted || events[4].Destination != filepath.Join(root, "processed", "clip.mp4") {
		t.Fatalf("unexpected completion event: %+v", events[4])
	}
	if output.Len() != 0 {
		t.Fatalf("application rendered batch text instead of emitting events: %q", output.String())
	}
}

func TestRunGeneratesSRTFromOriginalInput(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input café")
	source := filepath.Join(inputRoot, "clip original.mkv")
	modelPath := filepath.Join(root, "ggml-model.bin")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nHello from the source.\n"
	application := New(runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.WhisperModelPath = modelPath
	cfg.Language = "es"
	cfg.WhisperThreads = 4

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression and subtitles: %v", err)
	}
	plan, err := outputplan.PlanOutputs(inputRoot, cfg.OutputDir, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if contents, err := os.ReadFile(plan[0].Subtitle); err != nil || string(contents) != runner.whisperTranscript {
		t.Fatalf("expected validated SRT, contents=%q err=%v", contents, err)
	}
	if len(runner.audioInputs) != 1 || runner.audioInputs[0] != source {
		t.Fatalf("audio extraction did not use the original input: %#v", runner.audioInputs)
	}
	if !containsPair(runner.whisperArgs, "-m", modelPath) || !containsPair(runner.whisperArgs, "-l", "es") ||
		!containsPair(runner.whisperArgs, "-t", "4") {
		t.Fatalf("Whisper configuration was not passed to whisper.cpp: %#v", runner.whisperArgs)
	}
}

func TestRunPreservesSubtitleStageWhenCompressionFails(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "ggml-model.bin")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.encodeError = errors.New("encoder failed")
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nSubtitle survives.\n"
	application := New(runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.WhisperModelPath = modelPath

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "compress") {
		t.Fatalf("expected compression failure to be reported, got %v", err)
	}
	plan, err := outputplan.PlanOutputs(inputRoot, cfg.OutputDir, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if contents, err := os.ReadFile(plan[0].Subtitle); err != nil || string(contents) != runner.whisperTranscript {
		t.Fatalf("subtitle stage did not survive compression failure: contents=%q err=%v", contents, err)
	}
}

func TestRunKeepsVideoWhenSubtitlePrerequisitesAreMissing(t *testing.T) {
	tests := []struct {
		name           string
		whisperMissing bool
		modelMissing   bool
		wantError      string
	}{
		{name: "whisper executable", whisperMissing: true, wantError: "whisper-cli"},
		{name: "model file", modelMissing: true, wantError: "model"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			inputRoot := filepath.Join(root, "input")
			source := filepath.Join(inputRoot, "clip.mkv")
			if err := os.MkdirAll(inputRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
				t.Fatal(err)
			}

			runner := newFakeRunner()
			runner.paths["whisper-cli"] = "fake-whisper"
			modelPath := filepath.Join(root, "ggml-model.bin")
			if !test.modelMissing {
				if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.whisperMissing {
				delete(runner.paths, "whisper-cli")
			}
			application := New(runner, nil)
			application.platform = "windows"
			cfg := config.Defaults()
			cfg.InputRoot = inputRoot
			cfg.OutputDir = filepath.Join(root, "processed")
			cfg.WhisperModelPath = modelPath

			err := application.Run(context.Background(), cfg)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected subtitle prerequisite error containing %q, got %v", test.wantError, err)
			}
			videoOutput := filepath.Join(root, "processed", "clip.mp4")
			if contents, err := os.ReadFile(videoOutput); err != nil || string(contents) != "encoded output" {
				t.Fatalf("video output was lost after subtitle prerequisite failure: contents=%q err=%v", contents, err)
			}
		})
	}
}

func TestRunStopsWhenAMDEncoderProbeFailsWithoutCPUFallback(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mp4")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.encoderList = " V..... libx264 H.264 / AVC / MPEG-4 AVC\n"
	application := New(runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "compression prerequisites") {
		t.Fatalf("expected missing AMF capability to stop compression, got %v", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("compression ran despite missing AMF support: %#v", runner.commands)
	}
}

func TestRunDoesNotProbeHardwareWhenNoMediaIsFound(t *testing.T) {
	inputRoot := t.TempDir()
	runner := newFakeRunner()
	application := New(runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(t.TempDir(), "processed")

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "no supported video files") {
		t.Fatalf("expected a no-input error, got %v", err)
	}
	if runner.diagnosticCalls != 0 {
		t.Fatalf("dependency or GPU probes ran without work: %d calls", runner.diagnosticCalls)
	}
}

func TestDoctorFailsWhenWhisperModelIsNotConfigured(t *testing.T) {
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	var output bytes.Buffer
	application := New(runner, &output)
	application.platform = "windows"

	err := application.Doctor(context.Background(), config.Defaults())
	if err == nil {
		t.Fatalf("expected doctor to report incomplete subtitle readiness, report: %s", output.String())
	}
	if !strings.Contains(output.String(), "NOT CONFIGURED") || !strings.Contains(output.String(), "whisper_model_path") {
		t.Fatalf("doctor did not explain the unconfigured model: %q", output.String())
	}
}

func TestRunRejectsInvalidConfigurationBeforeDiagnostics(t *testing.T) {
	inputRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(inputRoot, "sample.mp4"), []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	application := New(runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.MaxWidth = 0

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "max_width") {
		t.Fatalf("expected invalid max_width to be rejected, got %v", err)
	}
	if runner.diagnosticCalls != 0 {
		t.Fatalf("diagnostics ran with invalid configuration: %d calls", runner.diagnosticCalls)
	}
}

type fakeRunner struct {
	paths             map[string]string
	encoderList       string
	commands          []process.Command
	encodeCalls       int
	diagnosticCalls   int
	progressEvents    int
	audioInputs       []string
	whisperArgs       []string
	whisperTranscript string
	whisperError      error
	encodeError       error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		paths:       map[string]string{"ffmpeg": "fake-ffmpeg", "ffprobe": "fake-ffprobe"},
		encoderList: " V..... h264_amf AMD H.264 Encoder\n",
	}
}

func (runner *fakeRunner) LookPath(name string) (string, error) {
	path, ok := runner.paths[name]
	if !ok {
		return "", errors.New("not found")
	}
	return path, nil
}

func (runner *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, []byte, error) {
	runner.diagnosticCalls++
	if containsArg(args, "format=duration") {
		return []byte("10.0\n"), nil, nil
	}
	if containsArg(args, "-encoders") {
		return []byte(runner.encoderList), nil, nil
	}
	if len(args) == 0 {
		return nil, nil, errors.New("probe output path is missing")
	}
	if err := os.WriteFile(args[len(args)-1], []byte("probe output"), 0o600); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func (runner *fakeRunner) Execute(_ context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	runner.commands = append(runner.commands, command)
	switch command.Executable {
	case "fake-ffmpeg":
		if containsPair(command.Args, "-ar", "16000") {
			for index, arg := range command.Args[:len(command.Args)-1] {
				if arg == "-i" && index+1 < len(command.Args) {
					runner.audioInputs = append(runner.audioInputs, command.Args[index+1])
					break
				}
			}
			if err := os.WriteFile(command.Args[len(command.Args)-1], []byte("temporary audio"), 0o600); err != nil {
				return process.Result{ExitCode: 1}, err
			}
			return process.Result{ExitCode: 0}, nil
		}
		runner.encodeCalls++
		if runner.encodeError != nil {
			return process.Result{ExitCode: 1}, runner.encodeError
		}
		if err := os.WriteFile(command.Args[len(command.Args)-1], []byte("encoded output"), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		if onOutput != nil {
			for frame := 1; frame <= runner.progressEvents; frame++ {
				data := []byte(fmt.Sprintf("frame=%d\nout_time_us=%d000000\nspeed=1.25x\nprogress=continue\n", frame, frame))
				onOutput(process.OutputChunk{Stream: process.Stdout, Data: data})
			}
			if runner.progressEvents > 0 {
				data := []byte(fmt.Sprintf("frame=%d\nout_time_us=%d000000\nspeed=1.25x\nprogress=end\n", runner.progressEvents, runner.progressEvents))
				onOutput(process.OutputChunk{Stream: process.Stdout, Data: data})
			}
		}
		return process.Result{ExitCode: 0}, nil
	case "fake-whisper":
		runner.whisperArgs = append([]string(nil), command.Args...)
		if runner.whisperError != nil {
			return process.Result{ExitCode: 1}, runner.whisperError
		}
		prefix, ok := argValue(command.Args, "-of")
		if !ok {
			return process.Result{ExitCode: 1}, errors.New("missing whisper output prefix")
		}
		if err := os.WriteFile(prefix+".srt", []byte(runner.whisperTranscript), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		return process.Result{ExitCode: 0}, nil
	case "fake-ffprobe":
		return process.Result{
			ExitCode: 0,
			Stdout:   []byte(`{"streams":[{"codec_name":"h264","width":640,"height":360}]}`),
		}, nil
	default:
		return process.Result{ExitCode: -1}, errors.New("unexpected executable")
	}
}

func (runner *fakeRunner) usedEncoder(name string) bool {
	for _, command := range runner.commands {
		if command.Executable != "fake-ffmpeg" {
			continue
		}
		for index := 0; index+1 < len(command.Args); index++ {
			if command.Args[index] == "-c:v" && command.Args[index+1] == name {
				return true
			}
		}
	}
	return false
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func containsPair(args []string, key, value string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key && args[index+1] == value {
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
