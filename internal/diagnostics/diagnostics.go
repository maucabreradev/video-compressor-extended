package diagnostics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

const (
	StatusOK            = "OK"
	StatusMissing       = "MISSING"
	StatusFailed        = "FAILED"
	StatusNotConfigured = "NOT CONFIGURED"
)

// Options selects the tools, model, and codec to check.
type Options struct {
	FFmpegCommand              string
	FFprobeCommand             string
	WhisperCommand             string
	WhisperModelPath           string
	VideoCodec                 string
	SkipWhisper                bool
	SkipFFprobe                bool
	SkipEncoder                bool
	EnableSoftwareH264Fallback bool
}

// Check describes the result of one dependency or capability check.
type Check struct {
	Name     string
	Status   string
	Message  string
	Optional bool
}

// Report contains the results of a dependency and encoder diagnostic run.
type Report struct {
	Checks            []Check
	FFmpegPath        string
	FFprobePath       string
	EncoderName       string
	EncoderDevice     string
	SoftwareH264Ready bool
}

// HasFailures reports whether any required tool, capability, or configuration is unavailable.
func (report Report) HasFailures() bool {
	for _, check := range report.Checks {
		if !check.Optional && (check.Status == StatusMissing || check.Status == StatusFailed || check.Status == StatusNotConfigured) {
			return true
		}
	}
	return false
}

// WriteReport prints the diagnostic report in a stable, user-readable format.
func (report Report) WriteReport(writer io.Writer) error {
	if _, err := fmt.Fprintln(writer, "VCX dependency diagnostics"); err != nil {
		return err
	}
	for _, check := range report.Checks {
		status := check.Status
		if check.Optional {
			status += " (optional)"
		}
		if _, err := fmt.Fprintf(writer, "%-24s %-15s %s\n", check.Name, status, check.Message); err != nil {
			return err
		}
	}
	return nil
}

// Runner abstracts executable lookup and command execution for deterministic
// diagnostics tests.
type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, string, ...string) ([]byte, []byte, error)
}

// Checker probes tools and the selected AMD hardware encoder.
type Checker struct {
	Runner          Runner
	Platform        string
	AMDDeviceFinder func() ([]string, error)
	TempDir         string
}

// Diagnose reports tool availability and verifies the requested encoder by
// producing a short synthetic clip in a temporary directory.
func (checker Checker) Diagnose(ctx context.Context, options Options) Report {
	report := Report{}
	runner := checker.Runner
	if runner == nil {
		runner = process.Runner{}
	}
	platform := checker.Platform
	if platform == "" {
		platform = runtime.GOOS
	}

	ffmpegCommand := commandOrDefault(options.FFmpegCommand, "ffmpeg")
	ffprobeCommand := commandOrDefault(options.FFprobeCommand, "ffprobe")
	whisperCommand := commandOrDefault(options.WhisperCommand, "whisper-cli")
	ffmpegPath, ffmpegOK := reportTool(&report, runner, "FFmpeg executable", ffmpegCommand)
	report.FFmpegPath = ffmpegPath
	if !options.SkipFFprobe {
		report.FFprobePath, _ = reportTool(&report, runner, "FFprobe executable", ffprobeCommand)
	}
	if !options.SkipWhisper {
		reportTool(&report, runner, "whisper.cpp executable", whisperCommand)
		checkModel(&report, options.WhisperModelPath)
	}

	if !ffmpegOK {
		if options.SkipEncoder {
			return report
		}
		report.add(Check{
			Name:    "Requested encoder",
			Status:  StatusMissing,
			Message: "cannot inspect encoders because FFmpeg is unavailable",
		})
		report.add(Check{
			Name:    "AMD hardware encode probe",
			Status:  StatusMissing,
			Message: "cannot run a hardware probe because FFmpeg is unavailable",
		})
		return report
	}
	if options.SkipEncoder {
		return report
	}

	stdout, stderr, err := runner.Run(ctx, ffmpegPath, "-hide_banner", "-encoders")
	if err != nil {
		report.add(Check{
			Name:    "FFmpeg encoder listing",
			Status:  StatusFailed,
			Message: commandError(err, stderr),
		})
		report.add(Check{
			Name:    "Requested encoder",
			Status:  StatusFailed,
			Message: "could not inspect the FFmpeg encoder list",
		})
		report.add(Check{
			Name:    "AMD hardware encode probe",
			Status:  StatusFailed,
			Message: "could not select an encoder to probe",
		})
		return report
	}
	report.add(Check{Name: "FFmpeg encoder listing", Status: StatusOK, Message: "encoder list was read"})

	backend := ffmpeg.BackendAMF
	if platform == "linux" {
		backend = ffmpeg.BackendVAAPI
	}
	encoder, err := ffmpeg.EncoderName(backend, options.VideoCodec)
	if err != nil {
		report.add(Check{Name: "Requested encoder", Status: StatusFailed, Message: err.Error()})
		report.add(Check{Name: "AMD hardware encode probe", Status: StatusFailed, Message: "no supported encoder was selected"})
		return report
	}
	report.EncoderName = encoder
	if !encoderListed(string(stdout)+"\n"+string(stderr), encoder) {
		report.add(Check{
			Name:    "Requested encoder",
			Status:  StatusMissing,
			Message: fmt.Sprintf("FFmpeg does not list %s", encoder),
		})
		report.add(Check{
			Name:    "AMD hardware encode probe",
			Status:  StatusMissing,
			Message: "the requested encoder is not listed by FFmpeg",
		})
		checker.probeSoftwareH264Fallback(ctx, runner, ffmpegPath, options, string(stdout)+"\n"+string(stderr), &report)
		return report
	}
	report.add(Check{Name: "Requested encoder", Status: StatusOK, Message: fmt.Sprintf("%s is listed by FFmpeg", encoder)})

	var devices []string
	if platform == "linux" {
		finder := checker.AMDDeviceFinder
		if finder == nil {
			finder = findAMDVAAPIDevices
		}
		devices, err = finder()
		if err != nil {
			report.add(Check{Name: "AMD hardware encode probe", Status: StatusFailed, Message: fmt.Sprintf("find AMD VAAPI devices: %v", err)})
			return report
		}
		if len(devices) == 0 {
			report.add(Check{
				Name:    "AMD hardware encode probe",
				Status:  StatusMissing,
				Message: "no AMD VAAPI render device was found; check the AMD VAAPI driver and permissions for /dev/dri render nodes",
			})
			return report
		}
	} else if platform != "windows" {
		report.add(Check{
			Name:    "AMD hardware encode probe",
			Status:  StatusFailed,
			Message: fmt.Sprintf("platform %q is not supported", platform),
		})
		return report
	}

	var probeErrors []string
	if len(devices) == 0 {
		if err := checker.probeEncoder(ctx, runner, ffmpegPath, backend, options.VideoCodec, ""); err == nil {
			report.add(Check{Name: "AMD hardware encode probe", Status: StatusOK, Message: "a temporary AMF encode succeeded"})
			return report
		} else {
			probeErrors = append(probeErrors, err.Error())
		}
	} else {
		for _, device := range devices {
			if err := checker.probeEncoder(ctx, runner, ffmpegPath, backend, options.VideoCodec, device); err == nil {
				report.EncoderDevice = device
				report.add(Check{Name: "AMD hardware encode probe", Status: StatusOK, Message: fmt.Sprintf("a temporary VAAPI encode succeeded on %s", device)})
				return report
			} else {
				probeErrors = append(probeErrors, fmt.Sprintf("%s: %v", device, err))
			}
		}
	}
	probeMessage := strings.Join(probeErrors, "; ")
	if platform == "windows" {
		probeMessage += "; check the AMD display driver and confirm an AMD GPU is available to FFmpeg"
	} else {
		probeMessage += "; check the AMD VAAPI driver and permissions for /dev/dri render nodes"
	}
	report.add(Check{
		Name:    "AMD hardware encode probe",
		Status:  StatusFailed,
		Message: probeMessage,
	})
	checker.probeSoftwareH264Fallback(ctx, runner, ffmpegPath, options, string(stdout)+"\n"+string(stderr), &report)
	return report
}

func (checker Checker) probeSoftwareH264Fallback(
	ctx context.Context,
	runner Runner,
	ffmpegPath string,
	options Options,
	encoderOutput string,
	report *Report,
) {
	if !options.EnableSoftwareH264Fallback || !strings.EqualFold(strings.TrimSpace(options.VideoCodec), "h264") {
		return
	}
	if !encoderListed(encoderOutput, "libx264") {
		report.add(Check{
			Name: "Software H.264 fallback", Status: StatusMissing,
			Message: "FFmpeg does not include the libx264 software encoder",
		})
		return
	}
	if err := checker.probeEncoder(ctx, runner, ffmpegPath, ffmpeg.BackendSoftware, "h264", ""); err != nil {
		report.add(Check{
			Name: "Software H.264 fallback", Status: StatusFailed,
			Message: commandError(err, nil),
		})
		return
	}
	report.SoftwareH264Ready = true
	report.add(Check{
		Name: "Software H.264 fallback", Status: StatusOK,
		Message: "a temporary libx264 encode succeeded; encoding may be slower and use more CPU",
	})
	markOptional(report, "Requested encoder")
	markOptional(report, "AMD hardware encode probe")
}

func markOptional(report *Report, name string) {
	for index := range report.Checks {
		if report.Checks[index].Name == name && report.Checks[index].Status != StatusOK {
			report.Checks[index].Optional = true
		}
	}
}

func (report *Report) add(check Check) {
	report.Checks = append(report.Checks, check)
}

func reportTool(report *Report, runner Runner, label, command string) (string, bool) {
	path, err := runner.LookPath(command)
	if err != nil {
		guidance := "install the dependency and add its executable directory to PATH"
		switch label {
		case "FFmpeg executable", "FFprobe executable":
			guidance = "install FFmpeg and add its bin directory to PATH"
		case "whisper.cpp executable":
			guidance = "install whisper.cpp and add whisper-cli to PATH"
		}
		report.add(Check{
			Name:    label,
			Status:  StatusMissing,
			Message: fmt.Sprintf("%q was not found; %s", command, guidance),
		})
		return "", false
	}
	report.add(Check{Name: label, Status: StatusOK, Message: path})
	return path, true
}

func checkModel(report *Report, path string) {
	if strings.TrimSpace(path) == "" {
		report.add(Check{
			Name:    "Whisper model",
			Status:  StatusNotConfigured,
			Message: "set whisper_model_path in the TOML file or pass --whisper-model-path to enable transcription",
		})
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		report.add(Check{
			Name:    "Whisper model",
			Status:  StatusMissing,
			Message: fmt.Sprintf("model file %q is unavailable: %v; download a whisper.cpp model and set whisper_model_path", path, err),
		})
		return
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		report.add(Check{Name: "Whisper model", Status: StatusFailed, Message: fmt.Sprintf("model path %q is not a non-empty regular file", path)})
		return
	}
	report.add(Check{Name: "Whisper model", Status: StatusOK, Message: path})
}

func encoderListed(output, encoder string) bool {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		for _, field := range strings.Fields(scanner.Text()) {
			if field == encoder {
				return true
			}
		}
	}
	return false
}

func (checker Checker) probeEncoder(ctx context.Context, runner Runner, ffmpegPath string, backend ffmpeg.Backend, codec, device string) error {
	tempDir, err := os.MkdirTemp(checker.TempDir, "vcx-encoder-probe-")
	if err != nil {
		return fmt.Errorf("create temporary probe directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	outputPath := filepath.Join(tempDir, "probe.mp4")
	args, err := ffmpeg.BuildProbeArgs(outputPath, ffmpeg.Options{
		Backend:     backend,
		Codec:       codec,
		VAAPIDevice: device,
		MaxWidth:    128,
		MaxHeight:   128,
	})
	if err != nil {
		return fmt.Errorf("build FFmpeg probe command: %w", err)
	}

	_, stderr, err := runner.Run(ctx, ffmpegPath, args...)
	if err != nil {
		return fmt.Errorf("FFmpeg encode probe failed: %w: %s", err, strings.TrimSpace(string(stderr)))
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("FFmpeg probe did not create output: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("FFmpeg probe created an empty output")
	}
	return nil
}

func commandOrDefault(command, fallback string) string {
	if strings.TrimSpace(command) == "" {
		return fallback
	}
	return command
}

func commandError(err error, stderr []byte) string {
	message := strings.TrimSpace(string(stderr))
	if message == "" {
		return err.Error()
	}
	return fmt.Sprintf("%v: %s", err, message)
}

func findAMDVAAPIDevices() ([]string, error) {
	const drmDirectory = "/dev/dri"
	entries, err := os.ReadDir(drmDirectory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	devices := make([]string, 0)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "renderD") {
			continue
		}
		vendorPath := filepath.Join("/sys/class/drm", entry.Name(), "device", "vendor")
		vendor, err := os.ReadFile(vendorPath)
		if err != nil || !strings.EqualFold(strings.TrimSpace(string(vendor)), "0x1002") {
			continue
		}
		devices = append(devices, filepath.Join(drmDirectory, entry.Name()))
	}
	sort.Strings(devices)
	return devices, nil
}
