package compression

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

func TestCompressPublishesOnlyAValidatedVideoAndEmitsProgress(t *testing.T) {
	plan := createOutputPlan(t)
	runner := &fakeRunner{
		probeOutput: `{"streams":[{"codec_name":"h264","width":640,"height":360}]}`,
	}
	var progress []process.FFmpegProgress
	options := testOptions()
	options.WorkspacePath = createCompressionWorkspace(t)

	err := Compress(context.Background(), runner, plan, options,
		func(event process.FFmpegProgress) { progress = append(progress, event) })
	if err != nil {
		t.Fatalf("compress media: %v", err)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "encoded video" {
		t.Fatalf("validated output was not published: contents=%q err=%v", contents, err)
	}
	if len(progress) != 1 || progress[0].Frame != 10 {
		t.Fatalf("expected FFmpeg progress event, got %#v", progress)
	}
	if runner.lastTempPath == plan.Video {
		t.Fatal("FFmpeg must write to a temporary output before publishing")
	}
	if _, err := os.Stat(runner.lastTempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary output was not cleaned up: %v", err)
	}
}

func TestCompressUsesFixedWorkspaceArtifactWithMP4Container(t *testing.T) {
	plan := createOutputPlan(t)
	workspacePath := filepath.Join(t.TempDir(), "w-001122334455")
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{probeOutput: `{"streams":[{"codec_name":"h264","width":640,"height":360}]}`}
	options := testOptions()
	options.WorkspacePath = workspacePath

	if err := Compress(context.Background(), runner, plan, options, nil); err != nil {
		t.Fatalf("compress through short workspace: %v", err)
	}
	want := filepath.Join(workspacePath, "video.mp4")
	if runner.lastTempPath != want {
		t.Fatalf("FFmpeg output = %q, want fixed workspace artifact %q", runner.lastTempPath, want)
	}
	if !containsPair(runner.commands[0].Args, "-f", "mp4") {
		t.Fatalf("FFmpeg MP4 container semantics were not retained: %#v", runner.commands[0].Args)
	}
	if _, err := os.Stat(want); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("workspace artifact was not removed after publication: %v", err)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "encoded video" {
		t.Fatalf("published output mismatch: contents=%q err=%v", contents, err)
	}
}

func TestCompressPassesBestEffortPriorityToEncoder(t *testing.T) {
	plan := createOutputPlan(t)
	runner := &fakeRunner{probeOutput: `{"streams":[{"codec_name":"h264","width":640,"height":360}]}`}
	options := testOptions()
	options.Priority = process.PriorityBelowNormal

	if err := Compress(context.Background(), runner, plan, options, nil); err != nil {
		t.Fatalf("compress video: %v", err)
	}
	if len(runner.commands) == 0 || runner.commands[0].Priority != process.PriorityBelowNormal {
		t.Fatalf("encoder did not receive the best-effort priority hint: %+v", runner.commands)
	}
}

func TestValidateOutputRequiresARegularNonEmptyValidVideo(t *testing.T) {
	plan := createOutputPlan(t)
	if err := os.MkdirAll(filepath.Dir(plan.Video), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.Video, []byte("video artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOutput(context.Background(), &fakeRunner{
		probeOutput: `{"streams":[{"codec_name":"h264","width":640,"height":360}]}`,
	}, "ffprobe", plan.Video, testOptions().Encoding); err != nil {
		t.Fatalf("validate completed video: %v", err)
	}
	if err := ValidateOutput(context.Background(), &fakeRunner{
		probeOutput: `{"streams":[{"codec_name":"hevc","width":640,"height":360}]}`,
	}, "ffprobe", plan.Video, testOptions().Encoding); err == nil {
		t.Fatal("expected codec mismatch to invalidate existing video")
	}
	if err := ValidateOutput(context.Background(), &fakeRunner{}, "ffprobe", filepath.Join(t.TempDir(), "missing.mp4"), testOptions().Encoding); err == nil {
		t.Fatal("expected missing output to be invalid")
	}
}

func TestCompressDoesNotPublishOutputWhenProbeValidationFails(t *testing.T) {
	plan := createOutputPlan(t)
	options := testOptions()
	options.WorkspacePath = createCompressionWorkspace(t)
	runner := &fakeRunner{
		probeOutput: `{"streams":[{"codec_name":"hevc","width":640,"height":360}]}`,
	}

	err := Compress(context.Background(), runner, plan, options, nil)
	if err == nil || !strings.Contains(err.Error(), "codec") {
		t.Fatalf("expected codec validation error, got %v", err)
	}
	if _, statErr := os.Stat(plan.Video); !os.IsNotExist(statErr) {
		t.Fatalf("invalid video was published: %v", statErr)
	}
	if _, statErr := os.Stat(runner.lastTempPath); !os.IsNotExist(statErr) {
		t.Fatalf("invalid temporary output was not cleaned: %v", statErr)
	}
}

func TestCompressDoesNotOverwriteExistingOutput(t *testing.T) {
	plan := createOutputPlan(t)
	if err := os.MkdirAll(filepath.Dir(plan.Video), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.Video, []byte("keep existing output"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{probeOutput: `{"streams":[{"codec_name":"h264","width":640,"height":360}]}`}
	options := testOptions()
	options.WorkspacePath = createCompressionWorkspace(t)

	if err := Compress(context.Background(), runner, plan, options, nil); err == nil {
		t.Fatal("expected existing output to be preserved")
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "keep existing output" {
		t.Fatalf("existing output was changed: contents=%q err=%v", contents, err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("external tools ran despite existing output: %#v", runner.commands)
	}
}

func TestCompressPreservesDestinationCreatedDuringEncodingAndCleansTemporaryFile(t *testing.T) {
	plan := createOutputPlan(t)
	runner := &fakeRunner{
		probeOutput: `{"streams":[{"codec_name":"h264","width":640,"height":360}]}`,
		beforeProbe: func() {
			if err := os.WriteFile(plan.Video, []byte("concurrent output"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	options := testOptions()
	options.WorkspacePath = createCompressionWorkspace(t)

	if err := Compress(context.Background(), runner, plan, options, nil); err == nil {
		t.Fatal("expected a destination created during encoding to block publication")
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "concurrent output" {
		t.Fatalf("concurrent destination changed: contents=%q err=%v", contents, err)
	}
	if _, err := os.Stat(runner.lastTempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary output was not removed after publish conflict: %v", err)
	}
}

func TestCompressCleansTemporaryFileAfterEncoderFailure(t *testing.T) {
	plan := createOutputPlan(t)
	runner := &fakeRunner{encodeError: os.ErrPermission}
	options := testOptions()
	options.WorkspacePath = createCompressionWorkspace(t)

	if err := Compress(context.Background(), runner, plan, options, nil); err == nil {
		t.Fatal("expected encoder failure to be returned")
	}
	if _, err := os.Stat(runner.lastTempPath); !os.IsNotExist(err) {
		t.Fatalf("failed temporary output was not removed: %v", err)
	}
	if _, err := os.Stat(plan.Video); !os.IsNotExist(err) {
		t.Fatalf("failed output was published: %v", err)
	}
}

func TestCompressCancellationCleansPartialOutputAndPreservesSource(t *testing.T) {
	plan := createOutputPlan(t)
	sourceBefore, err := os.ReadFile(plan.Source)
	if err != nil {
		t.Fatal(err)
	}
	runner := &blockingEncodeRunner{started: make(chan string, 1)}
	options := testOptions()
	options.WorkspacePath = createCompressionWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Compress(ctx, runner, plan, options, nil) }()

	var temporaryPath string
	select {
	case temporaryPath = <-runner.started:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("encoder did not start")
	}
	cancel()
	err = <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("compression error = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(temporaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial temporary output remains after cancellation: %v", err)
	}
	if _, err := os.Stat(plan.Video); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled video was published: %v", err)
	}
	if sourceAfter, err := os.ReadFile(plan.Source); err != nil || string(sourceAfter) != string(sourceBefore) {
		t.Fatalf("original media changed after cancellation: contents=%q err=%v", sourceAfter, err)
	}
}

type fakeRunner struct {
	commands     []process.Command
	probeOutput  string
	encodeError  error
	lastTempPath string
	beforeProbe  func()
}

type blockingEncodeRunner struct {
	started chan string
}

func (runner *blockingEncodeRunner) Execute(ctx context.Context, command process.Command, _ func(process.OutputChunk)) (process.Result, error) {
	if command.Executable != "ffmpeg" {
		return process.Result{ExitCode: -1}, os.ErrNotExist
	}
	temporaryPath := command.Args[len(command.Args)-1]
	if err := os.WriteFile(temporaryPath, []byte("partial encode"), 0o600); err != nil {
		return process.Result{ExitCode: 1}, err
	}
	runner.started <- temporaryPath
	<-ctx.Done()
	return process.Result{ExitCode: -1}, ctx.Err()
}

func (runner *fakeRunner) Execute(_ context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	runner.commands = append(runner.commands, command)
	switch command.Executable {
	case "ffmpeg":
		runner.lastTempPath = command.Args[len(command.Args)-1]
		if runner.encodeError != nil {
			return process.Result{ExitCode: 1}, runner.encodeError
		}
		if err := os.WriteFile(runner.lastTempPath, []byte("encoded video"), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		if onOutput != nil {
			onOutput(process.OutputChunk{Stream: process.Stdout, Data: []byte("frame=10\nout_time_us=1000000\nprogress=continue\n")})
		}
		return process.Result{ExitCode: 0}, nil
	case "ffprobe":
		if runner.beforeProbe != nil {
			runner.beforeProbe()
		}
		return process.Result{Stdout: []byte(runner.probeOutput), ExitCode: 0}, nil
	default:
		return process.Result{ExitCode: -1}, os.ErrNotExist
	}
}

func createOutputPlan(t *testing.T) outputplan.OutputPlan {
	t.Helper()
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	input := filepath.Join(inputRoot, "sample.mkv")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("source video"), 0o600); err != nil {
		t.Fatal(err)
	}
	plans, err := outputplan.PlanOutputs(inputRoot, filepath.Join(root, "output"), []string{input})
	if err != nil {
		t.Fatal(err)
	}
	return plans[0]
}

func testOptions() Options {
	return Options{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		Encoding: ffmpeg.Options{
			Backend:   ffmpeg.BackendAMF,
			Codec:     "h264",
			MaxWidth:  1280,
			MaxHeight: 720,
		},
	}
}

func createCompressionWorkspace(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "w-001122334455")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func containsPair(args []string, key, value string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key && args[index+1] == value {
			return true
		}
	}
	return false
}
