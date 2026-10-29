// Package benchmark defines and validates VCX's reproducible Windows benchmark
// report format.
package benchmark

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

// Report is one measured VCX run over one or more input files.
type Report struct {
	SchemaVersion    int           `json:"schema_version"`
	RecordedAt       time.Time     `json:"recorded_at"`
	DatasetName      string        `json:"dataset_name"`
	DatasetReference string        `json:"dataset_reference"`
	DatasetLicense   string        `json:"dataset_license"`
	Machine          Machine       `json:"machine"`
	Tools            Tools         `json:"tools"`
	Settings         Settings      `json:"settings"`
	Measurements     Measurements  `json:"measurements"`
	Media            []MediaResult `json:"media"`
}

// Machine describes the Windows host used for the benchmark.
type Machine struct {
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Architecture string `json:"architecture"`
	CPU          string `json:"cpu"`
	GPUs         []GPU  `json:"gpus"`
}

// GPU describes one adapter and driver installed on the benchmark host.
type GPU struct {
	Name          string `json:"name"`
	DriverVersion string `json:"driver_version"`
}

// Tools records the versions used to produce the benchmark.
type Tools struct {
	VCXVersion     string `json:"vcx_version"`
	VCXBuildSHA256 string `json:"vcx_build_sha256"`
	FFmpegVersion  string `json:"ffmpeg_version"`
	FFprobeVersion string `json:"ffprobe_version"`
}

// Settings records the encoder and resource limits for this run.
type Settings struct {
	Encoder         string `json:"encoder"`
	Codec           string `json:"codec"`
	ResourceProfile string `json:"resource_profile"`
	EncodeWorkers   int    `json:"encode_workers"`
	SubtitleWorkers int    `json:"subtitle_workers"`
	WhisperThreads  int    `json:"whisper_threads"`
	MaxWidth        int    `json:"max_width"`
	MaxHeight       int    `json:"max_height"`
	FrameRate       int    `json:"frame_rate"`
}

// Measurements contains run-level wall-clock and resource observations.
type Measurements struct {
	ElapsedSeconds                    float64  `json:"elapsed_seconds"`
	CPUTimeSeconds                    float64  `json:"cpu_time_seconds"`
	AverageCPUPercent                 float64  `json:"average_cpu_percent"`
	PeakWorkingSetBytes               uint64   `json:"peak_working_set_bytes"`
	PeakVideoEncodeUtilizationPercent *float64 `json:"peak_video_encode_utilization_percent,omitempty"`
	PeakDedicatedGPUMemoryBytes       *uint64  `json:"peak_dedicated_gpu_memory_bytes,omitempty"`
	GPUObservation                    string   `json:"gpu_observation"`
}

// MediaResult records source/output properties and an SSIM quality score.
type MediaResult struct {
	InputPath       string  `json:"input_path"`
	InputSHA256     string  `json:"input_sha256"`
	InputBytes      int64   `json:"input_bytes"`
	InputWidth      int     `json:"input_width"`
	InputHeight     int     `json:"input_height"`
	DurationSeconds float64 `json:"duration_seconds"`
	OutputPath      string  `json:"output_path"`
	OutputSHA256    string  `json:"output_sha256"`
	OutputBytes     int64   `json:"output_bytes"`
	OutputWidth     int     `json:"output_width"`
	OutputHeight    int     `json:"output_height"`
	OutputFrameRate string  `json:"output_frame_rate"`
	OutputCodec     string  `json:"output_codec"`
	SSIM            float64 `json:"ssim"`
}

// Load reads a strict JSON benchmark report and validates its measurements.
func Load(path string) (Report, error) {
	file, err := os.Open(path)
	if err != nil {
		return Report{}, fmt.Errorf("open benchmark report: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode benchmark report: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Report{}, fmt.Errorf("benchmark report contains multiple JSON values")
		}
		return Report{}, fmt.Errorf("read benchmark report trailer: %w", err)
	}
	if err := report.Validate(); err != nil {
		return Report{}, fmt.Errorf("validate benchmark report: %w", err)
	}
	return report, nil
}

// Validate rejects reports that are incomplete or inconsistent with the
// Windows AMF benchmark contract.
func (report Report) Validate() error {
	if report.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1, got %d", report.SchemaVersion)
	}
	if report.RecordedAt.IsZero() {
		return fmt.Errorf("recorded_at must be set")
	}
	if strings.TrimSpace(report.DatasetName) == "" || strings.TrimSpace(report.DatasetReference) == "" || strings.TrimSpace(report.DatasetLicense) == "" {
		return fmt.Errorf("dataset_name, dataset_reference, and dataset_license must be set")
	}
	if !strings.EqualFold(report.Machine.OS, "windows") {
		return fmt.Errorf("machine.os must be windows, got %q", report.Machine.OS)
	}
	if !strings.EqualFold(report.Machine.Architecture, "amd64") {
		return fmt.Errorf("machine.architecture must be amd64, got %q", report.Machine.Architecture)
	}
	for name, value := range map[string]string{
		"machine.os_version":   report.Machine.OSVersion,
		"machine.architecture": report.Machine.Architecture,
		"machine.cpu":          report.Machine.CPU,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be set", name)
		}
	}
	if len(report.Machine.GPUs) == 0 {
		return fmt.Errorf("machine.gpus must contain at least one adapter")
	}
	hasAMDGPU := false
	for index, gpu := range report.Machine.GPUs {
		if strings.TrimSpace(gpu.Name) == "" || strings.TrimSpace(gpu.DriverVersion) == "" {
			return fmt.Errorf("machine.gpus[%d] must include name and driver_version", index)
		}
		if strings.Contains(strings.ToLower(gpu.Name), "amd") || strings.Contains(strings.ToLower(gpu.Name), "advanced micro devices") {
			hasAMDGPU = true
		}
	}
	if !hasAMDGPU {
		return fmt.Errorf("machine.gpus must include the AMD adapter used by AMF")
	}
	for name, value := range map[string]string{
		"tools.vcx_version":     report.Tools.VCXVersion,
		"tools.ffmpeg_version":  report.Tools.FFmpegVersion,
		"tools.ffprobe_version": report.Tools.FFprobeVersion,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be set", name)
		}
	}
	if strings.TrimSpace(report.Settings.Encoder) == "" {
		return fmt.Errorf("settings.encoder must be set")
	}
	if !validSHA256(report.Tools.VCXBuildSHA256) {
		return fmt.Errorf("tools.vcx_build_sha256 must be a valid SHA-256 hash")
	}
	if !strings.HasSuffix(strings.ToLower(report.Settings.Encoder), "_amf") {
		return fmt.Errorf("settings.encoder must identify an AMF encoder, got %q", report.Settings.Encoder)
	}
	switch strings.ToLower(report.Settings.Codec) {
	case "h264", "hevc":
	default:
		return fmt.Errorf("settings.codec must be h264 or hevc, got %q", report.Settings.Codec)
	}
	expectedEncoder := strings.ToLower(report.Settings.Codec) + "_amf"
	if !strings.EqualFold(report.Settings.Encoder, expectedEncoder) {
		return fmt.Errorf("settings.encoder %q does not match codec %q", report.Settings.Encoder, report.Settings.Codec)
	}
	switch report.Settings.ResourceProfile {
	case "conservative", "custom":
	default:
		return fmt.Errorf("settings.resource_profile must be conservative or custom, got %q", report.Settings.ResourceProfile)
	}
	if report.Settings.EncodeWorkers < 1 || report.Settings.SubtitleWorkers < 1 || report.Settings.WhisperThreads < 1 {
		return fmt.Errorf("worker and thread settings must be at least 1")
	}
	if report.Settings.MaxWidth < 2 || report.Settings.MaxHeight < 2 || report.Settings.FrameRate < 0 {
		return fmt.Errorf("resolution must be positive and frame_rate must not be negative")
	}
	measurements := report.Measurements
	if !finite(measurements.ElapsedSeconds) || measurements.ElapsedSeconds <= 0 {
		return fmt.Errorf("measurements.elapsed_seconds must be finite and positive")
	}
	if !finite(measurements.CPUTimeSeconds) || measurements.CPUTimeSeconds <= 0 {
		return fmt.Errorf("measurements.cpu_time_seconds must be finite and positive")
	}
	if !finite(measurements.AverageCPUPercent) || measurements.AverageCPUPercent < 0 || measurements.AverageCPUPercent > 100 {
		return fmt.Errorf("measurements.average_cpu_percent must be between 0 and 100")
	}
	if measurements.PeakWorkingSetBytes == 0 {
		return fmt.Errorf("measurements.peak_working_set_bytes must be positive")
	}
	if strings.TrimSpace(measurements.GPUObservation) == "" {
		return fmt.Errorf("measurements.gpu_observation must describe the observation or why it was unavailable")
	}
	if measurements.PeakVideoEncodeUtilizationPercent != nil &&
		(!finite(*measurements.PeakVideoEncodeUtilizationPercent) || *measurements.PeakVideoEncodeUtilizationPercent < 0 || *measurements.PeakVideoEncodeUtilizationPercent > 100) {
		return fmt.Errorf("measurements.peak_video_encode_utilization_percent must be between 0 and 100")
	}
	if len(report.Media) == 0 {
		return fmt.Errorf("media must contain at least one measured input")
	}
	for index, media := range report.Media {
		if strings.TrimSpace(media.InputPath) == "" || strings.TrimSpace(media.OutputPath) == "" {
			return fmt.Errorf("media[%d] must include input_path and output_path", index)
		}
		if !validSHA256(media.InputSHA256) || !validSHA256(media.OutputSHA256) {
			return fmt.Errorf("media[%d] must include valid input and output SHA-256 hashes", index)
		}
		if media.InputBytes <= 0 || media.OutputBytes <= 0 || media.InputWidth < 2 || media.InputHeight < 2 ||
			media.OutputWidth < 2 || media.OutputHeight < 2 || !finite(media.DurationSeconds) || media.DurationSeconds <= 0 {
			return fmt.Errorf("media[%d] has invalid size, dimensions, or duration", index)
		}
		if strings.TrimSpace(media.OutputFrameRate) == "" {
			return fmt.Errorf("media[%d].output_frame_rate must be set", index)
		}
		if !strings.EqualFold(media.OutputCodec, report.Settings.Codec) {
			return fmt.Errorf("media[%d].output_codec %q does not match settings.codec %q", index, media.OutputCodec, report.Settings.Codec)
		}
		if !finite(media.SSIM) || media.SSIM < 0 || media.SSIM > 1 {
			return fmt.Errorf("media[%d].ssim must be between 0 and 1", index)
		}
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
