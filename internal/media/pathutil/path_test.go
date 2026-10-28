package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveResolvesExistingSymlinkAncestorsForMissingPath(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("directory symlink is unavailable: %v", err)
	}

	got, err := Resolve(filepath.Join(alias, "not-created", "output.mp4"))
	if err != nil {
		t.Fatalf("resolve path with missing suffix: %v", err)
	}
	want := filepath.Join(target, "not-created", "output.mp4")
	if got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}
