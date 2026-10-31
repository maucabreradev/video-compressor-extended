package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/media/publisher"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

// Runner is the process capability used by subtitle generation.
type Runner interface {
	Execute(context.Context, process.Command, func(process.OutputChunk)) (process.Result, error)
}

// Options contains the external tools, model, language, and CPU thread limit
// used for one transcription job.
type Options struct {
	FFmpegPath  string
	WhisperPath string
	ModelPath   string
	// WorkspacePath is caller-owned storage for fixed audio/SRT intermediates.
	// When empty, Generate creates and removes a temporary directory beside output.
	WorkspacePath string
	// OnStaged persists a validated SRT mapping before publication begins.
	OnStaged func() error
	Language string
	Threads  int
	UseGPU   bool
	Priority process.Priority
}

// ValidateOutput checks that path contains a non-empty, regular, valid SRT
// artifact suitable for reuse after restart.
func ValidateOutput(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect subtitle output %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("subtitle output %q is not a non-empty regular file", path)
	}
	if err := ValidateSRT(path); err != nil {
		return fmt.Errorf("validate subtitle output %q: %w", path, err)
	}
	return nil
}

// Generate extracts audio from the original source, transcribes it with
// whisper.cpp, validates the SRT, and publishes it without replacing a file.
func Generate(ctx context.Context, runner Runner, plan outputplan.OutputPlan, options Options) error {
	if runner == nil {
		runner = process.Runner{}
	}
	if strings.TrimSpace(plan.Source) == "" || strings.TrimSpace(plan.Subtitle) == "" {
		return fmt.Errorf("source and subtitle output paths are required")
	}
	if strings.TrimSpace(options.FFmpegPath) == "" {
		return fmt.Errorf("FFmpeg executable path is required to extract subtitle audio")
	}
	if strings.TrimSpace(options.WhisperPath) == "" {
		return fmt.Errorf("whisper.cpp executable was not found; install whisper-cli and add it to PATH")
	}
	if strings.TrimSpace(options.ModelPath) == "" {
		return fmt.Errorf("Whisper model is not configured; set whisper_model_path")
	}
	if strings.TrimSpace(options.Language) == "" {
		return fmt.Errorf("subtitle language must not be empty")
	}
	if options.Threads < 1 {
		return fmt.Errorf("Whisper CPU thread count must be at least 1")
	}
	if options.OnStaged != nil && strings.TrimSpace(options.WorkspacePath) == "" {
		return fmt.Errorf("a caller-owned workspace is required when persisting a staged SRT")
	}

	sourceInfo, err := os.Stat(plan.Source)
	if err != nil {
		return fmt.Errorf("inspect original media %q: %w", plan.Source, err)
	}
	if !sourceInfo.Mode().IsRegular() {
		return fmt.Errorf("original media %q is not a regular file", plan.Source)
	}
	modelInfo, err := os.Stat(options.ModelPath)
	if err != nil {
		return fmt.Errorf("inspect Whisper model %q: %w; download a whisper.cpp model and set whisper_model_path", options.ModelPath, err)
	}
	if !modelInfo.Mode().IsRegular() || modelInfo.Size() == 0 {
		return fmt.Errorf("Whisper model %q is not a non-empty regular file", options.ModelPath)
	}
	if _, err := os.Lstat(plan.Subtitle); err == nil {
		return fmt.Errorf("subtitle output %q already exists", plan.Subtitle)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect subtitle output %q: %w", plan.Subtitle, err)
	}

	outputDirectory := filepath.Dir(plan.Subtitle)
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return fmt.Errorf("create subtitle output directory %q: %w", outputDirectory, err)
	}
	temporaryDirectory := options.WorkspacePath
	if strings.TrimSpace(temporaryDirectory) == "" {
		temporaryDirectory, err = os.MkdirTemp(outputDirectory, ".vcx-subtitle-*")
		if err != nil {
			return fmt.Errorf("create temporary subtitle directory in %q: %w", outputDirectory, err)
		}
		defer os.RemoveAll(temporaryDirectory)
	} else {
		info, err := os.Lstat(temporaryDirectory)
		if err != nil {
			return fmt.Errorf("inspect subtitle workspace %q: %w", temporaryDirectory, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("subtitle workspace %q is not a regular directory", temporaryDirectory)
		}
	}

	audioPath := filepath.Join(temporaryDirectory, "audio.wav")
	if err := requireWorkspaceArtifactAbsent(audioPath); err != nil {
		return err
	}
	if err := extractAudio(ctx, runner, options.FFmpegPath, options.Priority, plan.Source, audioPath); err != nil {
		return fmt.Errorf("extract audio from original media %q: %w", plan.Source, err)
	}
	outputPrefix := filepath.Join(temporaryDirectory, "transcript")
	if err := requireWorkspaceArtifactAbsent(outputPrefix + ".srt"); err != nil {
		return err
	}
	if err := transcribe(ctx, runner, options, audioPath, outputPrefix); err != nil {
		return fmt.Errorf("transcribe original media %q: %w", plan.Source, err)
	}
	temporarySRT := outputPrefix + ".srt"
	if err := ValidateSRT(temporarySRT); err != nil {
		return fmt.Errorf("validate SRT output for %q: %w", plan.Source, err)
	}
	if options.OnStaged != nil {
		if err := options.OnStaged(); err != nil {
			return fmt.Errorf("persist validated staged SRT: %w", err)
		}
	}
	if err := PublishStaged(ctx, temporarySRT, plan.Subtitle); err != nil {
		return fmt.Errorf("publish subtitle output %q: %w", plan.Subtitle, err)
	}
	return nil
}

// PublishStaged revalidates a workspace SRT and publishes it without replacing
// an existing destination. The caller retains ownership of the staged file.
func PublishStaged(ctx context.Context, stagedPath, destination string) error {
	if strings.TrimSpace(destination) == "" {
		return fmt.Errorf("subtitle destination path is required")
	}
	if err := ValidateOutput(stagedPath); err != nil {
		return fmt.Errorf("validate staged subtitle before publication: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create subtitle output directory %q: %w", filepath.Dir(destination), err)
	}
	if err := publisher.PublishNoReplaceWithCopy(ctx, stagedPath, destination, ValidateOutput); err != nil {
		return err
	}
	return nil
}

func requireWorkspaceArtifactAbsent(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("subtitle workspace artifact %q already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect subtitle workspace artifact %q: %w", path, err)
	}
	return nil
}

func extractAudio(ctx context.Context, runner Runner, ffmpegPath string, priority process.Priority, source, destination string) error {
	result, err := runner.Execute(ctx, process.Command{
		Executable: ffmpegPath,
		Priority:   priority,
		Args: []string{
			"-hide_banner", "-nostdin", "-y",
			"-i", source,
			"-map", "0:a:0", "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le",
			destination,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("run FFmpeg: %w: %s", err, strings.TrimSpace(string(result.Stderr)))
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("FFmpeg exited with code %d: %s", result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	info, err := os.Stat(destination)
	if err != nil {
		return fmt.Errorf("FFmpeg did not create extracted audio: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("FFmpeg created empty extracted audio")
	}
	return nil
}

func transcribe(ctx context.Context, runner Runner, options Options, audioPath, outputPrefix string) error {
	args := []string{
		"-m", options.ModelPath,
		"-f", audioPath,
		"-l", options.Language,
		"-t", fmt.Sprintf("%d", options.Threads),
	}
	if !options.UseGPU {
		args = append(args, "-ng")
	}
	args = append(args, "-osrt", "-of", outputPrefix)
	executable, modelPath, directory, cleanup, err := prepareWhisperInvocation(options.WhisperPath, options.ModelPath)
	if err != nil {
		return fmt.Errorf("prepare Whisper invocation for runtime paths: %w", err)
	}
	defer cleanup()
	args[1] = modelPath
	result, err := runner.Execute(ctx, process.Command{
		Executable: executable,
		Dir:        directory,
		Args:       args,
		Priority:   options.Priority,
	}, nil)
	if err != nil {
		return fmt.Errorf("run whisper.cpp: %w: %s", err, strings.TrimSpace(string(result.Stderr)))
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("whisper.cpp exited with code %d: %s", result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	return nil
}
