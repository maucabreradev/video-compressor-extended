package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unicode/utf16"
)

func TestCreateUsesShortOpaqueWorkspaceAndKeepsArtifactNamesSafe(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vcx", "work")
	manager := Manager{Roots: []string{root}}
	outputDirectory := t.TempDir()
	for index := 0; index < 24; index++ {
		outputDirectory = filepath.Join(outputDirectory, "long nested folder")
	}

	workspace, err := manager.Create(outputDirectory)
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	t.Cleanup(func() {
		if err := workspace.Remove(); err != nil {
			t.Errorf("remove workspace: %v", err)
		}
	})

	if filepath.Dir(workspace.Path()) != root {
		t.Fatalf("workspace parent = %q, want %q", filepath.Dir(workspace.Path()), root)
	}
	base := filepath.Base(workspace.Path())
	if !strings.HasPrefix(base, "w-") || len(base) > 14 {
		t.Fatalf("workspace name %q is not a short opaque ID", base)
	}
	if strings.Contains(workspace.Path(), filepath.Base(outputDirectory)) {
		t.Fatalf("workspace path contains user-derived output name: %q", workspace.Path())
	}

	artifact, err := workspace.ArtifactPath("video.mp4")
	if err != nil {
		t.Fatalf("build artifact path: %v", err)
	}
	if filepath.Dir(artifact) != workspace.Path() || filepath.Ext(artifact) != ".mp4" {
		t.Fatalf("artifact path = %q; want an MP4 directly in the workspace", artifact)
	}
	if pathUnits(artifact) > defaultMaxPathUTF16Units {
		t.Fatalf("artifact path uses %d UTF-16 units; limit is %d", pathUnits(artifact), defaultMaxPathUTF16Units)
	}
}

func TestDefaultManagerUsesUnicodePerUserCacheRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("LOCALAPPDATA is Windows-specific")
	}

	cache := filepath.Join(t.TempDir(), "Profile café", "AppData", "Local")
	t.Setenv("LOCALAPPDATA", cache)
	workspace, err := (Manager{}).Create(t.TempDir())
	if err != nil {
		t.Fatalf("create workspace under Unicode profile: %v", err)
	}
	defer workspace.Remove()

	wantRoot := filepath.Join(cache, "vcx", "work")
	if filepath.Dir(workspace.Path()) != wantRoot {
		t.Fatalf("workspace parent = %q, want per-user cache root %q", filepath.Dir(workspace.Path()), wantRoot)
	}
}

func TestCreateUsesSameVolumeRootBeforeFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("volume preference is Windows-specific")
	}

	outputDirectory := t.TempDir()
	sameVolumeRoot := filepath.Join(t.TempDir(), "same-volume")
	fallbackVolume := `Z:`
	if strings.EqualFold(filepath.VolumeName(outputDirectory), fallbackVolume) {
		fallbackVolume = `Y:`
	}
	fallbackRoot := filepath.Join(fallbackVolume+`\`, "vcx-fallback")
	manager := Manager{Roots: []string{fallbackRoot, sameVolumeRoot}}

	workspace, err := manager.Create(outputDirectory)
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	defer workspace.Remove()
	if filepath.Dir(workspace.Path()) != sameVolumeRoot {
		t.Fatalf("workspace parent = %q, want same-volume root %q", filepath.Dir(workspace.Path()), sameVolumeRoot)
	}
}

func TestCreateFallsBackWhenPreferredRootIsUnavailable(t *testing.T) {
	parent := t.TempDir()
	blockedRoot := filepath.Join(parent, "not-a-directory")
	if err := os.WriteFile(blockedRoot, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	fallbackRoot := filepath.Join(parent, "fallback")
	manager := Manager{Roots: []string{blockedRoot, fallbackRoot}}

	workspace, err := manager.Create(parent)
	if err != nil {
		t.Fatalf("create workspace using fallback: %v", err)
	}
	defer workspace.Remove()
	if filepath.Dir(workspace.Path()) != fallbackRoot {
		t.Fatalf("workspace parent = %q, want fallback %q", filepath.Dir(workspace.Path()), fallbackRoot)
	}
}

func TestCreateRejectsRootsThatCannotFitToolPathBudget(t *testing.T) {
	root := filepath.Join(t.TempDir(), strings.Repeat("deep", 30))
	manager := Manager{Roots: []string{root}, MaxPathUTF16Units: 80}

	_, err := manager.Create(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "short") {
		t.Fatalf("expected actionable short-workspace error, got %v", err)
	}
	if _, statErr := os.Stat(root); !os.IsNotExist(statErr) {
		t.Fatalf("overlong candidate should not be created, stat error = %v", statErr)
	}
}

func TestCreateRetriesWorkspaceIDCollision(t *testing.T) {
	root := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	const occupiedID = "001122334455"
	if err := os.Mkdir(filepath.Join(root, workspacePrefix+occupiedID), 0o700); err != nil {
		t.Fatal(err)
	}
	ids := []string{occupiedID, "aabbccddeeff"}
	index := 0
	workspace, err := createInRoot(root, defaultMaxPathUTF16Units, func() (string, error) {
		id := ids[index]
		index++
		return id, nil
	})
	if err != nil {
		t.Fatalf("retry workspace ID collision: %v", err)
	}
	defer workspace.Remove()
	if index != 2 || filepath.Base(workspace.Path()) != workspacePrefix+ids[1] {
		t.Fatalf("workspace = %q, generated IDs = %d; want second ID after collision", workspace.Path(), index)
	}
}

func TestArtifactPathRejectsUserPathsAndOverlongNames(t *testing.T) {
	root := filepath.Join(t.TempDir(), "work")
	workspace, err := (Manager{Roots: []string{root}}).Create(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer workspace.Remove()

	for _, name := range []string{"", "..\\original.mp4", "nested/video.mp4", "nested\\video.mp4", "."} {
		t.Run(fmt.Sprintf("unsafe_%q", name), func(t *testing.T) {
			if _, err := workspace.ArtifactPath(name); err == nil {
				t.Fatalf("ArtifactPath(%q) unexpectedly succeeded", name)
			}
		})
	}

	manager := Manager{Roots: []string{root}, MaxPathUTF16Units: pathUnits(workspace.Path()) + 4}
	if _, err := manager.Create(t.TempDir()); err == nil {
		t.Fatal("expected the workspace to be rejected when its artifact path exceeds the configured limit")
	}
}

func TestRemoveDeletesOnlyOwnedWorkspace(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "work")
	source := filepath.Join(parent, "source.mp4")
	final := filepath.Join(parent, "final.mp4")
	for _, path := range []string{source, final} {
		if err := os.WriteFile(path, []byte(path), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	workspace, err := (Manager{Roots: []string{root}}).Create(parent)
	if err != nil {
		t.Fatal(err)
	}
	workspacePath := workspace.Path()
	if err := os.WriteFile(filepath.Join(workspacePath, "audio.wav"), []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Remove(); err != nil {
		t.Fatalf("remove owned workspace: %v", err)
	}
	if err := workspace.Remove(); err != nil {
		t.Fatalf("remove already-removed workspace: %v", err)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists: %v", err)
	}
	for _, path := range []string{source, final} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("workspace cleanup affected external file %q: %v", path, err)
		}
	}
}

func TestConcurrentCreateReturnsDistinctWorkspaces(t *testing.T) {
	manager := Manager{Roots: []string{filepath.Join(t.TempDir(), "work")}}
	const count = 32
	paths := make(chan string, count)
	errs := make(chan error, count)
	var group sync.WaitGroup
	for index := 0; index < count; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			workspace, err := manager.Create(t.TempDir())
			if err != nil {
				errs <- err
				return
			}
			paths <- workspace.Path()
		}()
	}
	group.Wait()
	close(paths)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent workspace creation failed: %v", err)
	}

	seen := make(map[string]struct{}, count)
	for path := range paths {
		if _, exists := seen[path]; exists {
			t.Errorf("workspace path collision: %q", path)
		}
		seen[path] = struct{}{}
		if err := os.RemoveAll(path); err != nil {
			t.Errorf("remove workspace %q: %v", path, err)
		}
	}
	if len(seen) != count {
		t.Fatalf("created %d unique workspaces; want %d", len(seen), count)
	}
}

func pathUnits(path string) int {
	return len(utf16.Encode([]rune(path)))
}
