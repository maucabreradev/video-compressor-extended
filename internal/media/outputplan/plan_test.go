package outputplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanPreservesRelativeDirectoriesAndCreatesVideoAndSubtitlePaths(t *testing.T) {
	base := t.TempDir()
	inputRoot := filepath.Join(base, "input")
	outputRoot := filepath.Join(base, "output")
	source := filepath.Join(inputRoot, "nested", "clip.MKV")
	writeFile(t, source)

	plans, err := PlanOutputs(inputRoot, outputRoot, []string{source})
	if err != nil {
		t.Fatalf("plan output paths: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one output plan, got %d", len(plans))
	}
	wantVideo := filepath.Join(outputRoot, "nested", "clip.mp4")
	wantSubtitle := filepath.Join(outputRoot, "nested", "clip.srt")
	if plans[0].Source != source || plans[0].Video != wantVideo || plans[0].Subtitle != wantSubtitle {
		t.Fatalf("unexpected output plan: %+v", plans[0])
	}
}

func TestPlanForExplicitFileRootUsesFileBaseName(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "media café", "scene_naïve.mov")
	writeFile(t, source)
	outputRoot := filepath.Join(base, "processed")

	plans, err := PlanOutputs(source, outputRoot, []string{source})
	if err != nil {
		t.Fatalf("plan explicit file: %v", err)
	}
	if len(plans) != 1 || plans[0].Video != filepath.Join(outputRoot, "scene_naïve.mp4") {
		t.Fatalf("unexpected output plan: %#v", plans)
	}
}

func TestPlanRejectsInputInsideOutputTree(t *testing.T) {
	base := t.TempDir()
	inputRoot := base
	outputRoot := filepath.Join(base, "vcx_processed")
	source := filepath.Join(outputRoot, "previous.mkv")
	writeFile(t, source)

	if _, err := PlanOutputs(inputRoot, outputRoot, []string{source}); err == nil {
		t.Fatal("expected an input inside the output tree to be rejected")
	}
}

func TestPlanRejectsCollidingOutputNames(t *testing.T) {
	inputRoot := t.TempDir()
	first := filepath.Join(inputRoot, "scene.mkv")
	second := filepath.Join(inputRoot, "scene.mov")
	writeFile(t, first)
	writeFile(t, second)

	_, err := PlanOutputs(inputRoot, filepath.Join(t.TempDir(), "output"), []string{first, second})
	if err == nil {
		t.Fatal("expected colliding output names to be rejected")
	}
	for _, source := range []string{first, second} {
		if !strings.Contains(err.Error(), filepath.Base(source)) {
			t.Errorf("collision error does not identify %q: %v", source, err)
		}
	}
}

func TestPlanNeverOverwritesAnInput(t *testing.T) {
	inputRoot := t.TempDir()
	source := filepath.Join(inputRoot, "already.mp4")
	writeFile(t, source)

	_, err := PlanOutputs(inputRoot, inputRoot, []string{source})
	if err == nil {
		t.Fatal("expected output planning to reject overwriting an input")
	}
}

func TestPlanRejectsOutputRootThatAliasesInputRoot(t *testing.T) {
	base := t.TempDir()
	inputRoot := filepath.Join(base, "input")
	writeFile(t, filepath.Join(inputRoot, "clip.mp4"))
	outputRoot := filepath.Join(base, "output-link")
	if err := os.Symlink(inputRoot, outputRoot); err != nil {
		t.Skipf("directory symlink is unavailable: %v", err)
	}

	if _, err := PlanOutputs(inputRoot, outputRoot, []string{filepath.Join(inputRoot, "clip.mp4")}); err == nil {
		t.Fatal("expected output aliasing the input root to be rejected")
	}
}

func TestPlanRejectsOutputPathThatAliasesSelectedInputThroughSymlink(t *testing.T) {
	base := t.TempDir()
	inputRoot := filepath.Join(base, "input")
	mkvInput := filepath.Join(inputRoot, "a", "clip.mkv")
	mp4Input := filepath.Join(inputRoot, "b", "clip.mp4")
	writeFile(t, mkvInput)
	writeFile(t, mp4Input)

	outputRoot := filepath.Join(base, "output")
	linkedOutputDirectory := filepath.Join(outputRoot, "a")
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(inputRoot, "b"), linkedOutputDirectory); err != nil {
		t.Skipf("directory symlink is unavailable: %v", err)
	}

	_, err := PlanOutputs(inputRoot, outputRoot, []string{mkvInput, mp4Input})
	if err == nil {
		t.Fatal("expected output resolving to a selected input to be rejected")
	}
	if !strings.Contains(err.Error(), "would overwrite input") {
		t.Fatalf("expected selected-input collision, got: %v", err)
	}
}

func TestPlanPreservesSelectedInputAliasInSourcePath(t *testing.T) {
	base := t.TempDir()
	inputDirectory := filepath.Join(base, "input")
	writeFile(t, filepath.Join(inputDirectory, "clip.mkv"))
	inputAlias := filepath.Join(base, "input-link")
	if err := os.Symlink(inputDirectory, inputAlias); err != nil {
		t.Skipf("directory symlink is unavailable: %v", err)
	}
	source := filepath.Join(inputAlias, "clip.mkv")

	plans, err := PlanOutputs(inputAlias, filepath.Join(base, "output"), []string{source})
	if err != nil {
		t.Fatalf("plan output through selected directory alias: %v", err)
	}
	if len(plans) != 1 || plans[0].Source != source {
		t.Fatalf("planning rewrote the selected source path: %#v", plans)
	}
}

func TestPlanRejectsInputOutsideRoot(t *testing.T) {
	inputRoot := t.TempDir()
	source := filepath.Join(t.TempDir(), "outside.mp4")
	writeFile(t, source)

	if _, err := PlanOutputs(inputRoot, filepath.Join(t.TempDir(), "output"), []string{source}); err == nil {
		t.Fatal("expected an input outside the root to be rejected")
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
}
