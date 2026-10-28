package diagnostics

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	StatusOK            = "OK"
	StatusMissing       = "MISSING"
	StatusFailed        = "FAILED"
	StatusNotConfigured = "NOT CONFIGURED"
)

// Options selects the tools, model, and codec to check.
type Options struct {
	FFmpegCommand    string
	FFprobeCommand   string
	WhisperCommand   string
	WhisperModelPath string
	VideoCodec       string
}

// Check describes the result of one dependency or capability check.
type Check struct {
	Name    string
	Status  string
	Message string
}

// Report contains the results of a dependency and encoder diagnostic run.
type Report struct {
	Checks []Check
}

// HasFailures reports whether any required tool or capability is unavailable.
func (report Report) HasFailures() bool {
	for _, check := range report.Checks {
		if check.Status == StatusMissing || check.Status == StatusFailed {
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
		if _, err := fmt.Fprintf(writer, "%-24s %-15s %s\n", check.Name, check.Status, check.Message); err != nil {
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
		runner = execRunner{}
	}
	platform := checker.Platform
	if platform == "" {
		platform = runtime.GOOS
	}

	ffmpegCommand := commandOrDefault(options.FFmpegCommand, "ffmpeg")
	ffprobeCommand := commandOrDefault(options.FFprobeCommand, "ffprobe")
	whisperCommand := commandOrDefault(options.WhisperCommand, "whisper-cli")
	ffmpegPath, ffmpegOK := reportTool(&report, runner, "FFmpeg executable", ffmpegCommand)
	reportTool(&report, runner, "FFprobe executable", ffprobeCommand)
	reportTool(&report, runner, "whisper.cpp executable", whisperCommand)
	checkModel(&report, options.WhisperModelPath)

	if !ffmpegOK {
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

	encoder, err := encoderName(platform, options.VideoCodec)
	if err != nil {
		report.add(Check{Name: "Requested encoder", Status: StatusFailed, Message: err.Error()})
		report.add(Check{Name: "AMD hardware encode probe", Status: StatusFailed, Message: "no supported encoder was selected"})
		return report
	}
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
		if err := checker.probeEncoder(ctx, runner, ffmpegPath, encoder, ""); err == nil {
			report.add(Check{Name: "AMD hardware encode probe", Status: StatusOK, Message: "a temporary AMF encode succeeded"})
			return report
		} else {
			probeErrors = append(probeErrors, err.Error())
		}
	} else {
		for _, device := range devices {
			if err := checker.probeEncoder(ctx, runner, ffmpegPath, encoder, device); err == nil {
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
	return report
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

func encoderName(platform, codec string) (string, error) {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if codec != "h264" && codec != "hevc" {
		return "", fmt.Errorf("unsupported video codec %q; choose h264 or hevc", codec)
	}
	suffix := "_amf"
	if platform == "linux" {
		suffix = "_vaapi"
	} else if platform != "windows" {
		return "", fmt.Errorf("platform %q is not supported", platform)
	}
	if codec == "h264" {
		return "h264" + suffix, nil
	}
	return "hevc" + suffix, nil
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

func (checker Checker) probeEncoder(ctx context.Context, runner Runner, ffmpegPath, encoder, device string) error {
	tempDir, err := os.MkdirTemp(checker.TempDir, "vcx-encoder-probe-")
	if err != nil {
		return fmt.Errorf("create temporary probe directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	outputPath := filepath.Join(tempDir, "probe.mp4")

	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if device != "" {
		args = append(args, "-vaapi_device", device)
	}
	args = append(args,
		"-f", "lavfi",
		"-i", "color=c=black:s=64x64:d=0.1",
		"-frames:v", "1",
		"-an",
	)
	if device != "" {
		args = append(args, "-vf", "format=nv12,hwupload")
	}
	args = append(args, "-c:v", encoder)
	if device != "" {
		args = append(args, "-qp", "32")
	}
	args = append(args, "-f", "mp4", outputPath)

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

type execRunner struct{}

func (execRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (execRunner) Run(ctx context.Context, executable string, args ...string) ([]byte, []byte, error) {
	command := exec.CommandContext(ctx, executable, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
