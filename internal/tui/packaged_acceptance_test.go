//go:build acceptance && windows

package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/media/ffmpeg"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
	"github.com/maucabreradev/video-compressor-extended/internal/scheduler"
)

func TestCleanPackagedWindowsRun(t *testing.T) {
	bundleRoot := requiredAcceptanceValue(t, "VCX_ACCEPTANCE_BUNDLE_ROOT")
	sample := requiredAcceptanceValue(t, "VCX_ACCEPTANCE_SAMPLE")
	cacheRoot := requiredAcceptanceValue(t, "VCX_ACCEPTANCE_CACHE_ROOT")
	if runtime.GOOS != "windows" {
		t.Fatalf("clean package acceptance requires Windows, got %s", runtime.GOOS)
	}
	if _, err := os.Stat(sample); err != nil {
		t.Fatalf("inspect acceptance media: %v", err)
	}
	for _, asset := range []string{"whisper-bin-x64.zip", "ggml-base.bin"} {
		if _, err := os.Stat(filepath.Join(bundleRoot, "runtime-assets", asset)); err != nil {
			t.Fatalf("guided package is missing bundled asset %q: %v", asset, err)
		}
	}
	if _, err := os.Stat(filepath.Join(bundleRoot, "runtime-assets", "ffmpeg-9.0.2-essentials_build.zip")); err == nil {
		t.Fatal("GPLv3 FFmpeg asset must not be bundled until its source obligations are cleared")
	}
	if _, err := os.Stat(filepath.Join(bundleRoot, "THIRD_PARTY_NOTICES.md")); err != nil {
		t.Fatalf("guided package is missing third-party notices: %v", err)
	}

	inputHash := acceptanceFileHash(t, sample)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	application := app.New(process.Runner{}, io.Discard)
	application.SetRuntimeAssetManager(runtimeassets.Manager{BundleRoot: filepath.Join(bundleRoot, "runtime-assets"), Root: cacheRoot})
	control := scheduler.NewDispatchControl()
	cfg := config.Defaults()
	model := NewWorkflowModel(cfg, application.Preview, control, cancel)
	model.workflow.ctx = ctx
	progressBytes := int64(0)
	consented := make(map[string]bool)
	model.SetWorkflowPreviewWithCallbacks(func(previewContext context.Context, previewConfig config.Config, callbacks app.RuntimeSetupCallbacks) (app.RunPreview, error) {
		callbacks.Consent = func(asset runtimeassets.Asset) bool {
			reply := make(chan bool, 1)
			updated, _ := model.Update(workflowAssetConsentMsg{asset: asset, reply: reply})
			model = updated.(*Model)
			view := model.View()
			for _, want := range []string{asset.ID, asset.Version, asset.SourceURL, asset.License, "y allow", "n decline"} {
				if !strings.Contains(view, want) {
					t.Errorf("consent prompt is missing %q: %q", want, view)
				}
			}
			consented[asset.ID] = true
			updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			model = updated.(*Model)
			return <-reply
		}
		callbacks.Progress = func(update runtimeassets.DownloadProgress) {
			if update.ReceivedBytes > progressBytes {
				progressBytes = update.ReceivedBytes
			}
			updated, _ := model.Update(workflowAssetProgressMsg{progress: update})
			model = updated.(*Model)
		}
		return application.PreviewWithCallbacks(previewContext, previewConfig, callbacks)
	})

	// Exercise the normal selection path using a Unicode media path without
	// configuring FFmpeg, Whisper, the model, or an output directory.
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = updated.(*Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(sample)})
	model = updated.(*Model)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*Model)
	if command == nil {
		t.Fatal("selecting media did not start readiness preview")
	}
	updated, _ = model.Update(command())
	model = updated.(*Model)
	if model.workflow.phase != workflowReview || model.workflow.preview.SubtitlesEnabled == false ||
		!model.workflow.preview.CompressionEnabled || len(model.workflow.preview.Plans) != 1 {
		t.Fatalf("clean-package preflight did not produce the default run review: %+v", model.workflow.preview)
	}
	if model.workflow.cfg.InputRoot != sample || model.workflow.cfg.OutputDir != config.Defaults().OutputDir || model.workflow.cfg.WhisperModelPath != "" {
		t.Fatalf("normal path required manual input/output/model configuration: %+v", model.workflow.cfg)
	}
	if !consented["ffmpeg"] || progressBytes == 0 {
		t.Fatalf("unbundled FFmpeg did not use consent-gated acquisition and progress: consented=%v bytes=%d", consented, progressBytes)
	}
	manifest, err := runtimeassets.Default()
	if err != nil {
		t.Fatal(err)
	}
	ffmpegAsset, ok := acceptanceManifestAsset(manifest, "ffmpeg")
	if !ok {
		t.Fatal("pinned manifest does not contain the FFmpeg acceptance asset")
	}
	ffmpegFile, ok := ffmpegAsset.Files["ffmpeg"]
	if !ok {
		t.Fatal("pinned FFmpeg asset does not map the ffmpeg executable")
	}
	ffmpegPath := filepath.Join(cacheRoot, "ffmpeg", ffmpegAsset.Revision, "content", filepath.FromSlash(ffmpegFile.Path))
	versionOutput, versionErrorOutput, err := (process.Runner{}).Run(ctx, ffmpegPath, "-hide_banner", "-version")
	if err != nil {
		t.Fatalf("read managed FFmpeg version: %v: %s", err, strings.TrimSpace(string(versionErrorOutput)))
	}
	ffmpegBanner := strings.SplitN(strings.TrimSpace(string(versionOutput)), "\n", 2)[0]
	if !strings.Contains(ffmpegBanner, "ffmpeg version 9.0.2") {
		t.Fatalf("managed FFmpeg banner = %q, want pinned 9.0.2 build", ffmpegBanner)
	}
	t.Logf("FFmpeg build: %s", ffmpegBanner)
	for _, check := range model.workflow.preview.Diagnostics.Checks {
		t.Logf("readiness %s: %s — %s", check.Name, check.Status, check.Message)
	}
	if !model.workflow.preview.SoftwareFallbackAvailable && model.workflow.preview.EncoderBackend != ffmpeg.BackendAMF {
		t.Fatalf("preflight selected no verified encoding path: backend=%q encoder=%q", model.workflow.preview.EncoderBackend, model.workflow.preview.EncoderName)
	}
	var runErr error
	model.startRun = func(runConfig config.Config) {
		if runConfig.InputRoot != sample {
			t.Errorf("run started with input %q, want selected media %q", runConfig.InputRoot, sample)
		}
		if model.workflow.preview.SoftwareFallbackAvailable && !runConfig.AllowSoftwareEncoding {
			t.Error("run started without the explicit software encoding confirmation")
		}
		runErr = application.RunWithControl(ctx, runConfig, control, nil)
	}
	if strings.EqualFold(os.Getenv("VCX_ACCEPTANCE_EXPECT_SOFTWARE"), "true") &&
		(model.workflow.preview.EncoderBackend != ffmpeg.BackendSoftware || !model.workflow.preview.SoftwareFallbackAvailable) {
		t.Fatalf("software-only acceptance host selected backend=%q; expected a probed libx264 fallback", model.workflow.preview.EncoderBackend)
	}
	if strings.EqualFold(os.Getenv("VCX_ACCEPTANCE_EXPECT_AMF"), "true") && model.workflow.preview.EncoderBackend != ffmpeg.BackendAMF {
		t.Fatalf("AMF hardware acceptance selected backend=%q encoder=%q; expected a successful real AMF probe", model.workflow.preview.EncoderBackend, model.workflow.preview.EncoderName)
	}
	if model.workflow.preview.SoftwareFallbackAvailable {
		if !model.workflow.preview.SoftwareFallbackConfirmationRequired || !strings.Contains(strings.ToLower(model.workflow.preview.SoftwareFallbackWarning), "cpu") {
			t.Fatalf("software fallback lacks its required warning/confirmation: %+v", model.workflow.preview)
		}
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(*Model)
		if !strings.Contains(model.View(), "CONFIRM SOFTWARE H.264") {
			t.Fatalf("TUI did not show the software fallback gate: %q", model.View())
		}
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		model = updated.(*Model)
	} else {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(*Model)
	}

	// The model's confirmation invokes the run handler. AMF requires one review
	// confirmation; software fallback requires the separate `y` above.
	if runErr != nil {
		t.Fatalf("process packaged media without manual tool/model paths: %v", runErr)
	}
	if model.workflow.phase != workflowRunning {
		t.Fatalf("TUI confirmation did not start the shared application engine: phase=%v", model.workflow.phase)
	}
	for _, plan := range model.workflow.preview.Plans {
		for _, output := range []string{plan.Video, plan.Subtitle} {
			info, err := os.Stat(output)
			if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
				t.Fatalf("validated packaged output missing or empty %q: info=%v err=%v", output, info, err)
			}
		}
	}
	if got := acceptanceFileHash(t, sample); got != inputHash {
		t.Fatalf("original input changed: before=%s after=%s", inputHash, got)
	}
	t.Logf("Windows=%s GPU=%s driver=%s backend=%s encoder=%s FFmpeg=manifest-pinned 9.0.2",
		os.Getenv("VCX_ACCEPTANCE_WINDOWS"), os.Getenv("VCX_ACCEPTANCE_GPU"),
		os.Getenv("VCX_ACCEPTANCE_DRIVER"), model.workflow.preview.EncoderBackend,
		model.workflow.preview.EncoderName)
}

func requiredAcceptanceValue(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if strings.TrimSpace(value) == "" {
		t.Fatalf("set %s to run clean package acceptance", name)
	}
	return value
}

func acceptanceFileHash(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open acceptance input %q: %v", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatalf("hash acceptance input %q: %v", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func acceptanceManifestAsset(manifest runtimeassets.Manifest, id string) (runtimeassets.Asset, bool) {
	for _, asset := range manifest.Assets {
		if asset.ID == id {
			return asset, true
		}
	}
	return runtimeassets.Asset{}, false
}
