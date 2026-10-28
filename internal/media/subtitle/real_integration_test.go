//go:build integration

package subtitle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

func TestGenerateWithRealWhisper(t *testing.T) {
	whisperPath := os.Getenv("VCX_WHISPER_CLI")
	modelPath := os.Getenv("VCX_WHISPER_MODEL")
	audioPath := os.Getenv("VCX_WHISPER_SAMPLE")
	if whisperPath == "" || modelPath == "" || audioPath == "" {
		t.Skip("set VCX_WHISPER_CLI, VCX_WHISPER_MODEL, and VCX_WHISPER_SAMPLE to run the real Whisper integration test")
	}
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skipf("FFmpeg is unavailable: %v", err)
	}
	if _, err := os.Stat(audioPath); err != nil {
		t.Fatalf("inspect real transcription sample: %v", err)
	}
	if _, err := os.Stat(modelPath); err != nil {
		t.Fatalf("inspect configured Whisper model: %v", err)
	}

	plan := outputplan.OutputPlan{
		Source:   audioPath,
		Subtitle: filepath.Join(t.TempDir(), "sample.srt"),
	}
	err = Generate(context.Background(), process.Runner{}, plan, Options{
		FFmpegPath:  ffmpegPath,
		WhisperPath: whisperPath,
		ModelPath:   modelPath,
		Language:    "en",
		Threads:     2,
	})
	if err != nil {
		t.Fatalf("generate real SRT: %v", err)
	}
	if err := ValidateSRT(plan.Subtitle); err != nil {
		t.Fatalf("validate real Whisper output: %v", err)
	}
}
