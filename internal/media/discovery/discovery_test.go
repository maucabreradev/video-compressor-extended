package discovery

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDiscoverRecursesFiltersExtensionsAndExcludesOutputTree(t *testing.T) {
	inputRoot := t.TempDir()
	outputRoot := filepath.Join(inputRoot, "vcx_processed")
	mediaPaths := []string{
		filepath.Join(inputRoot, "z.MP4"),
		filepath.Join(inputRoot, "nested", "clip.MKV"),
		filepath.Join(inputRoot, "nested", "movie.MOV"),
		filepath.Join(inputRoot, "café", "naïve.MP4"),
	}
	for _, path := range mediaPaths {
		writeFile(t, path)
	}
	writeFile(t, filepath.Join(inputRoot, "notes.txt"))
	writeFile(t, filepath.Join(outputRoot, "previous.mp4"))

	got, err := Discover([]string{inputRoot}, outputRoot)
	if err != nil {
		t.Fatalf("discover media: %v", err)
	}
	want := append([]string(nil), mediaPaths...)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected discovered files:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestDiscoverAcceptsExplicitFileAndDeduplicatesInputs(t *testing.T) {
	inputRoot := t.TempDir()
	video := filepath.Join(inputRoot, "clip.webm")
	writeFile(t, video)

	got, err := Discover([]string{inputRoot, video}, filepath.Join(t.TempDir(), "output"))
	if err != nil {
		t.Fatalf("discover explicit media: %v", err)
	}
	if len(got) != 1 || got[0] != video {
		t.Fatalf("expected one deduplicated input %q, got %#v", video, got)
	}
}

func TestDiscoverExcludesOutputTreeResolvedThroughDirectorySymlink(t *testing.T) {
	inputRoot := t.TempDir()
	outputDirectory := filepath.Join(inputRoot, "generated")
	writeFile(t, filepath.Join(outputDirectory, "previous.mp4"))
	outputAlias := filepath.Join(inputRoot, "vcx_processed")
	if err := os.Symlink(outputDirectory, outputAlias); err != nil {
		t.Skipf("directory symlink is unavailable: %v", err)
	}
	writeFile(t, filepath.Join(inputRoot, "source.mkv"))

	got, err := Discover([]string{inputRoot}, outputAlias)
	if err != nil {
		t.Fatalf("discover media with aliased output root: %v", err)
	}
	want := []string{filepath.Join(inputRoot, "source.mkv")}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected discovered files: got %#v, want %#v", got, want)
	}
}

func TestDiscoverRecognizesSupportedExtensionsWithoutCaseSensitivity(t *testing.T) {
	inputRoot := t.TempDir()
	extensions := []string{".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".ts", ".webm", ".wmv"}
	for _, extension := range extensions {
		writeFile(t, filepath.Join(inputRoot, "clip"+strings.ToUpper(extension)))
	}

	got, err := Discover([]string{inputRoot}, filepath.Join(t.TempDir(), "output"))
	if err != nil {
		t.Fatalf("discover supported formats: %v", err)
	}
	if len(got) != len(extensions) {
		t.Fatalf("expected %d supported files, got %d: %#v", len(extensions), len(got), got)
	}
}

func TestDiscoverReportsMissingInputPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := Discover([]string{missing}, filepath.Join(t.TempDir(), "output")); err == nil {
		t.Fatal("expected a missing input path to be reported")
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
