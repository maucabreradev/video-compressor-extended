package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/benchmark"
)

func TestRunValidatesAndSummarizesBenchmarkReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "benchmark.json")
	report := benchmark.Report{
		SchemaVersion:    1,
		RecordedAt:       time.Date(2026, time.October, 29, 12, 0, 0, 0, time.UTC),
		DatasetName:      "licensed benchmark sample",
		DatasetReference: "https://example.test/sample",
		DatasetLicense:   "CC-BY 3.0",
		Machine: benchmark.Machine{
			OS: "windows", OSVersion: "Windows 11", Architecture: "amd64", CPU: "AMD Ryzen",
			GPUs: []benchmark.GPU{{Name: "AMD Radeon", DriverVersion: "1.2.3"}},
		},
		Tools:        benchmark.Tools{VCXVersion: "2.0.0", VCXBuildSHA256: strings.Repeat("c", 64), FFmpegVersion: "FFmpeg 9", FFprobeVersion: "FFprobe 9"},
		Settings:     benchmark.Settings{Encoder: "h264_amf", Codec: "h264", ResourceProfile: "conservative", EncodeWorkers: 1, SubtitleWorkers: 1, WhisperThreads: 1, MaxWidth: 1280, MaxHeight: 720},
		Measurements: benchmark.Measurements{ElapsedSeconds: 1, CPUTimeSeconds: 0.5, AverageCPUPercent: 8.3, PeakWorkingSetBytes: 1024, GPUObservation: "Counters unavailable on test fixture"},
		Media:        []benchmark.MediaResult{{InputPath: "input.mp4", InputSHA256: strings.Repeat("a", 64), InputBytes: 10, InputWidth: 1920, InputHeight: 1080, DurationSeconds: 1, OutputPath: "output.mp4", OutputSHA256: strings.Repeat("b", 64), OutputBytes: 8, OutputWidth: 1280, OutputHeight: 720, OutputFrameRate: "30/1", OutputCodec: "h264", SSIM: 0.95}},
	}
	contents, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := run([]string{"-report", path}, &stdout); err != nil {
		t.Fatalf("validate benchmark report: %v", err)
	}
	if !strings.Contains(stdout.String(), "1 media file") || !strings.Contains(stdout.String(), "h264_amf") {
		t.Fatalf("report summary missing fields: %q", stdout.String())
	}
}
