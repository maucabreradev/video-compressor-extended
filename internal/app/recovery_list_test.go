package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/jobmanifest"
	"github.com/maucabreradev/video-compressor-extended/internal/media/outputplan"
)

func TestListRecoveryJobsSummarizesSavedStageState(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "media", "clip.mkv")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	item, err := jobmanifest.NewItem(source, filepath.Join(root, "out", "clip.mp4"), filepath.Join(root, "out", "clip.srt"))
	if err != nil {
		t.Fatal(err)
	}
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
	application := New(nil, nil)
	application.platform = "windows"
	application.jobManifests = store

	jobs, err := application.ListRecoveryJobs(context.Background())
	if err != nil {
		t.Fatalf("list saved jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("listed %d jobs, want one: %+v", len(jobs), jobs)
	}
	job := jobs[0]
	if job.ID != manifest.JobID || job.SourceSummary != "clip.mkv" || job.CompressionState != "pending 1" || job.Status != RecoveryJobReady {
		t.Fatalf("job summary omitted recovery details: %+v", job)
	}
}

func TestListRecoveryJobsMarksActiveStaleCorruptAndMissingStages(t *testing.T) {
	root := t.TempDir()
	store, err := jobmanifest.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	activeManifest := createRecoveryListManifest(t, root, "active", jobmanifest.StagePending, "")
	staleManifest := createRecoveryListManifest(t, root, "stale", jobmanifest.StagePending, "")
	missingStageManifest := createRecoveryListManifest(t, root, "missing-stage", jobmanifest.StageStaged, "w-111122223333")
	for _, manifest := range []jobmanifest.Manifest{activeManifest, staleManifest, missingStageManifest} {
		if err := store.Create(manifest); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "media", "stale.mkv"), []byte("changed source"), 0o600); err != nil {
		t.Fatal(err)
	}
	corruptID := "00000000000000000000000000000002"
	if err := os.WriteFile(filepath.Join(root, "jobs", corruptID+".json"), []byte(`{"schema_version":`), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := store.TryLock(activeManifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	application := New(nil, nil)
	application.platform = "windows"
	application.jobManifests = store
	application.workspaceManager.Roots = []string{filepath.Join(root, "workspaces")}

	jobs, err := application.ListRecoveryJobs(context.Background())
	if err != nil {
		t.Fatalf("list recovery jobs: %v", err)
	}
	byID := make(map[string]RecoveryJob, len(jobs))
	for _, job := range jobs {
		byID[job.ID] = job
	}
	if byID[activeManifest.JobID].Status != RecoveryJobActive {
		t.Errorf("active job status = %q, want active", byID[activeManifest.JobID].Status)
	}
	if byID[staleManifest.JobID].Status != RecoveryJobStale || !strings.Contains(byID[staleManifest.JobID].Detail, "changed") {
		t.Errorf("changed source was not identified: %+v", byID[staleManifest.JobID])
	}
	if byID[corruptID].Status != RecoveryJobInvalid || byID[corruptID].Detail == "" {
		t.Errorf("corrupt manifest was not explained: %+v", byID[corruptID])
	}
	if byID[missingStageManifest.JobID].Status != RecoveryJobReady || !strings.Contains(byID[missingStageManifest.JobID].Detail, "temporary artifact missing") {
		t.Errorf("missing stage artifact was not explained as recoverable: %+v", byID[missingStageManifest.JobID])
	}
}

func TestConfigFromManifestReconstructsMultipleNestedSourcesAndOutputs(t *testing.T) {
	root := t.TempDir()
	inputRoot, outputRoot := filepath.Join(root, "media"), filepath.Join(root, "processed")
	sources := []string{filepath.Join(inputRoot, "a", "clip.mkv"), filepath.Join(inputRoot, "b", "clip.mkv")}
	for _, source := range sources {
		if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plans, err := outputplan.PlanOutputs(inputRoot, outputRoot, sources)
	if err != nil {
		t.Fatal(err)
	}
	items := make([]jobmanifest.Item, 0, len(plans))
	for _, plan := range plans {
		item, err := jobmanifest.NewItem(plan.Source, plan.Video, plan.Subtitle)
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}
	manifest, err := jobmanifest.NewManifest(jobmanifest.ProcessingIdentity{
		CompressionEnabled: true, VideoBackend: "amf", VideoCodec: "h264",
		MaxWidth: 1280, MaxHeight: 720, FFmpegRevision: "fake-ffmpeg",
	}, items)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := configFromManifest(manifest)
	if err != nil {
		t.Fatalf("reconstruct config: %v", err)
	}
	rebuilt, err := outputplan.PlanOutputs(cfg.InputRoot, cfg.OutputDir, cfg.InputPaths)
	if err != nil {
		t.Fatal(err)
	}
	if len(rebuilt) != len(plans) {
		t.Fatalf("rebuilt %d plans, want %d", len(rebuilt), len(plans))
	}
	for index := range plans {
		if rebuilt[index] != plans[index] {
			t.Errorf("rebuilt plan %d = %+v, want %+v", index, rebuilt[index], plans[index])
		}
	}
}

func createRecoveryListManifest(t *testing.T, root, name string, state jobmanifest.StageState, workspaceID string) jobmanifest.Manifest {
	t.Helper()
	source := filepath.Join(root, "media", name+".mkv")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original source"), 0o600); err != nil {
		t.Fatal(err)
	}
	item, err := jobmanifest.NewItem(source, filepath.Join(root, "out", name+".mp4"), filepath.Join(root, "out", name+".srt"))
	if err != nil {
		t.Fatal(err)
	}
	if state != jobmanifest.StagePending {
		item.Compression = jobmanifest.StageRecord{State: state, WorkspaceID: workspaceID, ArtifactName: "video.mp4"}
	}
	manifest, err := jobmanifest.NewManifest(jobmanifest.ProcessingIdentity{
		CompressionEnabled: true, VideoBackend: "amf", VideoCodec: "h264",
		MaxWidth: 1280, MaxHeight: 720, FFmpegRevision: "fake-ffmpeg",
	}, []jobmanifest.Item{item})
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}
