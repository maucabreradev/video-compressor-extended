package diagnostics

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListedEncoderMustPassRealEncodeProbe(t *testing.T) {
	runner := newFakeRunner()
	runner.encoderList = " V..... h264_amf AMD H.264 Encoder\n"
	runner.probeError = errors.New("AMF device initialization failed")
	checker := Checker{Runner: runner, Platform: "windows", TempDir: t.TempDir()}

	report := checker.Diagnose(context.Background(), Options{
		VideoCodec:       "h264",
		WhisperModelPath: createModelFile(t),
	})

	if got := findCheck(t, report, "Requested encoder").Status; got != StatusOK {
		t.Fatalf("expected listed encoder status %q, got %q", StatusOK, got)
	}
	probeCheck := findCheck(t, report, "AMD hardware encode probe")
	if probeCheck.Status != StatusFailed {
		t.Fatalf("expected failed hardware probe, got %q", probeCheck.Status)
	}
	if !strings.Contains(probeCheck.Message, "AMD display driver") {
		t.Fatalf("probe failure lacks AMD troubleshooting guidance: %q", probeCheck.Message)
	}
	if !report.HasFailures() {
		t.Fatal("a listed but unusable encoder must make diagnostics fail")
	}
}

func TestWriteReportIncludesCheckStatusAndDetails(t *testing.T) {
	report := Report{Checks: []Check{{Name: "FFmpeg", Status: StatusOK, Message: "C:/tools/ffmpeg.exe"}}}
	var output strings.Builder
	if err := report.WriteReport(&output); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if !strings.Contains(output.String(), "FFmpeg") || !strings.Contains(output.String(), "C:/tools/ffmpeg.exe") {
		t.Fatalf("report omitted check details: %q", output.String())
	}
}

func TestWindowsProbeUsesAMFAndOnlyTemporaryOutput(t *testing.T) {
	runner := newFakeRunner()
	runner.encoderList = " V..... h264_amf AMD H.264 Encoder\n"
	tempDir := t.TempDir()
	checker := Checker{Runner: runner, Platform: "windows", TempDir: tempDir}

	report := checker.Diagnose(context.Background(), Options{
		VideoCodec:       "h264",
		WhisperModelPath: createModelFile(t),
	})

	if report.HasFailures() {
		t.Fatalf("expected healthy diagnostics, got %+v", report.Checks)
	}
	if runner.probeCalls != 1 || !containsPair(runner.probeArgs, "-c:v", "h264_amf") {
		t.Fatalf("expected an AMF encode probe, got %#v", runner.probeArgs)
	}
	output := runner.probeArgs[len(runner.probeArgs)-1]
	if !strings.HasPrefix(output, tempDir) {
		t.Fatalf("probe output escaped the temporary directory: %q", output)
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("probe left temporary output behind: %#v", entries)
	}
}

func TestLinuxProbeTargetsAMDVAAPIDevice(t *testing.T) {
	runner := newFakeRunner()
	runner.encoderList = " V..... hevc_vaapi H.265/HEVC (VAAPI)\n"
	device := "/dev/dri/renderD128"
	checker := Checker{
		Runner:   runner,
		Platform: "linux",
		TempDir:  t.TempDir(),
		AMDDeviceFinder: func() ([]string, error) {
			return []string{device}, nil
		},
	}

	report := checker.Diagnose(context.Background(), Options{
		VideoCodec:       "hevc",
		WhisperModelPath: createModelFile(t),
	})
	if report.HasFailures() {
		t.Fatalf("expected healthy diagnostics, got %+v", report.Checks)
	}
	if !containsPair(runner.probeArgs, "-vaapi_device", device) ||
		!containsPair(runner.probeArgs, "-c:v", "hevc_vaapi") {
		t.Fatalf("expected HEVC VAAPI probe on %s, got %#v", device, runner.probeArgs)
	}
}

func TestLinuxDoesNotProbeVAAPIWithoutAnAMDDevice(t *testing.T) {
	runner := newFakeRunner()
	runner.encoderList = " V..... h264_vaapi H.264/AVC (VAAPI)\n"
	checker := Checker{
		Runner:   runner,
		Platform: "linux",
		AMDDeviceFinder: func() ([]string, error) {
			return nil, nil
		},
	}

	report := checker.Diagnose(context.Background(), Options{VideoCodec: "h264"})
	if got := findCheck(t, report, "AMD hardware encode probe").Status; got != StatusMissing {
		t.Fatalf("expected missing AMD device status, got %q", got)
	}
	if runner.probeCalls != 0 {
		t.Fatalf("probe ran without an AMD device: %#v", runner.probeArgs)
	}
}

func TestMissingExecutablesAndModelAreReported(t *testing.T) {
	runner := &fakeRunner{paths: map[string]string{}}
	checker := Checker{Runner: runner, Platform: "windows"}
	missingModel := filepath.Join(t.TempDir(), "missing-model.bin")

	report := checker.Diagnose(context.Background(), Options{
		VideoCodec:       "h264",
		WhisperModelPath: missingModel,
	})
	for _, name := range []string{"FFmpeg executable", "FFprobe executable", "whisper.cpp executable", "Whisper model"} {
		if got := findCheck(t, report, name).Status; got != StatusMissing {
			t.Errorf("expected %q status for %s, got %q", StatusMissing, name, got)
		}
	}
	if !report.HasFailures() {
		t.Fatal("missing dependencies must make diagnostics fail")
	}
}

func TestMissingDependencyMessagesExplainHowToFixTheIssue(t *testing.T) {
	runner := &fakeRunner{paths: map[string]string{}}
	checker := Checker{Runner: runner, Platform: "windows"}
	modelPath := filepath.Join(t.TempDir(), "missing-model.bin")
	report := checker.Diagnose(context.Background(), Options{
		VideoCodec:       "h264",
		WhisperModelPath: modelPath,
	})

	ffmpegMessage := findCheck(t, report, "FFmpeg executable").Message
	if !strings.Contains(ffmpegMessage, "install") || !strings.Contains(ffmpegMessage, "PATH") {
		t.Fatalf("FFmpeg error does not explain how to fix it: %q", ffmpegMessage)
	}
	modelMessage := findCheck(t, report, "Whisper model").Message
	if !strings.Contains(modelMessage, "whisper_model_path") {
		t.Fatalf("model error does not explain how to configure the model: %q", modelMessage)
	}
}

type fakeRunner struct {
	paths       map[string]string
	encoderList string
	probeError  error
	probeCalls  int
	probeArgs   []string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{paths: map[string]string{
		"ffmpeg":      "test-bin/ffmpeg",
		"ffprobe":     "test-bin/ffprobe",
		"whisper-cli": "test-bin/whisper-cli",
	}}
}

func (runner *fakeRunner) LookPath(name string) (string, error) {
	path, ok := runner.paths[name]
	if !ok {
		return "", errors.New("executable not found")
	}
	return path, nil
}

func (runner *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, []byte, error) {
	if contains(args, "-encoders") {
		return []byte(runner.encoderList), nil, nil
	}
	runner.probeCalls++
	runner.probeArgs = append([]string(nil), args...)
	if runner.probeError != nil {
		return nil, []byte(runner.probeError.Error()), runner.probeError
	}
	if len(args) == 0 {
		return nil, nil, errors.New("probe output path is missing")
	}
	if err := os.WriteFile(args[len(args)-1], []byte("temporary encoded media"), 0o600); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func createModelFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model.bin")
	if err := os.WriteFile(path, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func findCheck(t *testing.T, report Report, name string) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("diagnostic report has no check named %q: %+v", name, report.Checks)
	return Check{}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsPair(values []string, first, second string) bool {
	for index := 0; index+1 < len(values); index++ {
		if values[index] == first && values[index+1] == second {
			return true
		}
	}
	return false
}
