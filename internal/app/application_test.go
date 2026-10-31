package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/jobmanifest"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
	"github.com/maucabreradev/video-compressor-extended/internal/workspace"
)

func TestRunCompressesDiscoveredInputThroughVerifiedAMF(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "nested", "clip.mkv")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	delete(runner.paths, "whisper-cli")
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = "processed"

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression: %v", err)
	}
	encoded := filepath.Join(inputRoot, "processed", "nested", "clip.mp4")
	if contents, err := os.ReadFile(encoded); err != nil || string(contents) != "encoded output" {
		t.Fatalf("expected validated output at %q, contents=%q err=%v", encoded, contents, err)
	}
	if !runner.usedEncoder("h264_amf") {
		t.Fatalf("run did not use the verified AMF encoder: %#v", runner.commands)
	}
}

func TestRunCompressesLongOutputThroughShortWorkspaceAndPersistsCompressionStage(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.MkdirAll(inputRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	outputRoot := root
	for range 12 {
		outputRoot = filepath.Join(outputRoot, "Unicode café long output directory")
	}
	if len(outputRoot) <= 260 {
		t.Fatalf("test output path length = %d, want more than legacy MAX_PATH", len(outputRoot))
	}

	runner := newFakeRunner()
	delete(runner.paths, "whisper-cli")
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	workspaceRoot := filepath.Join(root, "w")
	application.workspaceManager = workspace.Manager{Roots: []string{workspaceRoot}}
	manifestRoot := filepath.Join(root, "jobs")
	store, err := jobmanifest.NewStore(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	application.jobManifests = store
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = outputRoot

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression through short workspace: %v", err)
	}
	finalOutput := filepath.Join(outputRoot, "clip.mp4")
	if contents, err := os.ReadFile(finalOutput); err != nil || string(contents) != "encoded output" {
		t.Fatalf("compressed output not published at planned long destination: contents=%q err=%v", contents, err)
	}

	var encodeOutput string
	for _, command := range runner.commands {
		if command.Executable == "fake-ffmpeg" && containsArg(command.Args, "-progress") {
			encodeOutput = command.Args[len(command.Args)-1]
			break
		}
	}
	if encodeOutput == "" || filepath.Dir(filepath.Dir(encodeOutput)) != workspaceRoot || filepath.Base(encodeOutput) != "video.mp4" {
		t.Fatalf("FFmpeg output %q is not the fixed short-workspace video artifact", encodeOutput)
	}
	if _, err := os.Stat(encodeOutput); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("short workspace artifact was not cleaned after publication: %v", err)
	}

	manifests := readJobManifestFiles(t, manifestRoot)
	if len(manifests) != 1 {
		t.Fatalf("manifest count = %d; want one job manifest", len(manifests))
	}
	manifest, err := store.Load(strings.TrimSuffix(manifests[0].Name(), filepath.Ext(manifests[0].Name())))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Items) != 1 || manifest.Items[0].Compression.State != jobmanifest.StagePublished {
		t.Fatalf("compression stage was not persisted as published: %+v", manifest.Items)
	}
}

func TestPreviewListsPlannedInputsAndRequiredDependencies(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "selected media")
	source := filepath.Join(inputRoot, "nested", "clip.mkv")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")

	preview, err := application.Preview(context.Background(), cfg)
	if err != nil {
		t.Fatalf("preview selected media: %v", err)
	}
	if len(preview.Plans) != 1 || preview.Plans[0].Source != source {
		t.Fatalf("preview plans = %+v, want selected source %q", preview.Plans, source)
	}
	if preview.Plans[0].Video != filepath.Join(root, "processed", "nested", "clip.mp4") {
		t.Fatalf("preview video destination = %q", preview.Plans[0].Video)
	}
	if preview.Diagnostics.HasFailures() {
		t.Fatalf("default compression dependencies should be ready: %+v", preview.Diagnostics.Checks)
	}
	if preview.SubtitlesEnabled {
		t.Fatal("subtitles should not be enabled without a Whisper model")
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("preview started media encoding %d times", runner.encodeCalls)
	}
}

func TestPreviewSupportsMultipleInputPathsWithLegacyRootAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "media root")
	alpha := filepath.Join(inputRoot, "alpha", "clip.mkv")
	beta := filepath.Join(inputRoot, "beta", "clip.mkv")
	for _, source := range []string{alpha, beta} {
		if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	configPath := filepath.Join(root, "vcx.toml")
	contents := fmt.Sprintf("input_root = %q\noutput_dir = %q\nsubtitles_enabled = false\ninput_paths = [%q, %q, %q]\n",
		inputRoot, filepath.Join(root, "processed"), filepath.Join("alpha"),
		filepath.Join("beta", "clip.mkv"), filepath.Join("alpha", "clip.mkv"))
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load multiple-input configuration: %v", err)
	}

	runner := newFakeRunner()
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	preview, err := application.Preview(context.Background(), cfg)
	if err != nil {
		t.Fatalf("preview multiple inputs: %v", err)
	}
	if len(preview.Plans) != 2 {
		t.Fatalf("preview planned %d inputs, want 2 deduplicated files: %+v", len(preview.Plans), preview.Plans)
	}
	wantVideo := map[string]bool{
		filepath.Join(root, "processed", "alpha", "clip.mp4"): false,
		filepath.Join(root, "processed", "beta", "clip.mp4"):  false,
	}
	for _, plan := range preview.Plans {
		if _, ok := wantVideo[plan.Video]; !ok {
			t.Errorf("unexpected output path %q", plan.Video)
		}
		wantVideo[plan.Video] = true
	}
	for output, found := range wantVideo {
		if !found {
			t.Errorf("missing distinct relative output %q", output)
		}
	}
}

func TestPreviewRejectsMultipleInputOutsideCommonRoot(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "selected")
	if err := os.Mkdir(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside", "clip.mp4")
	if err := os.MkdirAll(filepath.Dir(outside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.InputPaths = []string{outside}
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.SubtitlesEnabled = false
	application := newTestApplication(t, newFakeRunner(), nil)
	application.platform = "windows"

	if _, err := application.Preview(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "outside input root") {
		t.Fatalf("preview should reject paths outside the common input root, got %v", err)
	}
}

func TestPreviewRejectsMultipleInputsWithCollidingOutputs(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	for _, name := range []string{"clip.mkv", "clip.avi"} {
		path := filepath.Join(inputRoot, "same", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("input"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.InputPaths = []string{filepath.Join("same", "clip.mkv"), filepath.Join("same", "clip.avi")}
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.SubtitlesEnabled = false
	application := newTestApplication(t, newFakeRunner(), nil)
	application.platform = "windows"

	if _, err := application.Preview(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "output collision") {
		t.Fatalf("preview should reject colliding output plans, got %v", err)
	}
}

func TestRunProcessesOnlyExplicitInputPaths(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	sources := []string{
		filepath.Join(inputRoot, "alpha", "clip.mkv"),
		filepath.Join(inputRoot, "beta", "clip.mkv"),
		filepath.Join(inputRoot, "not-selected.mp4"),
	}
	for _, source := range sources {
		if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.InputPaths = []string{filepath.Join("alpha"), filepath.Join("beta", "clip.mkv")}
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.SubtitlesEnabled = false
	application := newTestApplication(t, newFakeRunner(), nil)
	application.platform = "windows"

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run explicitly selected inputs: %v", err)
	}
	for _, relative := range []string{filepath.Join("alpha", "clip.mp4"), filepath.Join("beta", "clip.mp4")} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, relative)); err != nil {
			t.Errorf("selected input output %q is missing: %v", relative, err)
		}
	}
	if _, err := os.Stat(filepath.Join(cfg.OutputDir, "not-selected.mp4")); !os.IsNotExist(err) {
		t.Errorf("unselected input produced an output: err=%v", err)
	}
}

func TestPreviewOffersSoftwareFallbackWithoutStartingMedia(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "clip.mp4")
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	runner.encoderList = " V..... libx264 H.264 / AVC / MPEG-4 AVC\n"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = source
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.SubtitlesEnabled = false

	preview, err := application.Preview(context.Background(), cfg)
	if err != nil {
		t.Fatalf("preview software fallback: %v", err)
	}
	if !preview.SoftwareFallbackAvailable || !preview.SoftwareFallbackConfirmationRequired {
		t.Fatalf("preview did not disclose confirmation-gated fallback: %+v", preview)
	}
	if preview.EncoderBackend != "software" || preview.EncoderName != "libx264" {
		t.Fatalf("fallback selection = %q/%q, want software/libx264", preview.EncoderBackend, preview.EncoderName)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("preview started media encoding %d times", runner.encodeCalls)
	}
}

func TestPreviewPreparesManagedDefaultModelAndTools(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "clip.mp4")
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(root, "runtime with spaces")
	modelPath := filepath.Join(managed, "ggml-base.bin")
	if err := os.MkdirAll(managed, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{
		"ffmpeg":      filepath.Join(managed, "ffmpeg.exe"),
		"ffprobe":     filepath.Join(managed, "ffprobe.exe"),
		"whisper-cli": filepath.Join(managed, "whisper-cli.exe"),
		"model":       modelPath,
	}
	runner := newFakeRunner()
	for _, path := range paths {
		runner.paths[path] = path
	}
	manager := &fakeRuntimeAssetManager{assets: map[string]runtimeassets.PreparedAsset{
		"ffmpeg":        {Files: map[string]string{"ffmpeg": paths["ffmpeg"], "ffprobe": paths["ffprobe"]}},
		"whisper-cli":   {Files: map[string]string{"whisper-cli": paths["whisper-cli"]}},
		"whisper-model": {Files: map[string]string{"model": paths["model"]}},
	}}
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	application.SetRuntimeAssetManager(manager)
	cfg := config.Defaults()
	cfg.InputRoot = source
	cfg.OutputDir = filepath.Join(root, "processed")

	preview, err := application.Preview(context.Background(), cfg)
	if err != nil {
		t.Fatalf("preview managed defaults: %v", err)
	}
	if !preview.SubtitlesEnabled {
		t.Fatal("managed default model should keep subtitles enabled")
	}
	if strings.Join(manager.calls, ",") != "ffmpeg,whisper-cli,whisper-model" {
		t.Fatalf("prepared assets = %#v", manager.calls)
	}
}

func TestPreviewWithCallbacksForwardsRuntimeConsentAndDownloadProgress(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "clip.mp4")
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	manager := &fakeRuntimeAssetManager{
		assets: map[string]runtimeassets.PreparedAsset{
			"ffmpeg":        {Files: map[string]string{"ffmpeg": "fake-ffmpeg", "ffprobe": "fake-ffprobe"}},
			"whisper-cli":   {Files: map[string]string{"whisper-cli": "fake-whisper"}},
			"whisper-model": {Files: map[string]string{"model": filepath.Join(root, "model.bin")}},
		},
		onCallbacks: func(_ string, consent func(runtimeassets.Asset) bool, progress func(runtimeassets.DownloadProgress)) {
			asset := runtimeassets.Asset{ID: "whisper-model", Version: "1.0", SourceURL: "https://example.com/source", License: "MIT", SizeBytes: 1024}
			if consent != nil && consent(asset) && progress != nil {
				progress(runtimeassets.DownloadProgress{AssetID: asset.ID, ReceivedBytes: 512, TotalBytes: asset.SizeBytes})
			}
		},
	}
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	application.SetRuntimeAssetManager(manager)
	cfg := config.Defaults()
	cfg.InputRoot = source
	cfg.OutputDir = filepath.Join(root, "processed")
	var consented runtimeassets.Asset
	var progress runtimeassets.DownloadProgress
	preview, err := application.PreviewWithCallbacks(context.Background(), cfg, RuntimeSetupCallbacks{
		Consent:  func(asset runtimeassets.Asset) bool { consented = asset; return true },
		Progress: func(update runtimeassets.DownloadProgress) { progress = update },
	})
	if err != nil {
		t.Fatalf("preview with setup callbacks: %v", err)
	}
	if !preview.SubtitlesEnabled || consented.ID != "whisper-model" || consented.SizeBytes != 1024 {
		t.Fatalf("consent callback was not forwarded: preview=%+v asset=%+v", preview, consented)
	}
	if progress.AssetID != "whisper-model" || progress.ReceivedBytes != 512 || progress.TotalBytes != 1024 {
		t.Fatalf("download progress was not forwarded: %+v", progress)
	}
}

func TestPreviewRejectsInvalidSettingsBeforeDiagnostics(t *testing.T) {
	runner := newFakeRunner()
	application := newTestApplication(t, runner, nil)
	cfg := config.Defaults()
	cfg.EncodeWorkers = 0

	if _, err := application.Preview(context.Background(), cfg); err == nil {
		t.Fatal("preview accepted an invalid worker limit")
	}
	if runner.diagnosticCalls != 0 {
		t.Fatalf("invalid settings ran %d diagnostic commands", runner.diagnosticCalls)
	}
}

func TestPreviewReportsMissingDependenciesWithoutStartingMedia(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "clip.mkv")
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	delete(runner.paths, "ffmpeg")
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = source
	cfg.OutputDir = filepath.Join(root, "processed")

	preview, err := application.Preview(context.Background(), cfg)
	if err != nil {
		t.Fatalf("preview should return dependency findings for review: %v", err)
	}
	if !preview.Diagnostics.HasFailures() {
		t.Fatalf("missing FFmpeg was not reported as a failed preflight: %+v", preview.Diagnostics.Checks)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("preflight started media encoding %d times", runner.encodeCalls)
	}
}

func TestApplicationRejectsUnsupportedPlatformsBeforeDiagnosticsOrMediaWork(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "clip.mkv")
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.InputRoot = source
	cfg.OutputDir = filepath.Join(root, "processed")

	tests := []struct {
		name string
		call func(*Application, config.Config) error
	}{
		{
			name: "doctor",
			call: func(application *Application, cfg config.Config) error {
				return application.Doctor(context.Background(), cfg)
			},
		},
		{
			name: "preview",
			call: func(application *Application, cfg config.Config) error {
				_, err := application.Preview(context.Background(), cfg)
				return err
			},
		},
		{
			name: "run",
			call: func(application *Application, cfg config.Config) error {
				return application.RunWithControl(context.Background(), cfg, nil, nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := newFakeRunner()
			application := newTestApplication(t, runner, nil)
			application.platform = "linux"

			err := test.call(application, cfg)
			if err == nil || !strings.Contains(err.Error(), "Windows only") {
				t.Fatalf("unsupported platform error = %v, want a clear Windows-only message", err)
			}
			if runner.diagnosticCalls != 0 || runner.encodeCalls != 0 {
				t.Fatalf("unsupported platform performed work: diagnostics=%d encodes=%d", runner.diagnosticCalls, runner.encodeCalls)
			}
		})
	}
}

func TestCheckPlatformProvidesAnActionableNonWindowsError(t *testing.T) {
	application := newTestApplication(t, newFakeRunner(), nil)
	application.platform = "linux"

	err := application.CheckPlatform()
	if err == nil || !strings.Contains(err.Error(), "Windows only") || !strings.Contains(err.Error(), "linux") {
		t.Fatalf("platform error = %v, want an actionable Windows-only message", err)
	}
}

func TestRunEmitsStructuredLifecycleAndProgressEvents(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.progressEvents = 2
	var output bytes.Buffer
	application := newTestApplication(t, runner, &output)
	application.platform = "windows"
	var events []RunEvent
	application.SetRunEventHandler(func(event RunEvent) error {
		events = append(events, event)
		return nil
	})
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression: %v", err)
	}
	if len(events) != 8 {
		t.Fatalf("expected queued/running/completed stage states, start, three progress, and file completion events; got %#v", events)
	}
	if events[0].Kind != RunStageStateChanged || events[0].Stage != scheduler.Compression || events[0].State != scheduler.Queued {
		t.Fatalf("unexpected queued stage event: %+v", events[0])
	}
	if events[1].Kind != RunStarted || events[1].Source != source {
		t.Fatalf("unexpected start event: %+v", events[1])
	}
	if events[2].Kind != RunStageStateChanged || events[2].State != scheduler.Running {
		t.Fatalf("unexpected running stage event: %+v", events[2])
	}
	firstProgress := events[3]
	if firstProgress.Kind != RunProgress || firstProgress.Source != source || !firstProgress.DurationKnown ||
		firstProgress.Progress.Frame != 1 || firstProgress.Progress.OutTime != time.Second || firstProgress.Progress.Percent != 10 {
		t.Fatalf("unexpected first progress event: %+v", firstProgress)
	}
	lastProgress := events[5]
	if lastProgress.Kind != RunProgress || !lastProgress.Progress.Done || lastProgress.Progress.Percent != 100 {
		t.Fatalf("unexpected final progress event: %+v", lastProgress)
	}
	if events[6].Kind != RunStageStateChanged || events[6].State != scheduler.Completed {
		t.Fatalf("unexpected completed stage event: %+v", events[6])
	}
	if events[7].Kind != RunCompleted || events[7].Destination != filepath.Join(root, "processed", "clip.mp4") {
		t.Fatalf("unexpected file completion event: %+v", events[7])
	}
	if output.Len() != 0 {
		t.Fatalf("application rendered batch text instead of emitting events: %q", output.String())
	}
}

func TestRunGeneratesSRTFromOriginalInput(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input café")
	source := filepath.Join(inputRoot, "clip original.mkv")
	modelPath := filepath.Join(root, "ggml-model.bin")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nHello from the source.\n"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.WhisperModelPath = modelPath
	cfg.Language = "es"
	cfg.WhisperThreads = 4
	cfg.ResourceProfile = config.ResourceProfileCustom
	cfg.ProcessPriority = config.ProcessPriorityNormal
	cfg.WhisperGPU = true

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run compression and subtitles: %v", err)
	}
	plan, err := outputplan.PlanOutputs(inputRoot, cfg.OutputDir, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if contents, err := os.ReadFile(plan[0].Subtitle); err != nil || string(contents) != runner.whisperTranscript {
		t.Fatalf("expected validated SRT, contents=%q err=%v", contents, err)
	}
	if len(runner.audioInputs) != 1 || runner.audioInputs[0] != source {
		t.Fatalf("audio extraction did not use the original input: %#v", runner.audioInputs)
	}
	if !containsPair(runner.whisperArgs, "-m", modelPath) || !containsPair(runner.whisperArgs, "-l", "es") ||
		!containsPair(runner.whisperArgs, "-t", "4") || containsArg(runner.whisperArgs, "-ng") {
		t.Fatalf("Whisper configuration was not passed to whisper.cpp: %#v", runner.whisperArgs)
	}
	for _, command := range runner.commands {
		if command.Executable == "fake-ffmpeg" || command.Executable == "fake-whisper" {
			if command.Priority != process.PriorityNormal {
				t.Errorf("configured process priority was not propagated: %+v", command)
			}
		}
	}
}

func TestRunPreservesSubtitleStageWhenCompressionFails(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "ggml-model.bin")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.encodeError = errors.New("encoder failed")
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nSubtitle survives.\n"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	workspaceRoot := filepath.Join(root, "workspaces")
	application.workspaceManager = workspace.Manager{Roots: []string{workspaceRoot}}
	store, err := jobmanifest.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	application.jobManifests = store
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.WhisperModelPath = modelPath

	err = application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "compress") {
		t.Fatalf("expected compression failure to be reported, got %v", err)
	}
	plan, err := outputplan.PlanOutputs(inputRoot, cfg.OutputDir, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if contents, err := os.ReadFile(plan[0].Subtitle); err != nil || string(contents) != runner.whisperTranscript {
		t.Fatalf("subtitle stage did not survive compression failure: contents=%q err=%v", contents, err)
	}
	manifests := readJobManifestFiles(t, filepath.Join(root, "jobs"))
	if len(manifests) != 1 {
		t.Fatalf("manifest count = %d; want one", len(manifests))
	}
	manifest, err := store.Load(strings.TrimSuffix(manifests[0].Name(), filepath.Ext(manifests[0].Name())))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Items[0].Compression.State != jobmanifest.StageFailed || manifest.Items[0].Subtitles.State != jobmanifest.StagePublished {
		t.Fatalf("compression/subtitle failure states are not independent: %+v", manifest.Items[0])
	}
	workspaces, err := os.ReadDir(workspaceRoot)
	if err != nil || len(workspaces) != 0 {
		t.Fatalf("failed compression workspace was not cleaned: entries=%v err=%v", workspaces, err)
	}
	if !containsArg(runner.whisperArgs, "-ng") || !containsPair(runner.whisperArgs, "-t", "1") {
		t.Fatalf("conservative profile did not keep Whisper CPU-only with one thread: %#v", runner.whisperArgs)
	}
	for _, command := range runner.commands {
		if command.Executable == "fake-ffmpeg" || command.Executable == "fake-whisper" {
			if command.Priority != process.PriorityBelowNormal {
				t.Errorf("conservative profile did not request best-effort below-normal priority: %+v", command)
			}
		}
	}
}

func TestRunSchedulesCompressionAndTranscriptionIndependently(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "ggml-model.bin")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{source: "original media", modelPath: "model"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fake := newFakeRunner()
	fake.paths["whisper-cli"] = "fake-whisper"
	fake.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nConcurrent stages.\n"
	started := make(chan string, 2)
	release := make(chan struct{})
	runner := &gatedRunner{fakeRunner: fake, started: started, release: release}
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	var eventsMu sync.Mutex
	var events []RunEvent
	application.SetRunEventHandler(func(event RunEvent) error {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		events = append(events, event)
		return nil
	})
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.WhisperModelPath = modelPath

	runErr := make(chan error, 1)
	go func() { runErr <- application.Run(context.Background(), cfg) }()
	startedStages := map[string]bool{}
	for range 2 {
		select {
		case stage := <-started:
			startedStages[stage] = true
		case <-time.After(5 * time.Second):
			close(release)
			<-runErr
			t.Fatalf("compression and transcription did not overlap; started=%v", startedStages)
		}
	}
	close(release)
	if err := <-runErr; err != nil {
		t.Fatalf("run independent stages: %v", err)
	}
	if !startedStages["compression"] || !startedStages["transcription"] {
		t.Fatalf("both stages did not start concurrently: %v", startedStages)
	}

	states := map[scheduler.Stage]map[scheduler.State]bool{}
	eventsMu.Lock()
	defer eventsMu.Unlock()
	for _, event := range events {
		if event.Kind != RunStageStateChanged {
			continue
		}
		if states[event.Stage] == nil {
			states[event.Stage] = map[scheduler.State]bool{}
		}
		states[event.Stage][event.State] = true
	}
	for _, stage := range []scheduler.Stage{scheduler.Compression, scheduler.Transcription} {
		for _, state := range []scheduler.State{scheduler.Queued, scheduler.Running, scheduler.Completed} {
			if !states[stage][state] {
				t.Errorf("missing %s state %s in events: %+v", stage, state, events)
			}
		}
	}
}

func TestRunSkipsOnlyValidatedOutputsOnResume(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	outputRoot := filepath.Join(root, "processed")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		source: "original media", modelPath: "model",
		filepath.Join(outputRoot, "clip.mp4"): "valid video file",
		filepath.Join(outputRoot, "clip.srt"): "1\n00:00:00,000 --> 00:00:01,000\nExisting caption.\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = outputRoot
	cfg.WhisperModelPath = modelPath
	var skipped []scheduler.Stage
	application.SetRunEventHandler(func(event RunEvent) error {
		if event.Kind == RunStageStateChanged && event.State == scheduler.Skipped {
			skipped = append(skipped, event.Stage)
		}
		return nil
	})

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("resume completed outputs: %v", err)
	}
	if runner.encodeCalls != 0 || len(runner.audioInputs) != 0 {
		t.Fatalf("valid outputs were processed again: encode calls=%d audio inputs=%v", runner.encodeCalls, runner.audioInputs)
	}
	skippedStages := map[scheduler.Stage]int{}
	for _, stage := range skipped {
		skippedStages[stage]++
	}
	if len(skipped) != 2 || skippedStages[scheduler.Compression] != 1 || skippedStages[scheduler.Transcription] != 1 {
		t.Fatalf("validated stages were not reported as skipped: %v", skipped)
	}
}

func TestRunRetriesInvalidVideoWithoutDiscardingValidSubtitle(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	outputRoot := filepath.Join(root, "processed")
	videoPath := filepath.Join(outputRoot, "clip.mp4")
	subtitlePath := filepath.Join(outputRoot, "clip.srt")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		source: "original media", modelPath: "model",
		videoPath:    "invalid video bytes",
		subtitlePath: "1\n00:00:00,000 --> 00:00:01,000\nValid existing caption.\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	baseRunner := newFakeRunner()
	baseRunner.paths["whisper-cli"] = "fake-whisper"
	runner := &invalidVideoRunner{fakeRunner: baseRunner, invalidPaths: map[string]bool{videoPath: true}}
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = outputRoot
	cfg.WhisperModelPath = modelPath
	var skipped []scheduler.Stage
	application.SetRunEventHandler(func(event RunEvent) error {
		if event.Kind == RunStageStateChanged && event.State == scheduler.Skipped {
			skipped = append(skipped, event.Stage)
		}
		return nil
	})

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("retry invalid video: %v", err)
	}
	retryVideo := filepath.Join(outputRoot, "clip.vcx-retry-1.mp4")
	if contents, err := os.ReadFile(videoPath); err != nil || string(contents) != "invalid video bytes" {
		t.Fatalf("invalid existing video was not preserved: contents=%q err=%v", contents, err)
	}
	if contents, err := os.ReadFile(retryVideo); err != nil || string(contents) != "encoded output" {
		t.Fatalf("video retry was not published: contents=%q err=%v", contents, err)
	}
	if baseRunner.encodeCalls != 1 || len(baseRunner.audioInputs) != 0 {
		t.Fatalf("only the invalid stage should be retried: encodes=%d audio=%v", baseRunner.encodeCalls, baseRunner.audioInputs)
	}
	if len(skipped) != 1 || skipped[0] != scheduler.Transcription {
		t.Fatalf("valid subtitle stage was not skipped independently: %v", skipped)
	}
	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("resume validated retry output: %v", err)
	}
	if baseRunner.encodeCalls != 1 || len(baseRunner.audioInputs) != 0 {
		t.Fatalf("validated retry outputs were not reused: encodes=%d audio=%v", baseRunner.encodeCalls, baseRunner.audioInputs)
	}
}

func TestRunRetriesInvalidSubtitleWithoutDiscardingValidVideo(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	outputRoot := filepath.Join(root, "processed")
	videoPath := filepath.Join(outputRoot, "clip.mp4")
	subtitlePath := filepath.Join(outputRoot, "clip.srt")
	for path, contents := range map[string]string{
		source: "original media", modelPath: "model",
		videoPath: "valid video file", subtitlePath: "invalid subtitle bytes",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nRegenerated caption.\n"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = outputRoot
	cfg.WhisperModelPath = modelPath
	var skipped []scheduler.Stage
	application.SetRunEventHandler(func(event RunEvent) error {
		if event.Kind == RunStageStateChanged && event.State == scheduler.Skipped {
			skipped = append(skipped, event.Stage)
		}
		return nil
	})

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("retry invalid subtitle: %v", err)
	}
	retrySubtitle := filepath.Join(outputRoot, "clip.vcx-retry-1.srt")
	if contents, err := os.ReadFile(subtitlePath); err != nil || string(contents) != "invalid subtitle bytes" {
		t.Fatalf("invalid existing subtitle was not preserved: contents=%q err=%v", contents, err)
	}
	if contents, err := os.ReadFile(retrySubtitle); err != nil || string(contents) != runner.whisperTranscript {
		t.Fatalf("subtitle retry was not published: contents=%q err=%v", contents, err)
	}
	if runner.encodeCalls != 0 || len(runner.audioInputs) != 1 {
		t.Fatalf("only the invalid stage should be retried: encodes=%d audio=%v", runner.encodeCalls, runner.audioInputs)
	}
	if len(skipped) != 1 || skipped[0] != scheduler.Compression {
		t.Fatalf("valid video stage was not skipped independently: %v", skipped)
	}
}

func TestRunReturnsCancellationWhenNoQueuedStageCanStart(t *testing.T) {
	inputRoot := t.TempDir()
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(t.TempDir(), "processed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := application.Run(ctx, cfg)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run with cancelled context returned %v, want context.Canceled", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("compression started after cancellation: %d calls", runner.encodeCalls)
	}
	if runner.diagnosticCalls != 0 {
		t.Fatalf("diagnostics ran after cancellation: %d calls", runner.diagnosticCalls)
	}
}

func TestRunReturnsCancellationIfDiagnosticsAreInterrupted(t *testing.T) {
	inputRoot := t.TempDir()
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &cancelDiagnosticsRunner{fakeRunner: newFakeRunner(), cancel: cancel}
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(t.TempDir(), "processed")

	err := application.Run(ctx, cfg)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run error = %v, want context.Canceled", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("processing started after diagnostics were cancelled: %d encodes", runner.encodeCalls)
	}
}

func TestRunWithControlPausesCompressionDispatch(t *testing.T) {
	inputRoot := t.TempDir()
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(t.TempDir(), "processed")
	control := scheduler.NewDispatchControl()
	control.Pause()
	queued := make(chan struct{}, 1)
	result := make(chan error, 1)
	go func() {
		result <- application.RunWithControl(context.Background(), cfg, control, func(event RunEvent) error {
			if event.Kind == RunStageStateChanged && event.State == scheduler.Queued {
				queued <- struct{}{}
			}
			return nil
		})
	}()
	select {
	case <-queued:
	case <-time.After(5 * time.Second):
		control.Resume()
		t.Fatal("application did not queue its compression stage")
	}
	select {
	case err := <-result:
		control.Resume()
		t.Fatalf("run finished while paused: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if runner.encodeCalls != 0 {
		control.Resume()
		t.Fatalf("compression dispatched while paused: %d calls", runner.encodeCalls)
	}
	control.Resume()
	if err := <-result; err != nil {
		t.Fatalf("resume processing: %v", err)
	}
	if runner.encodeCalls != 1 {
		t.Fatalf("compression did not resume: %d calls", runner.encodeCalls)
	}
}

func TestRunCancellationPreservesCompletedStageAndOriginal(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	outputRoot := filepath.Join(root, "processed")
	for path, contents := range map[string]string{source: "original media", modelPath: "model"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fake := newFakeRunner()
	fake.paths["whisper-cli"] = "fake-whisper"
	fake.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nSubtitle survives cancellation.\n"
	runner := &blockingAppEncodeRunner{fakeRunner: fake, started: make(chan struct{})}
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transcriptionCompleted := make(chan struct{}, 1)
	application.SetRunEventHandler(func(event RunEvent) error {
		if event.Kind == RunStageStateChanged && event.Stage == scheduler.Transcription && event.State == scheduler.Completed {
			transcriptionCompleted <- struct{}{}
		}
		return nil
	})
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = outputRoot
	cfg.WhisperModelPath = modelPath
	runErr := make(chan error, 1)
	go func() { runErr <- application.Run(ctx, cfg) }()

	select {
	case <-runner.started:
	case <-time.After(5 * time.Second):
		cancel()
		<-runErr
		t.Fatal("compression did not reach the active process")
	}
	select {
	case <-transcriptionCompleted:
	case <-time.After(5 * time.Second):
		cancel()
		<-runErr
		t.Fatal("transcription did not complete before cancellation")
	}
	cancel()
	if err := <-runErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("run error = %v, want context.Canceled", err)
	}
	if contents, err := os.ReadFile(source); err != nil || string(contents) != "original media" {
		t.Fatalf("original source changed: contents=%q err=%v", contents, err)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "clip.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled video was published: %v", err)
	}
	if contents, err := os.ReadFile(filepath.Join(outputRoot, "clip.srt")); err != nil || string(contents) != fake.whisperTranscript {
		t.Fatalf("completed subtitle did not survive cancellation: contents=%q err=%v", contents, err)
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "clip.srt" {
		t.Fatalf("incomplete video temporary file remains: %#v", entries)
	}
}

func TestRunKeepsVideoWhenSubtitlePrerequisitesAreMissing(t *testing.T) {
	tests := []struct {
		name           string
		whisperMissing bool
		modelMissing   bool
		wantError      string
	}{
		{name: "whisper executable", whisperMissing: true, wantError: "whisper-cli"},
		{name: "model file", modelMissing: true, wantError: "model"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			inputRoot := filepath.Join(root, "input")
			source := filepath.Join(inputRoot, "clip.mkv")
			if err := os.MkdirAll(inputRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, []byte("original media"), 0o600); err != nil {
				t.Fatal(err)
			}

			runner := newFakeRunner()
			runner.paths["whisper-cli"] = "fake-whisper"
			modelPath := filepath.Join(root, "ggml-model.bin")
			if !test.modelMissing {
				if err := os.WriteFile(modelPath, []byte("model"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.whisperMissing {
				delete(runner.paths, "whisper-cli")
			}
			application := newTestApplication(t, runner, nil)
			application.platform = "windows"
			cfg := config.Defaults()
			cfg.InputRoot = inputRoot
			cfg.OutputDir = filepath.Join(root, "processed")
			cfg.WhisperModelPath = modelPath

			err := application.Run(context.Background(), cfg)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected subtitle prerequisite error containing %q, got %v", test.wantError, err)
			}
			videoOutput := filepath.Join(root, "processed", "clip.mp4")
			if contents, err := os.ReadFile(videoOutput); err != nil || string(contents) != "encoded output" {
				t.Fatalf("video output was lost after subtitle prerequisite failure: contents=%q err=%v", contents, err)
			}
		})
	}
}

func TestRunRequiresConfirmationBeforeSoftwareH264Fallback(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mp4")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeRunner()
	runner.encoderList = " V..... libx264 H.264 / AVC / MPEG-4 AVC\n"
	application := newTestApplication(t, runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "software H.264") || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("expected explicit software fallback confirmation error, got %v", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("software compression ran before confirmation: %#v", runner.commands)
	}
}

func TestRunUsesSoftwareH264AfterExplicitConfirmation(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	if err := os.MkdirAll(inputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputRoot, "clip.mp4"), []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	runner.encoderList = " V..... libx264 H.264 / AVC / MPEG-4 AVC\n"
	application := newTestApplication(t, runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.SubtitlesEnabled = false
	cfg.AllowSoftwareEncoding = true
	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run confirmed software H.264 fallback: %v", err)
	}
	if !runner.usedEncoder("libx264") {
		t.Fatalf("confirmed software fallback did not use libx264: %#v", runner.commands)
	}
}

func TestRunCanDisableCompressionAndKeepTranscriptionEnabled(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	for path, contents := range map[string]string{source: "original media", modelPath: "model"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nSubtitle only.\n"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "processed")
	cfg.WhisperModelPath = modelPath
	cfg.CompressionEnabled = false
	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run transcription without compression: %v", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("compression ran when disabled: %d calls", runner.encodeCalls)
	}
	if len(runner.audioInputs) != 1 || runner.audioInputs[0] != source {
		t.Fatalf("transcription did not run independently: %#v", runner.audioInputs)
	}
}

func TestRunTranscribesLongOutputThroughShortWorkspaceAndPersistsSubtitleStage(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	if err := os.MkdirAll(inputRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{source: "original media", modelPath: "test model"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outputRoot := root
	for index := 0; index < 12; index++ {
		outputRoot = filepath.Join(outputRoot, "Unicode café long output directory")
	}
	if len(outputRoot) <= 260 {
		t.Fatalf("test output path length = %d, want more than legacy MAX_PATH", len(outputRoot))
	}
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.whisperTranscript = "1\n00:00:00,000 --> 00:00:01,000\nWorkspace subtitle.\n"
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	application.workspaceManager = workspace.Manager{Roots: []string{filepath.Join(root, "w")}}
	manifestRoot := filepath.Join(root, "jobs")
	store, err := jobmanifest.NewStore(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	application.jobManifests = store
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = outputRoot
	cfg.WhisperModelPath = modelPath
	cfg.CompressionEnabled = false

	if err := application.Run(context.Background(), cfg); err != nil {
		t.Fatalf("run subtitle generation through short workspace: %v", err)
	}
	whisperPrefix, ok := argValue(runner.whisperArgs, "-of")
	if !ok || filepath.Dir(whisperPrefix) != filepath.Join(root, "w") && !strings.HasPrefix(whisperPrefix, filepath.Join(root, "w")) {
		t.Fatalf("Whisper SRT prefix is not in the short workspace: %q", whisperPrefix)
	}
	whisperAudio, ok := argValue(runner.whisperArgs, "-f")
	if !ok || filepath.Dir(whisperAudio) != filepath.Dir(whisperPrefix) {
		t.Fatalf("Whisper audio path %q and SRT prefix %q do not share the workspace", whisperAudio, whisperPrefix)
	}
	if len(runner.audioInputs) != 1 || runner.audioInputs[0] != source {
		t.Fatal("audio extraction must read the source but write to a workspace path")
	}
	output := filepath.Join(outputRoot, "clip.srt")
	if got, err := os.ReadFile(output); err != nil || !strings.Contains(string(got), "Workspace subtitle.") {
		t.Fatalf("subtitle not published to planned long destination: content=%q err=%v", got, err)
	}
	entries := readJobManifestFiles(t, manifestRoot)
	if len(entries) != 1 {
		t.Fatalf("manifest count = %d; want one job manifest", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(manifestRoot, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var manifest jobmanifest.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Items) != 1 || manifest.Items[0].Subtitles.State != jobmanifest.StagePublished {
		t.Fatalf("subtitle stage was not persisted as published: %+v", manifest.Items)
	}
}

func TestRunPersistsFailedSubtitleStageAndCleansWorkspace(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	if err := os.MkdirAll(inputRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputRoot, "clip.mkv"), []byte("original media"), 0o600); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(root, "model.bin")
	if err := os.WriteFile(modelPath, []byte("test model"), 0o600); err != nil {
		t.Fatal(err)
	}
	workspaceRoot := filepath.Join(root, "workspaces")
	manifestRoot := filepath.Join(root, "jobs")
	store, err := jobmanifest.NewStore(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	runner.whisperError = errors.New("transcription failed")
	application := newTestApplication(t, runner, nil)
	application.platform = "windows"
	application.workspaceManager = workspace.Manager{Roots: []string{workspaceRoot}}
	application.jobManifests = store
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(root, "output")
	cfg.WhisperModelPath = modelPath
	cfg.CompressionEnabled = false

	if err := application.Run(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "transcription failed") {
		t.Fatalf("expected transcription failure, got %v", err)
	}
	manifests := readJobManifestFiles(t, manifestRoot)
	if len(manifests) != 1 {
		t.Fatalf("manifest count = %d; want one", len(manifests))
	}
	manifest, err := store.Load(strings.TrimSuffix(manifests[0].Name(), filepath.Ext(manifests[0].Name())))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Items[0].Subtitles.State != jobmanifest.StageFailed {
		t.Fatalf("subtitle failure state = %q, want failed", manifest.Items[0].Subtitles.State)
	}
	workspaces, err := os.ReadDir(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("failed subtitle workspace was not cleaned: %v", workspaces)
	}
}

func TestRunDoesNotProbeHardwareWhenNoMediaIsFound(t *testing.T) {
	inputRoot := t.TempDir()
	runner := newFakeRunner()
	application := newTestApplication(t, runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.OutputDir = filepath.Join(t.TempDir(), "processed")

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "no supported video files") {
		t.Fatalf("expected a no-input error, got %v", err)
	}
	if runner.diagnosticCalls != 0 {
		t.Fatalf("dependency or GPU probes ran without work: %d calls", runner.diagnosticCalls)
	}
}

func TestDoctorFailsWhenWhisperModelIsNotConfigured(t *testing.T) {
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	var output bytes.Buffer
	application := newTestApplication(t, runner, &output)
	application.platform = "windows"

	err := application.Doctor(context.Background(), config.Defaults())
	if err == nil {
		t.Fatalf("expected doctor to report incomplete subtitle readiness, report: %s", output.String())
	}
	if !strings.Contains(output.String(), "NOT CONFIGURED") || !strings.Contains(output.String(), "whisper_model_path") {
		t.Fatalf("doctor did not explain the unconfigured model: %q", output.String())
	}
}

func TestRunRejectsInvalidConfigurationBeforeDiagnostics(t *testing.T) {
	inputRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(inputRoot, "sample.mp4"), []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	application := newTestApplication(t, runner, &bytes.Buffer{})
	application.platform = "windows"
	cfg := config.Defaults()
	cfg.InputRoot = inputRoot
	cfg.MaxWidth = 0

	err := application.Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "max_width") {
		t.Fatalf("expected invalid max_width to be rejected, got %v", err)
	}
	if runner.diagnosticCalls != 0 {
		t.Fatalf("diagnostics ran with invalid configuration: %d calls", runner.diagnosticCalls)
	}
}

type fakeRunner struct {
	mu                sync.Mutex
	paths             map[string]string
	encoderList       string
	commands          []process.Command
	encodeCalls       int
	diagnosticCalls   int
	progressEvents    int
	audioInputs       []string
	whisperArgs       []string
	whisperTranscript string
	whisperError      error
	encodeError       error
}

type fakeRuntimeAssetManager struct {
	assets      map[string]runtimeassets.PreparedAsset
	calls       []string
	err         error
	onCallbacks func(string, func(runtimeassets.Asset) bool, func(runtimeassets.DownloadProgress))
}

func (manager *fakeRuntimeAssetManager) Prepare(
	ctx context.Context,
	id string,
	consent func(runtimeassets.Asset) bool,
	progress func(runtimeassets.DownloadProgress),
) (runtimeassets.PreparedAsset, error) {
	manager.calls = append(manager.calls, id)
	if manager.onCallbacks != nil {
		manager.onCallbacks(id, consent, progress)
	}
	if err := ctx.Err(); err != nil {
		return runtimeassets.PreparedAsset{}, err
	}
	if manager.err != nil {
		return runtimeassets.PreparedAsset{}, manager.err
	}
	asset, ok := manager.assets[id]
	if !ok {
		return runtimeassets.PreparedAsset{}, fmt.Errorf("unknown fake asset %q", id)
	}
	return asset, nil
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		paths:       map[string]string{"ffmpeg": "fake-ffmpeg", "ffprobe": "fake-ffprobe"},
		encoderList: " V..... h264_amf AMD H.264 Encoder\n",
	}
}

func (runner *fakeRunner) LookPath(name string) (string, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	path, ok := runner.paths[name]
	if !ok {
		return "", errors.New("not found")
	}
	return path, nil
}

func (runner *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, []byte, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.diagnosticCalls++
	if containsArg(args, "format=duration") {
		return []byte("10.0\n"), nil, nil
	}
	if containsArg(args, "-encoders") {
		return []byte(runner.encoderList), nil, nil
	}
	if len(args) == 0 {
		return nil, nil, errors.New("probe output path is missing")
	}
	if err := os.WriteFile(args[len(args)-1], []byte("probe output"), 0o600); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func (runner *fakeRunner) Execute(_ context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.commands = append(runner.commands, command)
	switch command.Executable {
	case "fake-ffmpeg":
		if containsPair(command.Args, "-ar", "16000") {
			for index, arg := range command.Args[:len(command.Args)-1] {
				if arg == "-i" && index+1 < len(command.Args) {
					runner.audioInputs = append(runner.audioInputs, command.Args[index+1])
					break
				}
			}
			if err := os.WriteFile(command.Args[len(command.Args)-1], []byte("temporary audio"), 0o600); err != nil {
				return process.Result{ExitCode: 1}, err
			}
			return process.Result{ExitCode: 0}, nil
		}
		runner.encodeCalls++
		if runner.encodeError != nil {
			return process.Result{ExitCode: 1}, runner.encodeError
		}
		if err := os.WriteFile(command.Args[len(command.Args)-1], []byte("encoded output"), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		if onOutput != nil {
			for frame := 1; frame <= runner.progressEvents; frame++ {
				data := []byte(fmt.Sprintf("frame=%d\nout_time_us=%d000000\nspeed=1.25x\nprogress=continue\n", frame, frame))
				onOutput(process.OutputChunk{Stream: process.Stdout, Data: data})
			}
			if runner.progressEvents > 0 {
				data := []byte(fmt.Sprintf("frame=%d\nout_time_us=%d000000\nspeed=1.25x\nprogress=end\n", runner.progressEvents, runner.progressEvents))
				onOutput(process.OutputChunk{Stream: process.Stdout, Data: data})
			}
		}
		return process.Result{ExitCode: 0}, nil
	case "fake-whisper":
		runner.whisperArgs = append([]string(nil), command.Args...)
		if runner.whisperError != nil {
			return process.Result{ExitCode: 1}, runner.whisperError
		}
		prefix, ok := argValue(command.Args, "-of")
		if !ok {
			return process.Result{ExitCode: 1}, errors.New("missing whisper output prefix")
		}
		if err := os.WriteFile(prefix+".srt", []byte(runner.whisperTranscript), 0o600); err != nil {
			return process.Result{ExitCode: 1}, err
		}
		return process.Result{ExitCode: 0}, nil
	case "fake-ffprobe":
		return process.Result{
			ExitCode: 0,
			Stdout:   []byte(`{"streams":[{"codec_name":"h264","width":640,"height":360}]}`),
		}, nil
	default:
		return process.Result{ExitCode: -1}, errors.New("unexpected executable")
	}
}

func (runner *fakeRunner) usedEncoder(name string) bool {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	for _, command := range runner.commands {
		if command.Executable != "fake-ffmpeg" {
			continue
		}
		for index := 0; index+1 < len(command.Args); index++ {
			if command.Args[index] == "-c:v" && command.Args[index+1] == name {
				return true
			}
		}
	}
	return false
}

func readJobManifestFiles(t *testing.T, root string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	manifests := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			manifests = append(manifests, entry)
		}
	}
	return manifests
}

func newTestApplication(t *testing.T, runner process.CommandRunner, output io.Writer) *Application {
	t.Helper()
	application := New(runner, output)
	root := t.TempDir()
	store, err := jobmanifest.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	application.jobManifests = store
	application.workspaceManager = workspace.Manager{Roots: []string{filepath.Join(root, "workspaces")}}
	return application
}

type gatedRunner struct {
	*fakeRunner
	started chan<- string
	release <-chan struct{}
}

type invalidVideoRunner struct {
	*fakeRunner
	invalidPaths map[string]bool
}

type cancelDiagnosticsRunner struct {
	*fakeRunner
	cancel context.CancelFunc
}

func (runner *cancelDiagnosticsRunner) Run(ctx context.Context, executable string, args ...string) ([]byte, []byte, error) {
	if containsArg(args, "-encoders") {
		runner.cancel()
		return nil, nil, ctx.Err()
	}
	return runner.fakeRunner.Run(ctx, executable, args...)
}

type blockingAppEncodeRunner struct {
	*fakeRunner
	started chan struct{}
}

func (runner *blockingAppEncodeRunner) Execute(ctx context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	if command.Executable == "fake-ffmpeg" && containsPair(command.Args, "-c:v", "h264_amf") {
		runner.started <- struct{}{}
		<-ctx.Done()
		return process.Result{ExitCode: -1}, ctx.Err()
	}
	return runner.fakeRunner.Execute(ctx, command, onOutput)
}

func (runner *invalidVideoRunner) Execute(ctx context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	if command.Executable == "fake-ffprobe" && len(command.Args) > 0 && runner.invalidPaths[command.Args[len(command.Args)-1]] {
		return process.Result{
			ExitCode: 0,
			Stdout:   []byte(`{"streams":[{"codec_name":"hevc","width":640,"height":360}]}`),
		}, nil
	}
	return runner.fakeRunner.Execute(ctx, command, onOutput)
}

func (runner *gatedRunner) Execute(ctx context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	stage := ""
	if command.Executable == "fake-ffmpeg" {
		if containsPair(command.Args, "-ar", "16000") {
			stage = "transcription"
		} else if containsPair(command.Args, "-c:v", "h264_amf") {
			stage = "compression"
		}
	}
	if stage != "" {
		runner.started <- stage
		<-runner.release
	}
	return runner.fakeRunner.Execute(ctx, command, onOutput)
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func containsPair(args []string, key, value string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key && args[index+1] == value {
			return true
		}
	}
	return false
}

func argValue(args []string, key string) (string, bool) {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == key {
			return args[index+1], true
		}
	}
	return "", false
}
