package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/config"
	"github.com/maucabreradev/video-compressor-extended/internal/jobmanifest"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
	"github.com/maucabreradev/video-compressor-extended/internal/workspace"
)

func TestResumeJobPublishesValidatedStagedVideoWithoutReencoding(t *testing.T) {
	application, runner, cfg, store, manifest, plan, workspacePath := createCompressionRecoveryFixture(t, []byte("completed staged video"))

	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err != nil {
		t.Fatalf("resume staged video: %v", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("validated staged video was re-encoded %d times", runner.encodeCalls)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "completed staged video" {
		t.Fatalf("staged video was not published: contents=%q err=%v", contents, err)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("recovered workspace was not cleaned: %v", err)
	}
	updated, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Items[0].Compression.State != jobmanifest.StagePublished || updated.Items[0].Compression.WorkspaceID != "" {
		t.Fatalf("recovered compression stage state = %+v", updated.Items[0].Compression)
	}
	if updated.Items[0].Subtitles.State != jobmanifest.StagePending {
		t.Fatalf("independent subtitle stage changed during compression recovery: %s", updated.Items[0].Subtitles.State)
	}
}

func TestResumeJobRerunsAndCleansIncompleteRunningVideo(t *testing.T) {
	application, runner, cfg, store, manifest, plan, workspacePath := createCompressionRecoveryFixture(t, nil)

	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err != nil {
		t.Fatalf("resume incomplete compression: %v", err)
	}
	if runner.encodeCalls != 1 {
		t.Fatalf("incomplete staged video encode calls = %d, want one rerun", runner.encodeCalls)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "encoded output" {
		t.Fatalf("rerun video was not published: contents=%q err=%v", contents, err)
	}
	workspaces, err := os.ReadDir(filepath.Dir(workspacePath))
	if err != nil || len(workspaces) != 0 {
		t.Fatalf("incomplete and rerun workspaces remain: entries=%v err=%v", workspaces, err)
	}
	updated, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Items[0].Compression.State != jobmanifest.StagePublished {
		t.Fatalf("rerun compression state = %q, want published", updated.Items[0].Compression.State)
	}
}

func TestResumeJobRejectsStageThatFailsFFprobeValidation(t *testing.T) {
	application, baseRunner, cfg, store, manifest, plan, _ := createCompressionRecoveryFixture(t, []byte("not a valid h264 video"))
	manifestBefore, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	// The workspace ID in the saved record determines the staged path.
	stagedPath := filepath.Join(application.workspaceManager.Roots[0], manifestBefore.Items[0].Compression.WorkspaceID, "video.mp4")
	runner := &invalidStageVideoRunner{fakeRunner: baseRunner, invalidPath: stagedPath}
	application.runner = runner

	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err != nil {
		t.Fatalf("resume after invalid staged artifact: %v", err)
	}
	if baseRunner.encodeCalls != 1 {
		t.Fatalf("invalid FFprobe stage encode calls = %d, want one rerun", baseRunner.encodeCalls)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "encoded output" {
		t.Fatalf("validated rerun was not published: contents=%q err=%v", contents, err)
	}
}

func TestResumeJobRejectsChangedSourceAndProcessingIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, cfg *config.Config, runner *fakeRunner)
	}{
		{
			name: "source fingerprint",
			mutate: func(t *testing.T, cfg *config.Config, _ *fakeRunner) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(cfg.InputRoot, "clip.mkv"), []byte("changed source"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:   "processing identity",
			mutate: func(_ *testing.T, cfg *config.Config, _ *fakeRunner) { cfg.FrameRate = 30 },
		},
		{
			name: "runtime identity",
			mutate: func(_ *testing.T, _ *config.Config, runner *fakeRunner) {
				runner.paths["ffmpeg"] = "different-ffmpeg"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			application, runner, cfg, _, manifest, _, _ := createCompressionRecoveryFixture(t, []byte("completed staged video"))
			test.mutate(t, &cfg, runner)
			if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err == nil {
				t.Fatal("incompatible job was resumed")
			}
			if runner.encodeCalls != 0 {
				t.Fatalf("incompatible job started %d encodes", runner.encodeCalls)
			}
		})
	}
}

func TestResumeJobHonorsCrossProcessJobLock(t *testing.T) {
	application, _, cfg, store, manifest, _, _ := createCompressionRecoveryFixture(t, []byte("completed staged video"))
	lock, err := store.TryLock(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); !errors.Is(err, jobmanifest.ErrJobLocked) {
		t.Fatalf("resume while locked error = %v, want ErrJobLocked", err)
	}
}

func TestResumeJobSkipsValidatedFinalVideoIndependently(t *testing.T) {
	application, runner, cfg, store, manifest, plan, workspacePath := createCompressionRecoveryFixture(t, []byte("staged video"))
	if err := os.MkdirAll(filepath.Dir(plan.Video), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.Video, []byte("already published video"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err != nil {
		t.Fatalf("resume with valid final video: %v", err)
	}
	if runner.encodeCalls != 0 {
		t.Fatalf("valid final video was re-encoded %d times", runner.encodeCalls)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "already published video" {
		t.Fatalf("published video changed: contents=%q err=%v", contents, err)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("stale mapped workspace was not cleaned: %v", err)
	}
	updated, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Items[0].Compression.State != jobmanifest.StagePublished {
		t.Fatalf("valid final stage state = %q, want published", updated.Items[0].Compression.State)
	}
}

func TestResumeJobRetainsValidatedStageWhenFinalPublicationIsInterrupted(t *testing.T) {
	application, baseRunner, cfg, store, manifest, plan, workspacePath := createCompressionRecoveryFixture(t, []byte("completed staged video"))
	runner := &destinationConflictRunner{fakeRunner: baseRunner, destination: plan.Video}
	application.runner = runner

	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err == nil {
		t.Fatal("expected the destination conflict to interrupt publication")
	}
	updated, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Items[0].Compression.State != jobmanifest.StageStaged || updated.Items[0].Compression.WorkspaceID == "" {
		t.Fatalf("validated stage was not retained after publication failure: %+v", updated.Items[0].Compression)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "concurrent winner" {
		t.Fatalf("publication conflict replaced the destination: contents=%q err=%v", contents, err)
	}
	if err := os.Remove(plan.Video); err != nil {
		t.Fatal(err)
	}
	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err != nil {
		t.Fatalf("resume retained stage after conflict removal: %v", err)
	}
	if baseRunner.encodeCalls != 0 {
		t.Fatalf("retained stage was re-encoded; encode calls = %d", baseRunner.encodeCalls)
	}
	if contents, err := os.ReadFile(plan.Video); err != nil || string(contents) != "completed staged video" {
		t.Fatalf("retained stage was not published: contents=%q err=%v", contents, err)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("retained workspace was not cleaned after recovery: %v", err)
	}
}

func TestResumeJobPublishesValidatedStagedSubtitleWithoutRetranscribing(t *testing.T) {
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	outputRoot := filepath.Join(root, "output")
	workspaceRoot := filepath.Join(root, "workspaces")
	source := filepath.Join(inputRoot, "clip.mkv")
	modelPath := filepath.Join(root, "model.bin")
	for path, contents := range map[string]string{source: "source video", modelPath: "test model"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plans, err := outputplan.PlanOutputs(inputRoot, outputRoot, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	workspaceID := "w-001122334455"
	workspacePath := filepath.Join(workspaceRoot, workspaceID)
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript := "1\n00:00:00,000 --> 00:00:01,000\nRecovered subtitle.\n"
	if err := os.WriteFile(filepath.Join(workspacePath, "transcript.srt"), []byte(transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	item, err := jobmanifest.NewItem(source, plans[0].Video, plans[0].Subtitle)
	if err != nil {
		t.Fatal(err)
	}
	item.Subtitles = jobmanifest.StageRecord{State: jobmanifest.StageStaged, WorkspaceID: workspaceID, ArtifactName: "transcript.srt"}
	modelSHA256, err := fileSHA256(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.InputRoot, cfg.OutputDir, cfg.WhisperModelPath = inputRoot, outputRoot, modelPath
	cfg.CompressionEnabled = false
	resources := cfg.EffectiveResources()
	manifest, err := jobmanifest.NewManifest(jobmanifest.ProcessingIdentity{
		SubtitlesEnabled: true, FFmpegRevision: "fake-ffmpeg", Language: cfg.Language, WhisperThreads: resources.WhisperThreads,
		WhisperGPU: resources.WhisperGPU, WhisperRevision: "fake-whisper", ModelSHA256: modelSHA256,
	}, []jobmanifest.Item{item})
	if err != nil {
		t.Fatal(err)
	}
	store, err := jobmanifest.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	runner.paths["whisper-cli"] = "fake-whisper"
	application := New(runner, nil)
	application.platform = "windows"
	application.workspaceManager = workspace.Manager{Roots: []string{workspaceRoot}}
	application.jobManifests = store

	if err := application.ResumeJob(context.Background(), cfg, manifest.JobID); err != nil {
		t.Fatalf("resume staged subtitle: %v", err)
	}
	if len(runner.audioInputs) != 0 || len(runner.whisperArgs) != 0 {
		t.Fatalf("staged subtitle was regenerated: audio inputs=%v whisper args=%v", runner.audioInputs, runner.whisperArgs)
	}
	if contents, err := os.ReadFile(plans[0].Subtitle); err != nil || string(contents) != transcript {
		t.Fatalf("staged subtitle was not published: contents=%q err=%v", contents, err)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("recovered subtitle workspace was not cleaned: %v", err)
	}
}

func createCompressionRecoveryFixture(t *testing.T, stagedContents []byte) (*Application, *fakeRunner, config.Config, *jobmanifest.Store, jobmanifest.Manifest, outputplan.OutputPlan, string) {
	t.Helper()
	root := t.TempDir()
	inputRoot := filepath.Join(root, "input")
	outputRoot := filepath.Join(root, "output")
	workspaceRoot := filepath.Join(root, "workspaces")
	source := filepath.Join(inputRoot, "clip.mkv")
	if err := os.MkdirAll(inputRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("source video"), 0o600); err != nil {
		t.Fatal(err)
	}
	plans, err := outputplan.PlanOutputs(inputRoot, outputRoot, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	workspaceID := "w-001122334455"
	workspacePath := filepath.Join(workspaceRoot, workspaceID)
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspacePath, "video.mp4"), stagedContents, 0o600); err != nil {
		t.Fatal(err)
	}
	item, err := jobmanifest.NewItem(source, plans[0].Video, plans[0].Subtitle)
	if err != nil {
		t.Fatal(err)
	}
	item.Compression = jobmanifest.StageRecord{State: jobmanifest.StageRunning, WorkspaceID: workspaceID, ArtifactName: "video.mp4"}
	manifest, err := jobmanifest.NewManifest(jobmanifest.ProcessingIdentity{
		CompressionEnabled: true, VideoBackend: "amf", VideoCodec: "h264",
		MaxWidth: 1280, MaxHeight: 720, FFmpegRevision: "fake-ffmpeg",
	}, []jobmanifest.Item{item})
	if err != nil {
		t.Fatal(err)
	}
	store, err := jobmanifest.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	delete(runner.paths, "whisper-cli")
	application := New(runner, nil)
	application.platform = "windows"
	application.workspaceManager = workspace.Manager{Roots: []string{workspaceRoot}}
	application.jobManifests = store
	cfg := config.Defaults()
	cfg.InputRoot, cfg.OutputDir = inputRoot, outputRoot
	return application, runner, cfg, store, manifest, plans[0], workspacePath
}

type destinationConflictRunner struct {
	*fakeRunner
	destination          string
	videoValidationCount int
}

type invalidStageVideoRunner struct {
	*fakeRunner
	invalidPath string
}

func (runner *invalidStageVideoRunner) Execute(ctx context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	if command.Executable == "fake-ffprobe" && command.Args[len(command.Args)-1] == runner.invalidPath {
		return process.Result{ExitCode: 0, Stdout: []byte(`{"streams":[{"codec_name":"hevc","width":640,"height":360}]}`)}, nil
	}
	return runner.fakeRunner.Execute(ctx, command, onOutput)
}

func (runner *destinationConflictRunner) Execute(ctx context.Context, command process.Command, onOutput func(process.OutputChunk)) (process.Result, error) {
	if command.Executable == "fake-ffprobe" && containsPair(command.Args, "-show_entries", "stream=codec_name,width,height") {
		runner.videoValidationCount++
		if runner.videoValidationCount == 1 {
			if err := os.MkdirAll(filepath.Dir(runner.destination), 0o700); err != nil {
				return process.Result{ExitCode: 1}, err
			}
			if err := os.WriteFile(runner.destination, []byte("concurrent winner"), 0o600); err != nil {
				return process.Result{ExitCode: 1}, err
			}
		}
	}
	return runner.fakeRunner.Execute(ctx, command, onOutput)
}
