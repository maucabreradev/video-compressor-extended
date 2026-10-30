package jobmanifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestNewManifestRoundTripsDistinctFilesWithSameBasename(t *testing.T) {
	items := sameBasenameItems(t)
	manifest, err := NewManifest(testIdentity(), items)
	if err != nil {
		t.Fatalf("create job manifest: %v", err)
	}
	if manifest.SchemaVersion != CurrentSchemaVersion || !validJobID(manifest.JobID) {
		t.Fatalf("unexpected manifest identity: version=%d id=%q", manifest.SchemaVersion, manifest.JobID)
	}
	if manifest.CreatedAt.IsZero() || manifest.UpdatedAt.Before(manifest.CreatedAt) {
		t.Fatalf("invalid manifest timestamps: created=%v updated=%v", manifest.CreatedAt, manifest.UpdatedAt)
	}
	if len(manifest.Items) != 2 || manifest.Items[0].SourcePath == manifest.Items[1].SourcePath {
		t.Fatalf("distinct source paths with matching basenames were not retained: %+v", manifest.Items)
	}
	for _, item := range manifest.Items {
		if item.Compression.State != StagePending || item.Subtitles.State != StagePending {
			t.Fatalf("new stages should be pending: %+v", item)
		}
	}

	root := filepath.Join(t.TempDir(), "user-cache", "vcx", "jobs")
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("open manifest store: %v", err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatalf("persist job manifest: %v", err)
	}

	// A second Store models a later VCX process opening the same local job data.
	restarted, err := NewStore(root)
	if err != nil {
		t.Fatalf("reopen manifest store: %v", err)
	}
	got, err := restarted.Load(manifest.JobID)
	if err != nil {
		t.Fatalf("load job manifest after restart: %v", err)
	}
	if !reflectManifestEqual(got, manifest) {
		t.Fatalf("round-tripped manifest differs:\ngot:  %+v\nwant: %+v", got, manifest)
	}
	data, err := os.ReadFile(filepath.Join(root, manifest.JobID+manifestSuffix))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "source media payload") {
		t.Fatal("manifest unexpectedly contains source media contents")
	}
}

func TestStoreSerializesConcurrentStageUpdates(t *testing.T) {
	manifest := newTestManifest(t)
	store, err := NewStore(filepath.Join(t.TempDir(), "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}

	var group sync.WaitGroup
	errs := make(chan error, len(manifest.Items))
	for index := range manifest.Items {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			err := store.Update(manifest.JobID, func(current *Manifest) error {
				current.Items[index].Subtitles = StageRecord{
					State: StageStaged, WorkspaceID: fmt.Sprintf("w-%012x", index+1),
					ArtifactName: "transcript.srt",
				}
				return nil
			})
			if err != nil {
				errs <- err
			}
		}(index)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent manifest update: %v", err)
	}

	got, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	for index, item := range got.Items {
		if item.Subtitles.State != StageStaged || item.Subtitles.ArtifactName != "transcript.srt" {
			t.Errorf("stage update %d was lost: %+v", index, item.Subtitles)
		}
	}
	if !got.UpdatedAt.After(manifest.UpdatedAt) {
		t.Fatalf("updated timestamp did not advance: before=%v after=%v", manifest.UpdatedAt, got.UpdatedAt)
	}
}

func TestUpdateFailureLeavesPreviousManifestIntact(t *testing.T) {
	manifest := newTestManifest(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("mutation rejected")
	err = store.Update(manifest.JobID, func(current *Manifest) error {
		current.Items[0].Compression.State = StageRunning
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("update error = %v, want %v", err, wantErr)
	}
	got, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflectManifestEqual(got, manifest) {
		t.Fatalf("failed update changed persisted manifest:\ngot:  %+v\nwant: %+v", got, manifest)
	}
}

func TestStoreNeverOverwritesAnExistingJobID(t *testing.T) {
	manifest := newTestManifest(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	conflict := manifest
	conflict.Identity.VideoCodec = "hevc"
	if err := store.Create(conflict); !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate job creation error = %v, want os.ErrExist", err)
	}
	got, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Identity.VideoCodec != manifest.Identity.VideoCodec {
		t.Fatalf("existing job was overwritten: codec=%q", got.Identity.VideoCodec)
	}
}

func TestLoadRejectsCorruptFutureAndPathTraversalManifests(t *testing.T) {
	tests := []struct {
		name string
		raw  func(Manifest) []byte
	}{
		{name: "truncated", raw: func(Manifest) []byte { return []byte(`{"schema_version":`) }},
		{name: "future schema", raw: func(manifest Manifest) []byte {
			manifest.SchemaVersion++
			return mustMarshal(t, manifest)
		}},
		{name: "workspace traversal", raw: func(manifest Manifest) []byte {
			manifest.Items[0].Compression = StageRecord{
				State: StageStaged, WorkspaceID: "../outside", ArtifactName: "video.mp4",
			}
			return mustMarshal(t, manifest)
		}},
		{name: "artifact traversal", raw: func(manifest Manifest) []byte {
			manifest.Items[0].Subtitles = StageRecord{
				State: StageStaged, WorkspaceID: "w-000000000001", ArtifactName: `..\outside.srt`,
			}
			return mustMarshal(t, manifest)
		}},
		{name: "relative source", raw: func(manifest Manifest) []byte {
			manifest.Items[0].SourcePath = `relative\clip.mp4`
			return mustMarshal(t, manifest)
		}},
		{name: "non-normalized output", raw: func(manifest Manifest) []byte {
			destination := manifest.Items[0].VideoDestination
			manifest.Items[0].VideoDestination = filepath.Dir(destination) + string(filepath.Separator) +
				"child" + string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(destination)
			return mustMarshal(t, manifest)
		}},
		{name: "unknown field", raw: func(manifest Manifest) []byte {
			raw := mustMarshal(t, manifest)
			return append(raw[:len(raw)-1], []byte(`,"unexpected":true}`)...)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := newTestManifest(t)
			store, err := NewStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Create(manifest); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(store.root, manifest.JobID+manifestSuffix)
			if err := os.WriteFile(path, test.raw(manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(manifest.JobID); err == nil {
				t.Fatal("expected unsafe or unsupported manifest to be rejected")
			}
		})
	}
}

func TestCaptureSourceFingerprintTracksMetadataWithoutHashingContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := CaptureSourceFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.SizeBytes != 3 || first.ModifiedAt.IsZero() {
		t.Fatalf("unexpected initial fingerprint: %+v", first)
	}
	if err := os.WriteFile(path, []byte("changed source"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := CaptureSourceFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("source fingerprint did not change after size changed: %+v", second)
	}
}

func TestReuseIdentityRejectsChangedSourceAndProcessingRuntime(t *testing.T) {
	manifest := newTestManifest(t)
	item := manifest.Items[0]
	if matches, err := item.MatchesSource(item.SourcePath); err != nil || !matches {
		t.Fatalf("unchanged source should match: matches=%v err=%v", matches, err)
	}
	if err := os.WriteFile(item.SourcePath, []byte("replacement media with a different size"), 0o600); err != nil {
		t.Fatal(err)
	}
	if matches, err := item.MatchesSource(item.SourcePath); err != nil || matches {
		t.Fatalf("changed source should not match: matches=%v err=%v", matches, err)
	}
	otherPath := filepath.Join(t.TempDir(), filepath.Base(item.SourcePath))
	if err := os.WriteFile(otherPath, []byte("source media payload: chapter-one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if matches, err := item.MatchesSource(otherPath); err != nil || matches {
		t.Fatalf("different source path should not match: matches=%v err=%v", matches, err)
	}

	identity := manifest.Identity
	if !identity.Matches(manifest.Identity) {
		t.Fatal("identical processing identity did not match")
	}
	changedRuntime := identity
	changedRuntime.VideoBackend = "software"
	if identity.Matches(changedRuntime) {
		t.Fatal("different encoder backend must not match the recorded identity")
	}
	changedModel := identity
	changedModel.ModelSHA256 = strings.Repeat("b", 64)
	if identity.Matches(changedModel) {
		t.Fatal("different Whisper model must not match the recorded identity")
	}
}

func TestStoreRejectsInvalidJobIdentifiersBeforeResolvingPaths(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "../outside", strings.Repeat("a", 31), strings.Repeat("g", 32)} {
		if _, err := store.Load(id); err == nil {
			t.Errorf("Load(%q) unexpectedly succeeded", id)
		}
	}
}

func TestZeroValueStoreDoesNotResolveFilesFromWorkingDirectory(t *testing.T) {
	var store Store
	_, err := store.Load(strings.Repeat("a", 32))
	if err == nil || !strings.Contains(err.Error(), "initialized") {
		t.Fatalf("zero-value store error = %v, want an initialization error", err)
	}
}

func newTestManifest(t *testing.T) Manifest {
	t.Helper()
	manifest, err := NewManifest(testIdentity(), sameBasenameItems(t))
	if err != nil {
		t.Fatalf("create test manifest: %v", err)
	}
	return manifest
}

func sameBasenameItems(t *testing.T) []Item {
	t.Helper()
	inputRoot := t.TempDir()
	outputRoot := t.TempDir()
	items := make([]Item, 0, 2)
	for _, folder := range []string{"chapter-one", "chapter-two"} {
		source := filepath.Join(inputRoot, folder, "lesson.mp4")
		if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte("source media payload: "+folder), 0o600); err != nil {
			t.Fatal(err)
		}
		item, err := NewItem(
			source,
			filepath.Join(outputRoot, folder, "lesson.mp4"),
			filepath.Join(outputRoot, folder, "lesson.srt"),
		)
		if err != nil {
			t.Fatalf("create manifest item: %v", err)
		}
		items = append(items, item)
	}
	return items
}

func testIdentity() ProcessingIdentity {
	return ProcessingIdentity{
		CompressionEnabled: true,
		SubtitlesEnabled:   true,
		VideoBackend:       "amf",
		VideoCodec:         "h264",
		MaxWidth:           1280,
		MaxHeight:          720,
		Language:           "en",
		WhisperThreads:     1,
		FFmpegRevision:     "ffmpeg-test-build",
		WhisperRevision:    "whisper-test-build",
		ModelSHA256:        strings.Repeat("a", 64),
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func reflectManifestEqual(first, second Manifest) bool {
	firstBytes, firstErr := json.Marshal(first)
	secondBytes, secondErr := json.Marshal(second)
	return firstErr == nil && secondErr == nil && string(firstBytes) == string(secondBytes)
}

func TestStoreDefaultRootUsesPerUserCache(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("LOCALAPPDATA is Windows-specific")
	}
	cacheRoot := filepath.Join(t.TempDir(), "Profile café", "AppData", "Local")
	t.Setenv("LOCALAPPDATA", cacheRoot)
	store, err := NewStore("")
	if err != nil {
		t.Fatalf("create default store: %v", err)
	}
	want := filepath.Join(cacheRoot, "vcx", "jobs")
	if store.root != want {
		t.Fatalf("default manifest root = %q, want %q", store.root, want)
	}
}

func TestNewStoreRejectsVolumeRootAndSharedPOSIXPermissions(t *testing.T) {
	volumeRoot := filepath.VolumeName(t.TempDir()) + string(filepath.Separator)
	if _, err := NewStore(volumeRoot); err == nil {
		t.Fatal("expected a volume root to be rejected")
	}
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not Windows access-control lists")
	}
	sharedRoot := filepath.Join(t.TempDir(), "shared-jobs")
	if err := os.Mkdir(sharedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sharedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(sharedRoot); err == nil {
		t.Fatal("expected shared manifest storage to be rejected")
	}
	info, err := os.Stat(sharedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("store changed caller directory permissions to %o", info.Mode().Perm())
	}
}

func TestManifestRejectsDuplicateSourcesAndDestinations(t *testing.T) {
	items := sameBasenameItems(t)
	items[1].SourcePath = items[0].SourcePath
	if _, err := NewManifest(testIdentity(), items); err == nil {
		t.Fatal("expected duplicate source path to be rejected")
	}
	items = sameBasenameItems(t)
	items[1].VideoDestination = items[0].VideoDestination
	if _, err := NewManifest(testIdentity(), items); err == nil {
		t.Fatal("expected duplicate destination path to be rejected")
	}
}

func TestUpdatedManifestCannotChangeJobIdentity(t *testing.T) {
	manifest := newTestManifest(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{name: "job ID", mutate: func(current *Manifest) { current.JobID = strings.Repeat("b", 32) }},
		{name: "source fingerprint", mutate: func(current *Manifest) {
			current.Items[0].SourceFingerprint.SizeBytes++
		}},
		{name: "output destination", mutate: func(current *Manifest) {
			current.Items[0].VideoDestination += ".stale"
		}},
		{name: "processing identity", mutate: func(current *Manifest) {
			current.Identity.VideoBackend = "software"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := store.Update(manifest.JobID, func(current *Manifest) error {
				test.mutate(current)
				return nil
			}); err == nil {
				t.Fatal("expected immutable job definition mutation to be rejected")
			}
		})
	}
	got, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflectManifestEqual(got, manifest) {
		t.Fatal("rejected update changed the persisted job definition")
	}
}

func TestUpdatedAtIsMonotonic(t *testing.T) {
	manifest := newTestManifest(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := store.Update(manifest.JobID, func(*Manifest) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.Load(manifest.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdatedAt.After(manifest.UpdatedAt) || got.UpdatedAt.IsZero() {
		t.Fatalf("updated timestamp did not advance: %v", got.UpdatedAt)
	}
}

func TestRemoveDeletesOnlyManifestMetadata(t *testing.T) {
	manifest := newTestManifest(t)
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	source := manifest.Items[0].SourcePath
	video := manifest.Items[0].VideoDestination
	if err := os.MkdirAll(filepath.Dir(video), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(video, []byte("completed video"), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(t.TempDir(), "w-000000000001")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	workspaceArtifact := filepath.Join(workspace, "video.mp4")
	if err := os.WriteFile(workspaceArtifact, []byte("staged video"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(manifest.JobID); err != nil {
		t.Fatalf("remove manifest: %v", err)
	}
	if _, err := store.Load(manifest.JobID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manifest still exists after removal: %v", err)
	}
	for _, path := range []string{source, video, workspaceArtifact} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("manifest removal affected media path %q: %v", path, err)
		}
	}
}
