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
	if filepath.Base(got) != "output.mp4" || filepath.Base(filepath.Dir(got)) != "not-created" {
		t.Fatalf("resolved path has an unexpected missing suffix: %q", got)
	}
	resolvedAncestor := filepath.Dir(filepath.Dir(got))
	if same, err := os.Stat(resolvedAncestor); err != nil {
		t.Fatalf("inspect resolved ancestor: %v", err)
	} else if want, err := os.Stat(target); err != nil {
		t.Fatal(err)
	} else if !os.SameFile(same, want) {
		t.Fatalf("resolved ancestor %q does not identify target %q", resolvedAncestor, target)
	}
}
