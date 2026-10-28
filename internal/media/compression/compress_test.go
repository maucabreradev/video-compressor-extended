package compression

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	err := Compress(context.Background(), runner, plan, Options{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
		Encoding: ffmpeg.Options{
			Backend:   ffmpeg.BackendAMF,
			Codec:     "h264",
			MaxWidth:  1280,
			MaxHeight: 720,
		},
	}, func(event process.FFmpegProgress) { progress = append(progress, event) })
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

func TestCompressDoesNotPublishOutputWhenProbeValidationFails(t *testing.T) {
	plan := createOutputPlan(t)
	runner := &fakeRunner{
		probeOutput: `{"streams":[{"codec_name":"hevc","width":640,"height":360}]}`,
	}

	err := Compress(context.Background(), runner, plan, testOptions(), nil)
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

	if err := Compress(context.Background(), runner, plan, testOptions(), nil); err == nil {
		t.Fatal("expected existing output to be preserved")
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "keep existing output" {
		t.Fatalf("existing output was changed: contents=%q err=%v", contents, err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("external tools ran despite existing output: %#v", runner.commands)
	}
}

func TestCompressCleansTemporaryFileAfterEncoderFailure(t *testing.T) {
	plan := createOutputPlan(t)
	runner := &fakeRunner{encodeError: os.ErrPermission}

	if err := Compress(context.Background(), runner, plan, testOptions(), nil); err == nil {
		t.Fatal("expected encoder failure to be returned")
	}
	if _, err := os.Stat(runner.lastTempPath); !os.IsNotExist(err) {
		t.Fatalf("failed temporary output was not removed: %v", err)
	}
	if _, err := os.Stat(plan.Video); !os.IsNotExist(err) {
		t.Fatalf("failed output was published: %v", err)
	}
}

type fakeRunner struct {
	commands     []process.Command
	probeOutput  string
	encodeError  error
	lastTempPath string
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
