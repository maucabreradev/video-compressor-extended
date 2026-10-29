package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReportValidatesCompleteWindowsBenchmarkEvidence(t *testing.T) {
	report := validReport()
	if err := report.Validate(); err != nil {
		t.Fatalf("validate complete report: %v", err)
	}
}

func TestReportRejectsMissingMachineOrEncoderDetails(t *testing.T) {
	tests := []struct {
		name   string
		update func(*Report)
	}{
		{name: "missing GPU driver", update: func(report *Report) { report.Machine.GPUs[0].DriverVersion = "" }},
		{name: "missing FFmpeg version", update: func(report *Report) { report.Tools.FFmpegVersion = "" }},
		{name: "missing VCX build hash", update: func(report *Report) { report.Tools.VCXBuildSHA256 = "" }},
		{name: "missing encoder", update: func(report *Report) { report.Settings.Encoder = "" }},
		{name: "unsupported platform", update: func(report *Report) { report.Machine.OS = "linux" }},
		{name: "unsupported architecture", update: func(report *Report) { report.Machine.Architecture = "arm64" }},
		{name: "non-AMD GPU", update: func(report *Report) { report.Machine.GPUs[0].Name = "Intel Graphics" }},
		{name: "encoder codec mismatch", update: func(report *Report) { report.Settings.Encoder = "hevc_amf" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validReport()
			test.update(&report)
			if err := report.Validate(); err == nil {
				t.Fatal("invalid benchmark report passed validation")
			}
		})
	}
}

func TestReportRejectsMissingQualityAndResourceMeasurements(t *testing.T) {
	tests := []struct {
		name   string
		update func(*Report)
	}{
		{name: "missing SSIM", update: func(report *Report) { report.Media[0].SSIM = -1 }},
		{name: "missing elapsed time", update: func(report *Report) { report.Measurements.ElapsedSeconds = 0 }},
		{name: "missing CPU time", update: func(report *Report) { report.Measurements.CPUTimeSeconds = 0 }},
		{name: "CPU percentage outside normalized range", update: func(report *Report) { report.Measurements.AverageCPUPercent = 101 }},
		{name: "missing working set", update: func(report *Report) { report.Measurements.PeakWorkingSetBytes = 0 }},
		{name: "missing GPU observation note", update: func(report *Report) { report.Measurements.GPUObservation = "" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validReport()
			test.update(&report)
			if err := report.Validate(); err == nil {
				t.Fatal("incomplete benchmark report passed validation")
			}
		})
	}
}

func TestReportRequiresBenchmarkDatasetProvenance(t *testing.T) {
	report := validReport()
	report.DatasetReference = ""
	if err := report.Validate(); err == nil {
		t.Fatal("report without dataset provenance was accepted")
	}
	report = validReport()
	report.DatasetLicense = ""
	if err := report.Validate(); err == nil {
		t.Fatal("report without dataset license/status was accepted")
	}
}

func TestLoadValidatesBenchmarkReportJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "benchmark.json")
	contents, err := json.Marshal(validReport())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("load valid report: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("report with an unknown/incomplete field set was accepted")
	}
}

func validReport() Report {
	return Report{
		SchemaVersion:    1,
		RecordedAt:       time.Date(2026, time.October, 29, 12, 0, 0, 0, time.UTC),
		DatasetName:      "licensed benchmark sample",
		DatasetReference: "https://example.test/sample",
		DatasetLicense:   "CC-BY 3.0",
		Machine: Machine{
			OS: "windows", OSVersion: "Windows 11 Pro 10.0.26200", Architecture: "amd64",
			CPU: "AMD Ryzen", GPUs: []GPU{{Name: "AMD Radeon RX 5500 XT", DriverVersion: "32.0.21045.5002"}},
		},
		Tools: Tools{VCXVersion: "2.0.0", VCXBuildSHA256: strings.Repeat("c", 64), FFmpegVersion: "FFmpeg 9.0", FFprobeVersion: "FFprobe 9.0"},
		Settings: Settings{
			Encoder: "h264_amf", Codec: "h264", ResourceProfile: "conservative",
			EncodeWorkers: 1, SubtitleWorkers: 1, WhisperThreads: 1,
			MaxWidth: 1280, MaxHeight: 720, FrameRate: 0,
		},
		Measurements: Measurements{
			ElapsedSeconds: 12.5, CPUTimeSeconds: 9.5, AverageCPUPercent: 24.0,
			PeakWorkingSetBytes: 512 * 1024 * 1024, GPUObservation: "Video encode engine observed in Performance Monitor",
		},
		Media: []MediaResult{{
			InputPath: "input/clip.mp4", InputSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			InputBytes: 1000, InputWidth: 1920, InputHeight: 1080, DurationSeconds: 10,
			OutputPath: "output/clip.mp4", OutputSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			OutputBytes: 500, OutputWidth: 1280, OutputHeight: 720, OutputFrameRate: "30/1",
			OutputCodec: "h264", SSIM: 0.98,
		}},
	}
}
