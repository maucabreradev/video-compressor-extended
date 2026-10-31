package compression

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/media/publisher"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

// Runner is the process capability used by the compression service.
type Runner interface {
	Execute(context.Context, process.Command, func(process.OutputChunk)) (process.Result, error)
}

// Options contains the FFmpeg executables, backend settings, and optional input
// duration used to calculate progress percentages.
type Options struct {
	FFmpegPath  string
	FFprobePath string
	Encoding    ffmpeg.Options
	Duration    time.Duration
	Priority    process.Priority
}

// ValidateOutput checks that path is a regular, non-empty video with the
// configured codec and dimensions. It is used to decide whether a prior result
// is safe to reuse after restart.
func ValidateOutput(ctx context.Context, runner Runner, ffprobePath, path string, options ffmpeg.Options) error {
	if runner == nil {
		runner = process.Runner{}
	}
	if strings.TrimSpace(ffprobePath) == "" {
		return fmt.Errorf("FFprobe executable path is required")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect video output %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("video output %q is not a non-empty regular file", path)
	}
	if err := validateVideo(ctx, runner, ffprobePath, path, options); err != nil {
		return fmt.Errorf("validate video output %q: %w", path, err)
	}
	return nil
}

// Compress encodes one planned input to a temporary MP4, validates it with
// FFprobe, and publishes it only after validation succeeds. It refuses to
// overwrite existing outputs or source files.
func Compress(
	ctx context.Context,
	runner Runner,
	plan outputplan.OutputPlan,
	options Options,
	onProgress func(process.FFmpegProgress),
) error {
	if runner == nil {
		runner = process.Runner{}
	}
	if strings.TrimSpace(plan.Source) == "" || strings.TrimSpace(plan.Video) == "" {
		return fmt.Errorf("source and video output paths are required")
	}
	if pathsEqual(plan.Source, plan.Video) {
		return fmt.Errorf("video output %q would overwrite its source", plan.Video)
	}
	inputInfo, err := os.Stat(plan.Source)
	if err != nil {
		return fmt.Errorf("inspect source %q: %w", plan.Source, err)
	}
	if !inputInfo.Mode().IsRegular() {
		return fmt.Errorf("source %q is not a regular file", plan.Source)
	}
	if _, err := os.Lstat(plan.Video); err == nil {
		return fmt.Errorf("output %q already exists", plan.Video)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect output %q: %w", plan.Video, err)
	}
	if strings.TrimSpace(options.FFmpegPath) == "" || strings.TrimSpace(options.FFprobePath) == "" {
		return fmt.Errorf("FFmpeg and FFprobe executable paths are required")
	}

	outputDir := filepath.Dir(plan.Video)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", outputDir, err)
	}
	tempFile, err := os.CreateTemp(outputDir, ".vcx-encode-*.mp4")
	if err != nil {
		return fmt.Errorf("create temporary output in %q: %w", outputDir, err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temporary output %q: %w", tempPath, err)
	}
	defer os.Remove(tempPath)

	encodeArgs, err := ffmpeg.BuildEncodeArgs(plan.Source, tempPath, options.Encoding)
	if err != nil {
		return fmt.Errorf("build FFmpeg command for %q: %w", plan.Source, err)
	}
	var progressParser *process.FFmpegProgressParser
	if onProgress != nil {
		progressParser = process.NewFFmpegProgressParser(options.Duration)
	}
	outputHandler := func(chunk process.OutputChunk) {
		if progressParser == nil || chunk.Stream != process.Stdout {
			return
		}
		for _, progress := range progressParser.Feed(chunk.Data) {
			onProgress(progress)
		}
	}
	encodeResult, err := runner.Execute(ctx, process.Command{
		Executable: options.FFmpegPath,
		Args:       encodeArgs,
		Priority:   options.Priority,
	}, outputHandler)
	if err != nil {
		return fmt.Errorf("encode %q: %w: %s", plan.Source, err, strings.TrimSpace(string(encodeResult.Stderr)))
	}
	if encodeResult.ExitCode != 0 {
		return fmt.Errorf("encode %q exited with code %d: %s", plan.Source, encodeResult.ExitCode, strings.TrimSpace(string(encodeResult.Stderr)))
	}
	encodedInfo, err := os.Stat(tempPath)
	if err != nil {
		return fmt.Errorf("FFmpeg did not create an output for %q: %w", plan.Source, err)
	}
	if encodedInfo.Size() == 0 {
		return fmt.Errorf("FFmpeg created an empty output for %q", plan.Source)
	}

	if err := validateVideo(ctx, runner, options.FFprobePath, tempPath, options.Encoding); err != nil {
		return fmt.Errorf("validate encoded output for %q: %w", plan.Source, err)
	}
	if err := publisher.PublishNoReplaceWithCopy(ctx, tempPath, plan.Video, func(path string) error {
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect copied video: %w", err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("copied video is not a non-empty regular file")
		}
		return validateVideo(ctx, runner, options.FFprobePath, path, options.Encoding)
	}); err != nil {
		return fmt.Errorf("publish encoded output %q: %w", plan.Video, err)
	}
	return nil
}

func validateVideo(ctx context.Context, runner Runner, ffprobePath, path string, options ffmpeg.Options) error {
	result, err := runner.Execute(ctx, process.Command{
		Executable: ffprobePath,
		Args: []string{
			"-v", "error",
			"-select_streams", "v:0",
			"-show_entries", "stream=codec_name,width,height",
			"-of", "json",
			path,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("run FFprobe: %w: %s", err, strings.TrimSpace(string(result.Stderr)))
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("FFprobe exited with code %d: %s", result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	var probe struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(result.Stdout, &probe); err != nil {
		return fmt.Errorf("parse FFprobe output: %w", err)
	}
	if len(probe.Streams) == 0 {
		return fmt.Errorf("output contains no video stream")
	}
	expectedCodec := strings.ToLower(strings.TrimSpace(options.Codec))
	stream := probe.Streams[0]
	if stream.CodecName != expectedCodec {
		return fmt.Errorf("expected codec %q, got %q", expectedCodec, stream.CodecName)
	}
	if stream.Width < 1 || stream.Height < 1 {
		return fmt.Errorf("invalid output dimensions %dx%d", stream.Width, stream.Height)
	}
	if stream.Width > options.MaxWidth || stream.Height > options.MaxHeight {
		return fmt.Errorf("output dimensions %dx%d exceed maximum %dx%d", stream.Width, stream.Height, options.MaxWidth, options.MaxHeight)
	}
	return nil
}

func pathsEqual(first, second string) bool {
	first = filepath.Clean(first)
	second = filepath.Clean(second)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(first, second)
	}
	return first == second
}
